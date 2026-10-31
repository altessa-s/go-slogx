// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package attrwalk

import (
	"fmt"
	"reflect"

	"github.com/altessa-s/go-slogx/mask"
)

// maskAny checks the entire representation before fmt can recurse into it.
// Cycles exhaust the depth budget. A total node budget also bounds branching
// containers, including private fields and map keys that fmt can print.
// Consumer formatting methods remain the consumer's responsibility.
func maskAny(fn mask.Func, value any, depth int) string {
	remaining := maxContainerBreadth
	if reason := formatLimit(reflect.ValueOf(value), depth, &remaining); reason != "" {
		return reason
	}
	return invokeMask(fn, fmt.Sprint(value))
}

func formatLimit(v reflect.Value, depth int, remaining *int) string {
	if !v.IsValid() {
		return ""
	}
	if depth >= maxMaskDescentDepth {
		return depthLimitPlaceholder
	}
	if *remaining == 0 {
		return breadthLimitPlaceholder
	}
	*remaining--
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return formatLimit(v.Elem(), depth+1, remaining)
		}
	case reflect.Map:
		for iter := v.MapRange(); iter.Next(); {
			if reason := formatLimit(iter.Key(), depth+1, remaining); reason != "" {
				return reason
			}
			if reason := formatLimit(iter.Value(), depth+1, remaining); reason != "" {
				return reason
			}
		}
	case reflect.Struct:
		for _, field := range v.Fields() {
			if reason := formatLimit(field, depth+1, remaining); reason != "" {
				return reason
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if reason := formatLimit(v.Index(i), depth+1, remaining); reason != "" {
				return reason
			}
		}
	default:
		// Scalar values have no children for fmt to traverse.
	}
	return ""
}
