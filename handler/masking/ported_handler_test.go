// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/masking"
)

// captureStore is a shared store that survives handler cloning via WithAttrs/WithGroup.
type captureStore struct {
	attrs map[string]slog.Value
}

type captureHandler struct {
	store *captureStore
}

func newCaptureHandler() (*captureHandler, *captureStore) {
	s := &captureStore{}
	return &captureHandler{store: s}, s
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.store.attrs = make(map[string]slog.Value)
	r.Attrs(func(a slog.Attr) bool {
		h.store.attrs[a.Key] = a.Value
		return true
	})
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return &captureHandler{store: h.store}
}

func (h *captureHandler) WithGroup(string) slog.Handler {
	return &captureHandler{store: h.store}
}

func TestHandler_WithAttrs_PreservesPatterns(t *testing.T) {
	t.Parallel()
	inner, store := newCaptureHandler()
	h := portedHandler(t, inner,
		patternOption(`(?i).*secret.*`, fullMask()),
	)

	h2 := h.WithAttrs([]slog.Attr{slog.String("extra", "val")})

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	rec.AddAttrs(slog.String("my_secret_key", "sensitive"))

	require.NoError(t, h2.Handle(t.Context(), rec))

	got := store.attrs["my_secret_key"].String()
	require.Equal(t, "********", got)
}

func TestHandler_WithGroup_PreservesPatterns(t *testing.T) {
	t.Parallel()
	inner, store := newCaptureHandler()
	h := portedHandler(t, inner,
		patternOption(`(?i).*secret.*`, fullMask()),
	)

	h2 := h.WithGroup("group1")

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	rec.AddAttrs(slog.String("my_secret_key", "sensitive"))

	require.NoError(t, h2.Handle(t.Context(), rec))

	got := store.attrs["my_secret_key"].String()
	require.Equal(t, "********", got)
}

func TestHandler_MasksByFieldPattern(t *testing.T) {
	t.Parallel()
	inner, store := newCaptureHandler()
	h := portedHandler(t, inner,
		masking.WithNested(true),
		masking.WithCaseSensitive(true),
		patternOption(`(?i).*password.*`, fullMask()),
	)

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	rec.AddAttrs(slog.String("Password", "secret123"))

	require.NoError(t, h.Handle(t.Context(), rec))

	got := store.attrs["Password"].String()
	require.Equal(t, "********", got)
}
