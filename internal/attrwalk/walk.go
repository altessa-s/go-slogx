// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package attrwalk

import (
	"encoding"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/altessa-s/go-slogx/mask"
)

const maxMaskDescentDepth = 10
const maxContainerBreadth = 256
const breadthLimitPlaceholder = "<masking: max breadth exceeded>"
const depthLimitPlaceholder = "<masking: max depth exceeded>"

// Walker resolves attributes and masks nested structures within a depth budget.
// Match and Candidate must be immutable and safe for concurrent calls.
type Walker struct {
	Match        func(string, string) mask.Func
	Candidate    func(string) bool
	typeVerdicts sync.Map
}

// Attr returns a masked attribute without changing the caller's input.
func (h *Walker) Attr(a slog.Attr, groups []string) slog.Attr { return h.maskAttribute(a, groups) }
func (h *Walker) buildFieldPath(groups []string, key string) string {
	if len(groups) == 0 {
		return key
	}
	return strings.Join(groups, ".") + "." + key
}

// maskAttribute recursively masks an attribute based on configuration
func (h *Walker) maskAttribute(attr slog.Attr, groups []string) slog.Attr {
	return h.maskAttributeDepth(attr, groups, 0)
}

// maskAttributeDepth shares one depth budget across resolved groups and reflection.
func (h *Walker) maskAttributeDepth(attr slog.Attr, groups []string, depth int) slog.Attr {
	// Skip empty attributes
	if attr.Equal(slog.Attr{}) {
		return attr
	}

	// Resolve before inspecting the kind or applying a mask. Resolve bounds
	// direct LogValuer chains and recovers panics; depth also bounds values
	// that return groups or containers containing themselves.
	if attr.Value.Kind() == slog.KindLogValuer {
		attr.Value = attr.Value.Resolve()
	}
	if depth >= maxMaskDescentDepth && (attr.Value.Kind() == slog.KindGroup ||
		(attr.Value.Kind() == slog.KindAny && hasMaskableSubstructure(reflect.ValueOf(attr.Value.Any())))) {
		return slog.String(attr.Key, depthLimitPlaceholder)
	}

	// Handle groups recursively. Group keys themselves are never
	// mask-checked (only members are), so the field path is built after
	// this branch. slices.Concat gives the members a fresh backing array —
	// a plain append could let sibling subtrees share and overwrite the
	// same storage mid-walk.
	if attr.Value.Kind() == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		maskedGroup := make([]slog.Attr, 0, min(len(groupAttrs), maxContainerBreadth)+1)
		childGroups := groups
		if attr.Key != "" {
			childGroups = slices.Concat(groups, []string{attr.Key})
		}

		for _, ga := range groupAttrs[:min(len(groupAttrs), maxContainerBreadth)] {
			maskedGroup = append(maskedGroup, h.maskAttributeDepth(ga, childGroups, depth+1))
		}

		if len(groupAttrs) > maxContainerBreadth {
			maskedGroup = append(maskedGroup, slog.String("...", breadthLimitPlaceholder))
		}
		return slog.Attr{Key: attr.Key, Value: slog.GroupValue(maskedGroup...)}
	}

	// Check if field should be masked
	fieldPath := h.buildFieldPath(groups, attr.Key)
	if mask := h.Match(attr.Key, fieldPath); mask != nil {
		return h.applyMask(attr, mask, depth)
	}

	// Top-level field name didn't match — but a KindAny value can hide
	// a struct/map/slice with sensitive subfields (`payload.password`,
	// `request.token`). Descend via reflection and rebuild the value
	// as slog.GroupValue with matched leaves masked. Without this the
	// original implementation flattened the value via fmt.Sprint and
	// every nested secret leaked verbatim.
	if attr.Value.Kind() == slog.KindAny {
		if masked, ok := h.descendIntoAny(attr.Key, attr.Value, groups, depth); ok {
			return masked
		}
	}

	return attr
}

// descendIntoAny walks a KindAny attribute via reflection and returns a
// new attribute with sensitive subfields masked. The bool result is
// false when no leaf changed or descent is unnecessary (nil or atomic values).
// The caller keeps the original attribute in that case.
func (h *Walker) descendIntoAny(key string, value slog.Value, groups []string, depth int) (slog.Attr, bool) {
	raw := value.Any()
	if raw == nil {
		return slog.Attr{}, false
	}
	// Fields-only configurations can prove statically that a type's
	// transitive field names never match a mask — skip the reflection walk
	// and let the value render natively, exactly like the no-masking fast
	// path in Handle.
	if h.Candidate != nil && !h.typeCanMatch(reflect.TypeOf(raw)) {
		return slog.Attr{}, false
	}
	walked, walkedOK := h.walkAny(reflect.ValueOf(raw), slices.Concat(groups, []string{key}), depth)
	if !walkedOK {
		return slog.Attr{}, false
	}
	return slog.Attr{Key: key, Value: walked}, true
}

