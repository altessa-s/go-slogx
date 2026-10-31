// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package handlerbase

import (
	"context"
	"log/slog"
	"strings"
)

// Base provides common functionality for slog middleware handlers.
// It should be embedded in concrete handler implementations.
//
// Base is immutable after creation - AttrsBase and GroupBase
// return new Base instances rather than modifying the existing one.
type Base struct {
	inner  slog.Handler
	groups []string
}

// NewBase creates a new Base wrapping the given inner handler.
// Panics if inner is nil.
func NewBase(inner slog.Handler) Base {
	if inner == nil {
		panic("base: inner handler cannot be nil")
	}
	return Base{
		inner:  inner,
		groups: make([]string, 0),
	}
}

// Inner returns the wrapped handler.
func (b *Base) Inner() slog.Handler {
	return b.inner
}

// Groups returns the current group path.
// The returned slice should not be modified.
func (b *Base) Groups() []string {
	return b.groups
}

// GroupPath returns the full path for an attribute key, including all groups.
// For example, if groups is ["user", "profile"] and key is "name",
// returns "user.profile.name".
func (b *Base) GroupPath(key string) string {
	if len(b.groups) == 0 {
		return key
	}

	var sb strings.Builder
	for _, g := range b.groups {
		sb.WriteString(g)
		sb.WriteByte('.')
	}
	sb.WriteString(key)
	return sb.String()
}

// Enabled delegates to the inner handler's Enabled method.
// This method can be called directly or used by embedding handlers.
func (b *Base) Enabled(ctx context.Context, level slog.Level) bool {
	return b.inner.Enabled(ctx, level)
}

// AttrsBase returns a new Base with the inner handler updated via WithAttrs.
// The groups are preserved. Use this in your handler's WithAttrs implementation:
//
//	func (h *MyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
//	    processedAttrs := h.processAttrs(attrs)
//	    return &MyHandler{
//	        Base:  h.AttrsBase(processedAttrs),
//	        field: h.field,
//	    }
//	}
func (b *Base) AttrsBase(attrs []slog.Attr) Base {
	return Base{
		inner:  b.inner.WithAttrs(attrs),
		groups: b.groups,
	}
}

// GroupBase returns a new Base with the group added to the path
// and inner handler updated via WithGroup.
// Use this in your handler's WithGroup implementation:
//
//	func (h *MyHandler) WithGroup(name string) slog.Handler {
//	    if name == "" {
//	        return h
//	    }
//	    return &MyHandler{
//	        Base:  h.GroupBase(name),
//	        field: h.field,
//	    }
//	}
func (b *Base) GroupBase(name string) Base {
	newGroups := make([]string, len(b.groups)+1)
	copy(newGroups, b.groups)
	newGroups[len(b.groups)] = name

	return Base{
		inner:  b.inner.WithGroup(name),
		groups: newGroups,
	}
}
