// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogxtest_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestRecorderAndConformance(t *testing.T) {
	t.Parallel()
	var r *slogxtest.Recorder
	slogxtest.Conformance(t, func(t *testing.T) slog.Handler { r = slogxtest.NewRecorder(); return r }, func(t *testing.T) map[string]any {
		rows, err := r.Records()
		require.NoError(t, err)
		require.Len(t, rows, 1)
		return rows[0]
	})
	r = slogxtest.NewRecorder()
	require.Same(t, r, r.WithGroup(""))
	require.False(t, slogxtest.Discard().Enabled(t.Context(), slog.LevelError))
	slog.New(r).With("captured", "yes").WithGroup("g").Info("record", "x", 1)
	rows, err := r.Records()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	r.Reset()
	rows, err = r.Records()
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NoError(t, slogxtest.GoroutineLeaks())
}
func TestGolden(t *testing.T) {
	t.Parallel()
	slogxtest.Golden(t, "slogxtest/example.golden", []byte("golden\n"), false)
}
