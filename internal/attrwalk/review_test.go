// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package attrwalk_test

import (
	"fmt"
	"log/slog"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/internal/attrwalk"
)

func TestH2BreadthBound(t *testing.T) {
	t.Parallel()
	w := &attrwalk.Walker{Match: matcher}
	fields := make([]reflect.StructField, 300)
	values := make([]any, 300)
	entries := make(map[string]any, 300)
	attrs := make([]slog.Attr, 300)
	for i := range values {
		name := fmt.Sprintf("Field%d", i)
		fields[i] = reflect.StructField{Name: name, Type: reflect.TypeFor[string]()}
		values[i] = "visible"
		entries[name] = "visible"
		attrs[i] = slog.String(name, "visible")
	}
	for _, a := range []slog.Attr{slog.Any("slice", values), slog.Any("map", entries), slog.Any("struct", reflect.New(reflect.StructOf(fields)).Elem().Interface()), slog.GroupAttrs("group", attrs...)} {
		out := w.Attr(a, nil)
		if a.Value.Kind() == slog.KindGroup {
			require.Len(t, out.Value.Group(), 257)
		}
		require.Contains(t, fmt.Sprint(out), "max breadth exceeded")
	}
}
