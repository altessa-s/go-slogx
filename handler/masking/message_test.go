// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/masking"
	"github.com/altessa-s/go-slogx/mask"
)

func TestMessageMaskOptIn(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		fn   mask.Func
		want string
	}{
		{name: "default", want: "private message"},
		{name: "enabled", fn: mask.Fixed("redacted"), want: "redacted"},
		{name: "panic", fn: func(string) string { panic("private panic") }, want: "********"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer

			h, err := masking.New(slog.NewJSONHandler(&out, nil), masking.WithMessageMask(tc.fn))
			require.NoError(t, err)

			derived := h.WithGroup("scope").WithAttrs([]slog.Attr{slog.Bool("captured", true)})
			record := slog.NewRecord(time.Time{}, slog.LevelInfo, "private message", 0)
			record.AddAttrs(slog.String(slog.MessageKey, "attribute"))
			require.NoError(t, derived.Handle(t.Context(), record))
			require.Equal(t, "private message", record.Message)

			var got map[string]any
			require.NoError(t, json.Unmarshal(out.Bytes(), &got))
			require.Equal(t, tc.want, got[slog.MessageKey])
			require.Equal(t, map[string]any{"captured": true, slog.MessageKey: "attribute"}, got["scope"])
		})
	}
}

func TestFullPathMasksCapturedAndRecordAttrsWithoutDefaults(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	h, err := masking.New(slog.NewJSONHandler(&out, nil), masking.WithRules(
		mask.Rule{Field: "request.auth.id", Mask: mask.Fixed("path")},
		mask.Rule{Field: "id", Mask: mask.Fixed("leaf")},
		mask.Rule{Pattern: `^token$`, Mask: mask.Fixed("pattern")},
	))
	require.NoError(t, err)

	logger := slog.New(h).WithGroup("Request").With(slog.Group("Auth", slog.String("ID", "captured secret")))
	logger.LogAttrs(t.Context(), slog.LevelInfo, "message",
		slog.Group("other", slog.String("id", "leaf secret"), slog.String("token", "pattern secret")),
		slog.Any("auth", map[string]string{"id": "record secret"}),
	)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, map[string]any{
		"Auth":  map[string]any{"ID": "path"},
		"other": map[string]any{"id": "leaf", "token": "pattern"},
		"auth":  map[string]any{"id": "path"},
	}, got["Request"])
}