// walkAny is the reflect-driven recursive walker used by descendIntoAny.
// Returns (slog.Value, true) when descent rebuilt the value, or
// (zero, false) when the value is unchanged (the caller keeps the original).
// Depth and per-container breadth are bounded.
func (h *Walker) walkAny(rv reflect.Value, groups []string, depth int) (slog.Value, bool) {
	if !rv.IsValid() || depth >= maxMaskDescentDepth {
		return slog.Value{}, false
	}

	if value, ok := reflectedLogValue(rv); ok {
		return h.maskResolvedValue(value.Resolve(), groups, depth+1), true
	}

	// Unwrap pointers/interfaces. nil pointer/interface short-circuits.
	for steps := 0; rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface; steps++ {
		if steps >= maxMaskDescentDepth {
			return slog.StringValue(depthLimitPlaceholder), true
		}
		if rv.IsNil() {
			return slog.Value{}, false
		}
		rv = rv.Elem()
	}

	// Atomic types: well-known stdlib types that have their own canonical
	// representation and MUST NOT be flattened into a slog group. Adding
	// to this list is the migration path when new types surface in audits.
	if isAtomicType(rv.Type()) {
		return slog.Value{}, false
	}

	switch rv.Kind() {
	case reflect.Struct:
		return h.walkStruct(rv, groups, depth)
	case reflect.Map:
		return h.walkMap(rv, groups, depth)
	case reflect.Slice, reflect.Array:
		return h.walkSlice(rv, groups, depth)
	default:
		// Scalars, pointers, channels, funcs, interfaces, and unsafe
		// pointers do not have walkable substructure. Returning
		// (zero, false) signals the caller to pass the value through
		// to the underlying handler unchanged.
		return slog.Value{}, false
	}
}

// reflectedLogValue preserves pointer receivers before reflection unwraps them.
func reflectedLogValue(rv reflect.Value) (slog.Value, bool) {
	for depth := 0; rv.IsValid(); depth++ {
		if depth >= maxMaskDescentDepth {
			return slog.StringValue(depthLimitPlaceholder), true
		}
		if rv.CanInterface() && rv.Type().Implements(logValuerType) {
			return slog.AnyValue(rv.Interface()), true
		}
		if rv.Kind() != reflect.Pointer && rv.Kind() != reflect.Interface {
			break
		}
		if rv.IsNil() {
			break
		}
		rv = rv.Elem()
	}
	return slog.Value{}, false
}

// maskResolvedValue walks the representation of a nested LogValuer at its
// existing field path, without adding a synthetic key to that path.
func (h *Walker) maskResolvedValue(value slog.Value, groups []string, depth int) slog.Value {
	if depth >= maxMaskDescentDepth {
		return slog.StringValue(depthLimitPlaceholder)
	}
	switch value.Kind() {
	case slog.KindGroup:
		attrs := value.Group()
		masked := make([]slog.Attr, 0, min(len(attrs), maxContainerBreadth)+1)
		for _, attr := range attrs[:min(len(attrs), maxContainerBreadth)] {
			masked = append(masked, h.maskAttributeDepth(attr, groups, depth+1))
		}
		if len(attrs) > maxContainerBreadth {
			masked = append(masked, slog.String("...", breadthLimitPlaceholder))
		}
		return slog.GroupValue(masked...)
	case slog.KindAny:
		if nested, ok := h.walkAny(reflect.ValueOf(value.Any()), groups, depth); ok {
			return nested
		}
	default:
		// Resolved scalar values have no nested fields to mask.
	}
	return value
}

