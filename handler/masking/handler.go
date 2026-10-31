// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/altessa-s/go-slogx/internal/attrwalk"
	"github.com/altessa-s/go-slogx/internal/handlerbase"
	"github.com/altessa-s/go-slogx/internal/pathcache"
	"github.com/altessa-s/go-slogx/mask"
)

// Handler masks attributes, including captured and nested values.
// Message text is not masked.
type Handler struct {
	handlerbase.Base
	walker *attrwalk.Walker
}
type pattern struct {
	re   *regexp.Regexp
	mask mask.Func
}
type matcher struct {
	fields            map[string]mask.Func
	patterns          []pattern
	nested, sensitive bool
	cache             pathcache.Cache[mask.Func]
}

// New validates every rule before constructing the handler.
func New(next slog.Handler, opts ...Option) (*Handler, error) {
	o := newOptions(opts...)
	if next == nil {
		return nil, errors.New("masking: nil handler")
	}
	if o.defaultMask == nil {
		o.defaultMask = mask.Full()
	}
	if o.defaults {
		o.nested = true
	}
	m := &matcher{fields: make(map[string]mask.Func), nested: o.nested, sensitive: o.caseSensitive}
	var rules []mask.Rule
	if o.defaults {
		rules = []mask.Rule{
			{Field: "password", Mask: mask.Full()}, {Field: "token", Mask: mask.Full()},
			{Field: "secret", Mask: mask.Full()}, {Field: "api_key", Mask: mask.Full()},
			{Field: "authorization", Mask: mask.Full()}, {Field: "credit_card", Mask: mask.CreditCard()},
			{Field: "email", Mask: mask.Email()}, {Field: "phone", Mask: mask.Phone()},
		}
	}
	rules = append(rules, o.rules...)
	if o.defaults {
		rules = append(rules, mask.Rule{Pattern: mask.DefaultCredentialPattern, Mask: mask.Full()})
	}

	var errs []error
	for _, r := range rules {
		if err := r.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("rule %q: %w", r.Pattern, err))
			continue
		}
		f := r.Mask
		if f == nil {
			f = o.defaultMask
		}
		if r.Field != "" {
			key := r.Field
			if !m.sensitive {
				key = strings.ToLower(key)
			}
			m.fields[key] = f
		} else {
			expression := r.Pattern
			if !m.sensitive {
				expression = "(?i)" + expression
			}
			re, err := regexp.Compile(expression)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			m.patterns = append(m.patterns, pattern{re, f})
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return &Handler{Base: handlerbase.NewBase(next)}, nil
	}
	walker := &attrwalk.Walker{Match: m.match}
	if len(m.patterns) == 0 {
		walker.Candidate = m.candidate
	}
	return &Handler{Base: handlerbase.NewBase(next), walker: walker}, nil
}
func (m *matcher) candidate(name string) bool {
	if !m.sensitive {
		name = strings.ToLower(name)
	}
	for field := range m.fields {
		if field == name || strings.HasSuffix(field, "."+name) {
			return true
		}
	}
	return false
}
func (m *matcher) match(name, path string) mask.Func {
	if !m.nested {
		path = name
	}
	if !m.sensitive {
		name = strings.ToLower(name)
		path = strings.ToLower(path)
	}
	// Length-prefix the name so arbitrary dotted keys and separators cannot
	// collide with a different (name, path) pair.
	key := strconv.Itoa(len(name)) + ":" + name + path
	if f, ok := m.cache.Get(key); ok {
		return f
	}
	f := m.fields[path]
	if f == nil {
		f = m.fields[name]
	}
	if f == nil {
		for _, p := range m.patterns {
			if p.re.MatchString(path) {
				f = p.mask
				break
			}
		}
	}
	m.cache.Put(key, f)
	return f
}

// Handle masks a copy of the record's attributes.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if h.walker == nil {
		return h.Inner().Handle(ctx, r)
	}
	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool { out.AddAttrs(h.walker.Attr(a, h.Groups())); return true })
	return h.Inner().Handle(ctx, out)
}

// WithAttrs resolves and masks captured attributes on the caller goroutine.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.walker == nil {
		return &Handler{Base: h.AttrsBase(attrs)}
	}
	out := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		out[i] = h.walker.Attr(a, h.Groups())
	}
	return &Handler{Base: h.AttrsBase(out), walker: h.walker}
}

// WithGroup returns a derived handler or the receiver for an empty name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &Handler{Base: h.GroupBase(name), walker: h.walker}
}
