// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package leveled_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx"
	"github.com/altessa-s/go-slogx/handler/leveled"
	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestDynamicLevelAndGroupedSubsystem(t *testing.T) {
	// Serial: testing.AllocsPerRun changes process-wide runtime settings.
	recorder := slogxtest.NewRecorder()
	level := new(slog.LevelVar)
	level.Set(slog.LevelWarn)
	levels := map[string]slog.Level{"verbose": slog.LevelDebug, "off": slogx.LevelOff}
	h := mustNew(t, recorder, leveled.WithDefaultLevel(level), leveled.WithSubsystemLevels(levels))
	levels["verbose"] = slog.LevelError
	require.Equal(t, slog.LevelDebug, h.MinLevel().Level())
	require.Same(t, h, h.WithGroup(""))
	log := slog.New(h)
	log.Debug("record", slogx.Module("verbose"))
	log.With(slogx.Module("verbose")).Debug("captured")
	log.WithGroup("g").With(slogx.Module("verbose")).Debug("grouped")
	log.WithGroup("g").Debug("grouped-record", slogx.Module("verbose"))
	log.With(slogx.Module("off")).Error("off")
	log.Info("default-filtered")
	level.Set(slog.LevelDebug)
	log.Debug("default-visible")
	level.Set(slogx.LevelOff)
	log.With(slogx.Module("verbose")).Error("global-off")
	records, err := recorder.Records()
	require.NoError(t, err)
	require.Len(t, records, 3)
	require.Equal(t, slogx.LevelOff, h.MinLevel().Level())
	blocked := mustNew(t, slog.DiscardHandler, leveled.WithSubsystemLevels(map[string]slog.Level{"x": slog.LevelDebug}))
	require.False(t, blocked.WithAttrs([]slog.Attr{slogx.Module("x")}).Enabled(t.Context(), slog.LevelError))
	require.Zero(t, testing.AllocsPerRun(100, func() { _ = h.Enabled(t.Context(), slog.LevelInfo) }))
}