// hasMaskableSubstructure reports whether rv could still contain a field a
// masking rule would match — that is, whether walkAny would have descended into
// it had depth allowed. It mirrors walkAny's dispatch deliberately: the two
// must agree about what "walkable" means, or the depth cutoff would either
// print a value it should have hidden or hide a scalar it should have printed.
func hasMaskableSubstructure(rv reflect.Value) bool {
	if !rv.IsValid() {
		return false
	}
	if _, ok := reflectedLogValue(rv); ok {
		return true
	}
	for steps := 0; rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface; steps++ {
		if steps >= maxMaskDescentDepth {
			return true
		}
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	if isAtomicType(rv.Type()) {
		return false
	}
	switch rv.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

// walkStruct walks the exported fields of a struct and rebuilds the
// value as a slog group with masked fields where rules match.
func (h *Walker) walkStruct(rv reflect.Value, groups []string, depth int) (slog.Value, bool) {
	t := rv.Type()
	attrs := make([]slog.Attr, 0, min(rv.NumField(), maxContainerBreadth)+1)
	changed := false
	for i := range min(rv.NumField(), maxContainerBreadth) {
		ft := t.Field(i)
		if !ft.IsExported() {
			if ft.Anonymous {
				return slog.StringValue("<masking: unexported embedding>"), true
			}
			continue
		}
		name := structFieldName(ft)
		if name == "" { // json:"-" must also be omitted from native text output.
			changed = true
			continue
		}
		a, masked := h.attrForField(name, rv.Field(i), groups, depth)
		changed = changed || masked
		attrs = append(attrs, a)
	}
	// No exported fields: nothing to mask or rebuild. An empty group would
	// replace the value, and slog drops empty groups — so the attribute would
	// vanish. Signal "not rebuilt" so the caller keeps the original and lets the
	// downstream handler render it canonically (e.g. an error via Error()).
	if rv.NumField() > maxContainerBreadth {
		attrs = append(attrs, slog.String("...", breadthLimitPlaceholder))
		changed = true
	}
	if !changed {
		return slog.Value{}, false
	}
	return slog.GroupValue(attrs...), true
}

// walkMap walks string-keyed maps and fails closed for other key types.
// Their values can contain secrets, and coercing arbitrary keys is lossy.
func (h *Walker) walkMap(rv reflect.Value, groups []string, depth int) (slog.Value, bool) {
	if rv.Type().Key().Kind() != reflect.String {
		return slog.StringValue("<masking: unsupported map keys>"), true
	}
	attrs := make([]slog.Attr, 0, min(rv.Len(), maxContainerBreadth)+1)
	changed := false
	for iter := rv.MapRange(); len(attrs) < maxContainerBreadth && iter.Next(); {
		key := iter.Key().String()
		a, masked := h.attrForField(key, iter.Value(), groups, depth)
		changed = changed || masked
		attrs = append(attrs, a)
	}
	// Same empty-group hazard as walkStruct: an empty map rebuilds to an empty
	// group, which slog omits — the attribute would vanish. Signal "not rebuilt"
	// so the caller keeps the original and it renders canonically (e.g. map[]).
	if rv.Len() > maxContainerBreadth {
		attrs = append(attrs, slog.String("...", breadthLimitPlaceholder))
		changed = true
	}
	if !changed {
		return slog.Value{}, false
	}
	return slog.GroupValue(attrs...), true
}

// walkSlice walks slice/array elements; sensitive subfields inside
// element structs/maps are still caught because the parent group path
// is preserved. The index is used as the attr key so consumers can map
// each masked leaf back to its original position.
func (h *Walker) walkSlice(rv reflect.Value, groups []string, depth int) (slog.Value, bool) {
	attrs := make([]slog.Attr, 0, min(rv.Len(), maxContainerBreadth)+1)
	changed := false
	for i := range min(rv.Len(), maxContainerBreadth) {
		elem := rv.Index(i)
		key := sliceIndexKey(i)
		if nested, ok := h.walkAny(elem, groups, depth+1); ok {
			changed = true
			attrs = append(attrs, slog.Attr{Key: key, Value: nested})
			continue
		}
		// Same split as attrForField: an atomic element is safe to pass
		// through, one the depth limit cut off is not.
		if depth+1 >= maxMaskDescentDepth && hasMaskableSubstructure(elem) {
			changed = true
			attrs = append(attrs, slog.String(key, depthLimitPlaceholder))
			continue
		}
		// Atomic element — pass through as-is via slog.AnyValue.
		attrs = append(attrs, slog.Attr{Key: key, Value: slog.AnyValue(elem.Interface())})
	}
	// Same empty-group hazard as walkStruct: an empty slice/array rebuilds to an
	// empty group, which slog omits — the attribute would vanish. Signal "not
	// rebuilt" so the caller keeps the original and it renders canonically (e.g. []).
	if rv.Len() > maxContainerBreadth {
		attrs = append(attrs, slog.String("...", breadthLimitPlaceholder))
		changed = true
	}
	if !changed {
		return slog.Value{}, false
	}
	return slog.GroupValue(attrs...), true
}

// sliceIndexKeys pre-renders the "[i]" keys walkSlice assigns to elements so
// the per-element hot path avoids fmt.Sprintf. 64 covers realistic log
// payloads; larger indexes fall back to allocating.
var sliceIndexKeys = func() (keys [64]string) {
	for i := range keys {
		keys[i] = "[" + strconv.Itoa(i) + "]"
	}
	return keys
}()

func sliceIndexKey(i int) string {
	if i < len(sliceIndexKeys) {
		return sliceIndexKeys[i]
	}
	return "[" + strconv.Itoa(i) + "]"
}

// attrForField checks whether the supplied field matches a mask rule
// and, if so, applies it to the stringified value. Otherwise it
// recurses via walkAny (or passes the leaf through unchanged).
func (h *Walker) attrForField(name string, field reflect.Value, parentGroups []string, depth int) (slog.Attr, bool) {
	if value, ok := reflectedLogValue(field); ok {
		return h.maskAttributeDepth(slog.Attr{Key: name, Value: value}, parentGroups, depth+1), true
	}
	fieldPath := h.buildFieldPath(parentGroups, name)
	if mask := h.Match(name, fieldPath); mask != nil {
		// Match: mask the stringified value. Use Interface() so types
		// with custom Stringer get their own representation before
		// masking — keeps masked output consistent with downstream
		// log formatting.
		var v any
		if field.CanInterface() {
			v = field.Interface()
		}
		return slog.String(name, maskAny(mask, v, depth+1)), true
	}
	// slices.Concat allocates a fresh backing array — recursive walks
	// otherwise share parentGroups' storage and a sibling branch can
	// overwrite an earlier child's path mid-walk.
	childGroups := slices.Concat(parentGroups, []string{name})
	if nested, ok := h.walkAny(field, childGroups, depth+1); ok {
		return slog.Attr{Key: name, Value: nested}, true
	}
	// walkAny reports "not rebuilt" for two different reasons, and only one of
	// them is safe to pass through: an atomic leaf has nothing maskable inside,
	// while a value the depth limit cut off may have plenty.
	if depth+1 >= maxMaskDescentDepth && hasMaskableSubstructure(field) {
		return slog.String(name, depthLimitPlaceholder), true
	}
	// Leaf: preserve the original value as-is so structured handlers
	// (JSON, OTLP) render it natively.
	if field.CanInterface() {
		return slog.Any(name, field.Interface()), false
	}
	return slog.Attr{Key: name}, false
}

// structFieldName returns the name to use for a struct field in the
// masked output. Honors `json` tags (so callers who already JSON-tagged
// their structs get the masking matching they expect) but falls back to
// the Go field name.
func structFieldName(ft reflect.StructField) string {
	tag, ok := ft.Tag.Lookup("json")
	if !ok {
		return ft.Name
	}
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	switch tag {
	case "":
		return ft.Name
	case "-":
		return ""
	default:
		return tag
	}
}

// timeType / textMarshalerType / stringerType / logValuerType pre-cache
// the reflect.Type values we compare against in [isAtomicType] so the
// hot path does not allocate. logValuer is the slog way to control a
// value's representation; respecting it keeps custom types intact.
var (
	timeType          = reflect.TypeFor[time.Time]()
	durationType      = reflect.TypeFor[time.Duration]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
	logValuerType     = reflect.TypeFor[slog.LogValuer]()
	byteSliceType     = reflect.TypeFor[[]byte]()
)

// isAtomicType reports whether reflect descent must stop at this type.
// Types that ship their own canonical representation (time.Time,
// encoding.TextMarshaler implementations, []byte) should be rendered as-is by the downstream
// handler, not flattened into a slog group.
func isAtomicType(t reflect.Type) bool {
	switch t {
	case timeType, durationType, byteSliceType:
		return true
	}
	if t.Implements(textMarshalerType) || reflect.PointerTo(t).Implements(textMarshalerType) {
		return true
	}
	// fmt.Stringer is intentionally NOT treated as atomic: it is a display
	// hint, not a canonical encoding. A struct with sensitive fields that also
	// implements String() must still be descended into and masked field by
	// field; otherwise its String() output would be logged verbatim, bypassing
	// the masking handler. LogValuer representations are resolved before this
	// check; encoding.TextMarshaler still stops descent here.
	return false
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
		// maskAttribute level via descendIntoAny / walkAny, which is
		// where nested matches inside structs/maps/slices are found.
		return slog.String(attr.Key, maskAny(mask, attr.Value.Any(), depth))

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
