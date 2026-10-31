// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/masking"
)

// errUnexportedMsg models the shape that triggers the regression: the
// human-readable payload lives in an unexported field and is reachable only
// through Error()/String() (as for a proto validation error).
type errUnexportedMsg struct{ msg string }

func (e *errUnexportedMsg) Error() string { return e.msg }

// Private error payloads must not reach native sink formatting.
func TestHandler_KindAny_PrivateErrorUsesJSON(t *testing.T) {
	t.Parallel()
	inner, store := newCaptureHandler()
	// WithDefaults mirrors the service wiring (factory uses it), so the masking
	// reflect-descent path is active rather than the no-rules fast path.
	h := portedHandler(t, inner, masking.WithDefaults(true))
	lg := slog.New(h)

	lg.LogAttrs(t.Context(), slog.LevelWarn, "validation failed",
		slog.Any("error", &errUnexportedMsg{msg: "slug: required field cannot be cleared"}))

	v, ok := store.attrs["error"]
	require.True(t, ok, "error attribute must reach the inner handler")
	require.Equal(t, map[string]any{}, v.Any())
}

// stringerOnly exposes a private payload through native text formatting.
type stringerOnly struct{ v string }

func (s stringerOnly) String() string { return s.v }

// plainUnexported has no JSON fields.
type plainUnexported struct{ x int }

// renderThroughMasking logs a single attribute through the masking handler into
// a real slog text handler (WithDefaults mirrors the service wiring). Unlike the
// capture handler, a real handler omits empty groups — which is exactly the slog
// behavior that makes a collapsed attribute disappear from the log — so this is
// what actually proves an attribute survives to the output.
func renderThroughMasking(t *testing.T, attr slog.Attr) string {
	t.Helper()
	var buf bytes.Buffer
	h := portedHandler(t, slog.NewTextHandler(&buf, nil), masking.WithDefaults(true))
	rec := slog.NewRecord(time.Time{}, slog.LevelWarn, "validation failed", 0)
	rec.AddAttrs(attr)
	require.NoError(t, h.Handle(t.Context(), rec))
	return buf.String()
}

// Empty JSON containers remain present instead of becoming omitted slog groups.
func TestHandler_KindAny_NoExportedFieldsSurviveDescent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		key   string
		value any
		want  string // substring that must appear in the rendered line
	}{
		{"empty map", "meta", map[string]string{}, "meta"},
		{"empty slice", "ids", []string{}, "ids"},
		{"stringer only", "note", stringerOnly{v: "note"}, "note"},
		{"plain unexported struct", "blob", plainUnexported{x: 7}, "blob"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := renderThroughMasking(t, slog.Any(tc.key, tc.value))
			require.Contains(t, out, tc.key,
				"attribute key must survive masking descent, not collapse to an omitted empty group")
			require.Contains(t, out, tc.want)
		})
	}
}
