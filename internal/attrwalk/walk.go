// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package attrwalk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/altessa-s/go-slogx/mask"
)

const maxMaskDescentDepth = 10
const maxContainerBreadth = 256
const breadthLimitPlaceholder = "<masking: max breadth exceeded>"
const depthLimitPlaceholder = "<masking: max depth exceeded>"
const marshalFailurePlaceholder = "<masking: JSON encoding failed>"

// Walker masks resolved attributes and their JSON representation.
// Match must be immutable and safe for concurrent calls.
type Walker struct {
	Match func(string, string) mask.Func
}

// Attr returns a masked attribute without changing the caller's input.
func (h *Walker) Attr(a slog.Attr, groups []string) slog.Attr {
	return h.maskAttributeDepth(a, groups, 0)
}

func (h *Walker) buildFieldPath(groups []string, key string) string {
	if len(groups) == 0 {
		return key
	}

	return strings.Join(groups, ".") + "." + key
}

func (h *Walker) maskAttributeDepth(attr slog.Attr, groups []string, depth int) slog.Attr {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return attr
	}

	if depth >= maxMaskDescentDepth && (attr.Value.Kind() == slog.KindGroup || attr.Value.Kind() == slog.KindAny) {
		return slog.String(attr.Key, depthLimitPlaceholder)
	}

	if attr.Value.Kind() == slog.KindGroup {
		attrs := attr.Value.Group()
		out := make([]slog.Attr, 0, min(len(attrs), maxContainerBreadth)+1)

		children := groups
		if attr.Key != "" {
			children = slices.Concat(groups, []string{attr.Key})
		}

		for _, a := range attrs[:min(len(attrs), maxContainerBreadth)] {
			out = append(out, h.maskAttributeDepth(a, children, depth+1))
		}

		if len(attrs) > maxContainerBreadth {
			out = append(out, slog.String("...", breadthLimitPlaceholder))
		}

		return slog.Attr{Key: attr.Key, Value: slog.GroupValue(out...)}
	}

	if fn := h.Match(attr.Key, h.buildFieldPath(groups, attr.Key)); fn != nil {
		return h.applyMask(attr, fn, depth)
	}

	if attr.Value.Kind() == slog.KindAny {
		tree, failure := jsonTree(attr.Value.Any())
		if failure != "" {
			return slog.String(attr.Key, failure)
		}

		attr.Value = h.walkJSON(tree, slices.Concat(groups, []string{attr.Key}), depth)
		if attr.Key == "" && tree == nil {
			// Preserve JSON null without turning this into an omitted zero Attr.
			attr.Value = slog.AnyValue((*string)(nil))
		}

		if attr.Key == "" && attr.Value.Kind() == slog.KindGroup {
			// An empty-key slog group is inlined, but a JSON object keeps its key.
			attr.Value = slog.AnyValue(jsonValue(attr.Value))
		}
	}

	return attr
}

// jsonTree delegates names, promotion, omission, and custom encoding to JSON.
// Neither errors nor panic payloads are allowed to reach a sink.
func jsonTree(raw any) (tree any, failure string) {
	defer func() {
		if recover() != nil {
			tree, failure = nil, marshalFailurePlaceholder
		}
	}()

	remaining := maxContainerBreadth
	if reason := encodingLimit(reflect.ValueOf(raw), 0, &remaining); reason != "" {
		return nil, reason
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, marshalFailurePlaceholder
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&tree); err != nil {
		return nil, marshalFailurePlaceholder
	}

	return tree, ""
}

// encodingLimit prevents fatal pointer/interface recursion in the pinned JSON
// encoder. It only bounds input; JSON alone chooses the emitted fields.
func encodingLimit(v reflect.Value, depth int, remaining *int) string {
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
			return encodingLimit(v.Elem(), depth+1, remaining)
		}
	case reflect.Struct:
		for fieldType, field := range v.Fields() {
			// Reject unsupported names; never guess the name a JSON version emits.
			name, _, _ := strings.Cut(fieldType.Tag.Get("json"), ",")
			if !validJSONName(name) {
				return marshalFailurePlaceholder
			}

			if reason := encodingLimit(field, depth+1, remaining); reason != "" {
				return reason
			}
		}
	case reflect.Map:
		for it := v.MapRange(); it.Next(); {
			if reason := encodingLimit(it.Key(), depth+1, remaining); reason != "" {
				return reason
			}

			if reason := encodingLimit(it.Value(), depth+1, remaining); reason != "" {
				return reason
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if reason := encodingLimit(v.Index(i), depth+1, remaining); reason != "" {
				return reason
			}
		}
	default:
		// Scalars have no children; JSON rejects unsupported scalar types.
	}

	return ""
}

// validJSONName accepts the encoding/json v1 documented field-name alphabet.
// The pinned encoder truncates some invalid tags rather than ignoring them.
func validJSONName(name string) bool {
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", r) {
			return false
		}
	}

	return true
}

