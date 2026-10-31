// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package leveled

import (
	"context"
	"errors"
	"log/slog"
	"maps"

	"github.com/altessa-s/go-slogx"
	"github.com/altessa-s/go-slogx/internal/handlerbase"
)

// Handler filters records using a live default and static subsystem overrides.
type Handler struct {
	handlerbase.Base
	opts      *options
	subsystem string
	captured  bool
}

// New constructs a handler. A nil next handler returns an error.
func New(next slog.Handler, opts ...Option) (*Handler, error) {
	if next == nil {
		return nil, errors.New("leveled: nil handler")
	}
	o := newOptions(opts...)
	if o.defaultLevel == nil {
		o.defaultLevel = DefaultLevel
	}
	o.subsystemLevels = maps.Clone(o.subsystemLevels)
	return &Handler{Base: handlerbase.NewBase(next), opts: o}, nil
}

type minimum struct{ opts *options }

func (m minimum) Level() slog.Level {
	level := m.opts.defaultLevel.Level()
	if level == slogx.LevelOff {
		return level
	}
	for _, v := range m.opts.subsystemLevels {
		level = min(level, v)
	}
	return level
}

// MinLevel returns a live floor for the downstream sink's Enabled method.
func (h *Handler) MinLevel() slog.Leveler { return minimum{h.opts} }
func (h *Handler) threshold(name string) slog.Level {
	level := h.opts.defaultLevel.Level()
	if level == slogx.LevelOff {
		return level
	}
	if v, ok := h.opts.subsystemLevels[name]; ok {
		return v
	}
	return level
}
func allowed(level, threshold slog.Level) bool {
	return threshold != slogx.LevelOff && level >= threshold
}

// Enabled respects both the live threshold and downstream admission.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	threshold := h.MinLevel().Level()
	if h.captured {
		threshold = h.threshold(h.subsystem)
	}
	return allowed(level, threshold) && h.Inner().Enabled(ctx, level)
}

// Handle selects the first top-level subsystem and filters the record.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	name := h.subsystem
	if !h.captured && len(h.Groups()) == 0 {
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == h.opts.subsystemKey && a.Value.Kind() == slog.KindString {
				name = a.Value.String()
				return false
			}
			return true
		})
	}
	if !allowed(r.Level, h.threshold(name)) {
		return nil
	}
	return h.Inner().Handle(ctx, r)
}

// WithAttrs captures only ungrouped subsystem attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.Base = h.WithAttrsBase(attrs)
	if !c.captured && len(h.Groups()) == 0 {
		for _, a := range attrs {
			if a.Key == h.opts.subsystemKey && a.Value.Kind() == slog.KindString {
				c.subsystem = a.Value.String()
				c.captured = true
				break
			}
		}
	}
	return &c
}

// WithGroup opens a group without changing the captured subsystem.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.Base = h.WithGroupBase(name)
	return &c
}
