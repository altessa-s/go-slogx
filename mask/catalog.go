// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"regexp"
	"slices"
	"unicode/utf8"
)

// Factory creates a configured mask.
type Factory func(Params) (Func, error)

// Catalog is an immutable collection of named mask factories.
type Catalog struct{ factories map[string]Factory }

// NewCatalog copies custom entries and installs the built-in masks by default.
func NewCatalog(opts ...Option) (*Catalog, error) {
	o := newOptions(opts...)
	for name, f := range o.masks {
		if name == "" || f == nil {
			return nil, ErrParams
		}
	}
	for name, f := range o.factories {
		if name == "" || f == nil {
			return nil, ErrParams
		}
	}
	c := &Catalog{factories: make(map[string]Factory)}
	if o.builtins {
		for name, f := range map[string]Func{"full": Full(), "email": Email(), "phone": Phone(), "credit_card": CreditCard(), "url": URL()} {
			c.factories[name] = func(Params) (Func, error) { return f, nil }
		}
		c.factories["fixed"] = func(p Params) (Func, error) {
			s, err := p.String("value", "")
			if err != nil || s == "" {
				return nil, ErrParams
			}
			return Fixed(s), nil
		}
		c.factories["pattern"] = c.pattern
		c.factories["partial"] = func(p Params) (Func, error) {
			first, e1 := p.Int("showFirst", 0)
			last, e2 := p.Int("showLast", 0)
			s, e3 := p.String("maskChar", "*")
			if err := errors.Join(e1, e2, e3); err != nil {
				return nil, err
			}
			if utf8.RuneCountInString(s) != 1 {
				return nil, ErrParams
			}
			r, _ := utf8.DecodeRuneInString(s)
			return Partial(first, last, r)
		}
	}
	for name, f := range o.masks {
		c.factories[name] = func(Params) (Func, error) {
			return f, nil
		}
	}
	maps.Copy(c.factories, o.factories)
	return c, nil
}

// Resolve creates a named mask; errors preserve ErrUnknownMask or ErrParams.
func (c *Catalog) Resolve(name string, p Params) (Func, error) {
	f, ok := c.factories[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownMask, name)
	}
	result, err := f(p)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParams, err)
	}
	if result == nil {
		return nil, ErrParams
	}
	return result, nil
}

// Names iterates catalog names in stable lexical order.
func (c *Catalog) Names() iter.Seq[string] {
	return slices.Values(slices.Sorted(maps.Keys(c.factories)))
}

func (c *Catalog) pattern(p Params) (Func, error) {
	expression, err := p.String("pattern", "")
	if err != nil || expression == "" {
		return nil, ErrParams
	}
	re, err := regexp.Compile(expression)
	if err != nil {
		return nil, err
	}
	if _, ok := p["replacement"]; ok {
		replacement, parseErr := p.String("replacement", "")
		if parseErr != nil {
			return nil, parseErr
		}
		return func(s string) string { return re.ReplaceAllString(s, replacement) }, nil
	}
	kind, err := p.String("replacementType", "fixed")
	if err != nil {
		return nil, err
	}
	var replacement Func
	if kind == "fixed" {
		value, parseErr := p.String("replacementValue", "***")
		if parseErr != nil {
			return nil, parseErr
		}
		replacement = Fixed(value)
	} else {
		replacement, err = c.Resolve(kind, nil)
		if err != nil {
			return nil, err
		}
	}
	return func(s string) (result string) {
		defer func() {
			if recover() != nil {
				result = panicPlaceholder
			}
		}()
		if re.MatchString(s) {
			return replacement(s)
		}
		return s
	}, nil
}