// walkJSON visits only decoded data: user formatting methods cannot run again.
// Array elements retain their JSON shape and the parent matching path.
func (h *Walker) walkJSON(tree any, groups []string, depth int) slog.Value {
	switch v := tree.(type) {
	case map[string]any:
		if depth >= maxMaskDescentDepth {
			return slog.StringValue(depthLimitPlaceholder)
		}

		if len(v) == 0 {
			return slog.AnyValue(v)
		}

		keys := slices.Sorted(maps.Keys(v))

		attrs := make([]slog.Attr, 0, min(len(v), maxContainerBreadth)+1)
		for _, key := range keys[:min(len(keys), maxContainerBreadth)] {
			child := v[key]

			var value slog.Value
			if fn := h.Match(key, h.buildFieldPath(groups, key)); fn != nil {
				value = slog.StringValue(maskTree(fn, child, depth+1))
			} else {
				value = h.walkJSON(child, slices.Concat(groups, []string{key}), depth+1)
			}

			attrs = append(attrs, slog.Attr{Key: key, Value: value})
		}

		if len(v) > maxContainerBreadth {
			attrs = append(attrs, slog.String("...", breadthLimitPlaceholder))
		}

		if _, hasEmptyKey := v[""]; hasEmptyKey {
			// Slog inlines empty-key groups and drops empty-key null attributes.
			// Keep this object as JSON data before GroupValue can discard them.
			out := make(map[string]any, len(attrs))
			for _, attr := range attrs {
				out[attr.Key] = jsonValue(attr.Value)
			}

			return slog.AnyValue(out)
		}

		return slog.GroupValue(attrs...)
	case []any:
		if depth >= maxMaskDescentDepth {
			return slog.StringValue(depthLimitPlaceholder)
		}

		if len(v) == 0 {
			return slog.AnyValue(v)
		}

		out := make([]any, 0, min(len(v), maxContainerBreadth)+1)
		for _, child := range v[:min(len(v), maxContainerBreadth)] {
			out = append(out, jsonValue(h.walkJSON(child, groups, depth+1)))
		}

		if len(v) > maxContainerBreadth {
			out = append(out, breadthLimitPlaceholder)
		}

		return slog.AnyValue(out)
	default:
		return slog.AnyValue(tree)
	}
}

// jsonValue converts sanitized groups back to generic JSON data.
func jsonValue(value slog.Value) any {
	if value.Kind() != slog.KindGroup {
		return value.Any()
	}

	out := make(map[string]any, len(value.Group()))
	for _, attr := range value.Group() {
		out[attr.Key] = jsonValue(attr.Value)
	}

	return out
}

func maskJSON(fn mask.Func, raw any, depth int) string {
	tree, failure := jsonTree(raw)
	if failure != "" {
		return failure
	}

	return maskTree(fn, tree, depth)
}

func maskTree(fn mask.Func, tree any, depth int) string {
	remaining := maxContainerBreadth
	if reason := treeLimit(tree, depth, &remaining); reason != "" {
		return reason
	}

	return invokeMask(fn, fmt.Sprint(tree))
}

// treeLimit bounds conversion of a matched compound value before fmt sees it.
func treeLimit(tree any, depth int, remaining *int) string {
	if depth >= maxMaskDescentDepth {
		return depthLimitPlaceholder
	}

	if *remaining == 0 {
		return breadthLimitPlaceholder
	}

	*remaining--

	switch v := tree.(type) {
	case map[string]any:
		for _, child := range v {
			if reason := treeLimit(child, depth+1, remaining); reason != "" {
				return reason
			}
		}
	case []any:
		for _, child := range v {
			if reason := treeLimit(child, depth+1, remaining); reason != "" {
				return reason
			}
		}
	}

	return ""
}

// applyMask applies the masking function to an attribute value
func (h *Walker) applyMask(attr slog.Attr, mask mask.Func, depth int) slog.Attr {
	switch attr.Value.Kind() {
	case slog.KindString:
		return slog.String(attr.Key, invokeMask(mask, attr.Value.String()))

	case slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool:
		// Convert to string, mask, and keep as string
		str := fmt.Sprint(attr.Value.Any())
		return slog.String(attr.Key, invokeMask(mask, str))

	case slog.KindDuration:
		return slog.String(attr.Key, invokeMask(mask, attr.Value.Duration().String()))

	case slog.KindTime:
		return slog.String(attr.Key, invokeMask(mask, attr.Value.Time().String()))

	case slog.KindAny:
		// Top-level key matched: mask the entire value as a single
		// string. Per-subfield masking for KindAny is handled at the
		// maskAttribute level via walkJSON, which is
		// where nested matches inside structs/maps/slices are found.
		return slog.String(attr.Key, maskJSON(mask, attr.Value.Any(), depth))

	default:
		// For unknown types, convert to string and mask
		str := fmt.Sprint(attr.Value.Any())
		return slog.String(attr.Key, invokeMask(mask, str))
	}
}

// invokeMask contains consumer callbacks and never exposes a panic payload.
func invokeMask(fn mask.Func, value string) (result string) {
	defer func() {
		if recover() != nil {
			result = "********"
		}
	}()

	return fn(value)
}
