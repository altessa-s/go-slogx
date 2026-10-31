// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx

import (
	"context"
	"iter"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
)

// Field is a key-value pair for structured log fields.
type Field struct {
	Key   string
	Value any
}

// Fields is an ordered collection of [Field] values.
// It provides methods for manipulation and conversion to [slog.Attr].
type Fields []Field

// Append appends a field to the fields.
func (f Fields) Append(key string, value any) Fields {
	return append(f, Field{Key: key, Value: value})
}

// AppendFields appends fields to the fields.
func (f Fields) AppendFields(fields Fields) Fields {
	return append(f, fields...)
}

// Delete removes the first field with the given key. Like [slices.Delete], it
// shifts elements within the receiver's backing array and zeroes the vacated
// tail slot (so the removed value is not retained); use the returned slice.
func (f Fields) Delete(key string) Fields {
	for i := range f {
		if f[i].Key == key {
			return slices.Delete(f, i, i+1)
		}
	}
	return f
}

// Unique returns a new Fields with unique keys.
// If there are duplicate keys, the first occurrence is kept.
func (f Fields) Unique() Fields {
	seen := make(map[string]bool, len(f))
	out := make(Fields, 0, len(f))
	for _, field := range f {
		if !seen[field.Key] {
			seen[field.Key] = true
			out = append(out, field)
		}
	}
	return out
}

// All returns an iterator over all fields as key-value pairs.
// This allows using Fields with Go 1.23+ range over function syntax.
//
// Example:
//
//	for key, value := range fields.All() {
//	    fmt.Printf("%s: %v\n", key, value)
//	}
func (f Fields) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for _, field := range f {
			if !yield(field.Key, field.Value) {
				return
			}
		}
	}
}

// Args converts Fields to slog key-value args for Logger.With().
//
// Example:
//
//	logger.With(fields.Args()...)
func (f Fields) Args() []any {
	out := make([]any, 0, len(f)*2)
	for _, field := range f {
		out = append(out, field.Key, field.Value)
	}
	return out
}

// Attrs converts Fields to slog.Attr slice for LogAttrs().
func (f Fields) Attrs() []slog.Attr {
	attrs := make([]slog.Attr, 0, len(f))
	for _, field := range f {
		attrs = append(attrs, slog.Any(field.Key, field.Value))
	}
	return attrs
}

// fieldsWrapper wraps Fields with a mutex for thread-safe access.
type fieldsWrapper struct {
	mu     sync.RWMutex
	fields Fields
}

type fieldsContextKey struct{}

// FieldsFromContext returns a copy of the [Fields] stored via [InjectFields].
// Returns nil if no fields have been injected.
// Safe for concurrent use.
func FieldsFromContext(ctx context.Context) Fields {
	if ctx == nil {
		return nil
	}
	fw, ok := ctx.Value(fieldsContextKey{}).(*fieldsWrapper)
	if !ok || fw == nil {
		return nil
	}

	fw.mu.RLock()
	n := make(Fields, len(fw.fields))
	copy(n, fw.fields)
	fw.mu.RUnlock()

	return n
}

// InjectFields stores fields in the context for later retrieval via [FieldsFromContext].
// Subsequent calls to [AppendField] and [AppendFields] mutate the stored fields.
//
// The wrapper is allocated fresh per context and deliberately not pooled: it is
// request-scoped and lives as long as the context, so returning it to a sync.Pool
// while the context still references it would let a later request observe or
// overwrite another request's fields (a cross-request data leak). The previous
// pool was also never returned, so it provided no reuse regardless.
func InjectFields(ctx context.Context, f Fields) context.Context {
	fw := &fieldsWrapper{fields: append(make(Fields, 0, len(f)), f...)}
	return context.WithValue(ctx, fieldsContextKey{}, fw)
}

// AppendField appends a field to the [Fields] previously stored via [InjectFields].
// No-op if no fields have been injected. Safe for concurrent use.
func AppendField(ctx context.Context, key string, value any) {
	if ctx == nil {
		return
	}
	fw, ok := ctx.Value(fieldsContextKey{}).(*fieldsWrapper)
	if !ok || fw == nil {
		return
	}
	fw.mu.Lock()
	fw.fields = fw.fields.Append(key, value)
	fw.mu.Unlock()
}

// AppendFields appends multiple fields to the [Fields] previously stored via [InjectFields].
// No-op if no fields have been injected. Safe for concurrent use.
func AppendFields(ctx context.Context, f Fields) {
	if ctx == nil {
		return
	}
	fw, ok := ctx.Value(fieldsContextKey{}).(*fieldsWrapper)
	if !ok || fw == nil {
		return
	}
	fw.mu.Lock()
	fw.fields = fw.fields.AppendFields(f)
	fw.mu.Unlock()
}

// Grouped converts logging fields to slog attributes with safe grouping
// by dot-separated keys. Order is deterministic (keys are sorted at each level).
func (f Fields) Grouped() []slog.Attr {
	if len(f) == 0 {
		return nil
	}

	root := make(fieldGroup, len(f))

fieldsLoop:
	for _, field := range f {
		key := field.Key
		idx := strings.IndexByte(key, '.')
		if idx <= 0 {
			// keep first value if duplicate key appears
			if _, exists := root[key]; !exists {
				root[key] = field.Value
			}
			continue
		}

		// Walk the dot-separated segments in place (no strings.Split allocation).
		current := root
		rest := key
		for {
			i := strings.IndexByte(rest, '.')
			if i < 0 {
				break
			}
			part := rest[:i]
			rest = rest[i+1:]
			next, ok := current[part]
			if !ok {
				child := make(fieldGroup)
				current[part] = child
				current = child
				continue
			}
			child, ok := next.(fieldGroup)
			if !ok {
				// The first value owns the path, including scalar/group conflicts.
				continue fieldsLoop
			}
			current = child
		}

		if _, exists := current[rest]; !exists {
			current[rest] = field.Value
		}
	}

	return mapToAttrs(root)
}

type fieldGroup map[string]any

func mapToAttrs(data fieldGroup) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(data))
	for _, k := range slices.Sorted(maps.Keys(data)) {
		if child, ok := data[k].(fieldGroup); ok {
			attrs = append(attrs, slog.GroupAttrs(k, mapToAttrs(child)...))
			continue
		}
		attrs = append(attrs, slog.Any(k, data[k]))
	}
	return attrs
}
