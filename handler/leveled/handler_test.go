// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package leveled_test

import (
	"fmt"
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

type subsystemValue struct {
	calls *int
	value slog.Value
}

func (v subsystemValue) LogValue() slog.Value {
	*v.calls++
	return v.value
}

func TestResolvedInlineSubsystem(t *testing.T) {
	t.Parallel()
	for _, captured := range []bool{false, true} {
		for _, name := range []string{"off", "verbose"} {
			for _, shape := range []string{"inline", "valuer", "group-valuer", "named"} {
				t.Run(fmt.Sprintf("%v/%s/%s", captured, name, shape), func(t *testing.T) {
					t.Parallel()
					recorder := slogxtest.NewRecorder()
					h := mustNew(t, recorder, leveled.WithDefaultLevel(slog.LevelWarn),
						leveled.WithSubsystemLevels(map[string]slog.Level{"off": slogx.LevelOff, "verbose": slog.LevelDebug}))
					calls := 0
					attr := slog.String("subsystem", name)
					switch shape {
					case "inline":
						attr = slog.GroupAttrs("", slog.GroupAttrs("", attr))
					case "valuer":
						attr = slog.Any("subsystem", subsystemValue{&calls, slog.StringValue(name)})
					case "group-valuer":
						attr = slog.Any("", subsystemValue{&calls, slog.GroupValue(attr)})
					case "named":
						attr = slog.GroupAttrs("named", attr)
					}
					logger := slog.New(h)
					level := slog.LevelError
					if name == "verbose" {
						level = slog.LevelDebug
					}
					if captured {
						logger = logger.With(attr)
						logger.Log(t.Context(), level, "record")
					} else {
						logger.Log(t.Context(), level, "record", attr)
					}
					records, err := recorder.Records()
					require.NoError(t, err)
					visible := name == "verbose"
					if shape == "named" {
						visible = name == "off"
					}
					if visible {
						require.Len(t, records, 1)
					} else {
						require.Empty(t, records)
					}
					if shape == "valuer" || shape == "group-valuer" {
						require.Equal(t, 1, calls)
					}
				})
			}
		}
	}
}

type recursiveInlineValue struct{}

func (recursiveInlineValue) LogValue() slog.Value {
	return slog.GroupValue(slog.Any("", recursiveInlineValue{}))
}

func TestRecursiveInlineSubsystemIsBounded(t *testing.T) {
	t.Parallel()
	for _, captured := range []bool{false, true} {
		recorder := slogxtest.NewRecorder()
		logger := slog.New(mustNew(t, recorder))
		attr := slog.Any("", recursiveInlineValue{})
		if captured {
			logger.With(attr).Info("record")
		} else {
			logger.Info("record", attr)
		}
		records, err := recorder.Records()
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Contains(t, fmt.Sprint(records), "max depth exceeded")
	}
}
