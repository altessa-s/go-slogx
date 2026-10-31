// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package attrwalk

import (
	"reflect"
)

// typeCanMatch reports whether a value of type t can transitively expose a
// field name (or dynamic map key) matching a configured mask field. Verdicts
// are memoized per type; the type universe of logged values is bounded by
// the program's code, so the cache cannot grow unbounded.
func (h *Walker) typeCanMatch(t reflect.Type) bool {
	if v, ok := h.typeVerdicts.Load(t); ok {
		return v.(bool) //nolint:errcheck // only bools are stored
	}
	verdict := h.computeTypeCanMatch(t, make(map[reflect.Type]bool))
	h.typeVerdicts.Store(t, verdict)
	return verdict
}

// computeTypeCanMatch is the uncached DFS behind [Handler.typeCanMatch]. It
// mirrors walkAny's descent rules: pointers unwrap, atomic types stop,
// structs/slices recurse, and maps either expose dynamic keys or require
// fail-closed replacement (conservatively a match). Interfaces hide the dynamic
// type, so they are conservatively a match too. The visiting set breaks
// recursive types: revisiting a type cannot add names beyond its first pass.
func (h *Walker) computeTypeCanMatch(t reflect.Type, visiting map[reflect.Type]bool) bool {
	for depth := 0; ; depth++ {
		if depth >= maxMaskDescentDepth {
			return true
		}
		if t.Implements(logValuerType) {
			return true
		}
		if t.Kind() != reflect.Pointer {
			break
		}
		t = t.Elem()
	}
	if t.Kind() == reflect.Interface {
		return true
	}
	if isAtomicType(t) {
		return false
	}
	if visiting[t] {
		return false
	}

	switch t.Kind() {
	case reflect.Struct:
		visiting[t] = true
		for ft := range t.Fields() {
			if !ft.IsExported() {
				if ft.Anonymous {
					return true
				}
				continue
			}
			name := structFieldName(ft)
			if name == "" { // Omission requires rebuilding for text sinks too.
				return true
			}
			if h.fieldNameCanMatch(name) || h.computeTypeCanMatch(ft.Type, visiting) {
				return true
			}
		}
		return false
	case reflect.Map:
		// Dynamic string keys may match; other key types must fail closed.
		return true
	case reflect.Slice, reflect.Array:
		return h.computeTypeCanMatch(t.Elem(), visiting)
	default:
		return false
	}
}

// fieldNameCanMatch reports whether a walked field name can satisfy either
// lookup in getMaskForField: the exact-name check (key == name) or the
// dotted-path check — a path's last segment is always the field name, so a
// configured key can only match when it ends with "."+name.
func (h *Walker) fieldNameCanMatch(name string) bool { return h.Candidate(name) }
