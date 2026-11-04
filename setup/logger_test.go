// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas-core/runtime/leakcheck"

	"github.com/altessa-s/go-slogx"
	"github.com/altessa-s/go-slogx/mask"
	"github.com/altessa-s/go-slogx/setup"
	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestMain(m *testing.M) {
	code := m.Run()

	if err := leakcheck.Check(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		code = 1
	}

	os.Exit(code)
}
func TestDynamicLevelsAndChain(t *testing.T) {
	t.Parallel()

	for _, format := range []setup.Format{setup.Text, setup.JSON} {
		t.Run(fmt.Sprint(format), func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				var b bytes.Buffer

				l, err := setup.New(&b, setup.WithFormat(format), setup.WithBufferSize(8), setup.WithMaskDefaults(true), setup.WithAppName("test"), setup.WithAppVersion("v1"), setup.WithServiceID("id"), setup.WithTags(map[string]string{"z": "last", "a": "first"}), setup.WithSubsystemLevels(map[string]slog.Level{"debug": slog.LevelDebug, "off": slogx.LevelOff}))
				require.NoError(t, err)
				l.Logger().Debug("override", slogx.Module("debug"))
				l.Logger().Debug("hidden")
				l.SetLevel(slog.LevelDebug)
				require.Equal(t, slog.LevelDebug, l.Level())
				l.Logger().Debug("runtime")
				l.Logger().Error("bypass", slogx.Module("debug"), "password", "canary")
				require.Contains(t, b.String(), "bypass")
				l.Logger().Error("off", slogx.Module("off"))
				l.SetLevel(slogx.LevelOff)
				l.Logger().Error("disabled")
				require.NoError(t, l.Shutdown(t.Context()))
				require.NoError(t, l.Shutdown(t.Context()))

				out := b.String()
				require.Contains(t, out, "override")
				require.Contains(t, out, "runtime")
				require.NotContains(t, out, "hidden")
				require.NotContains(t, out, "canary")
				require.NotContains(t, out, "disabled")

				if format == setup.JSON {
					require.Contains(t, out, `"subsystem":"debug"`)
				} else {
					require.Contains(t, out, `"subsystem"="[debug]"`)
				}
			})
		})
	}
}

type mutableError struct{ message string }

func (e *mutableError) Error() string { return e.message }

type mutableValue struct{ data map[string]string }

func (v mutableValue) LogValue() slog.Value { return slog.AnyValue(v.data) }
func TestM6R1BufferedEncodingOwnsBytes(t *testing.T) {
	t.Parallel()

	for _, format := range []setup.Format{setup.Text, setup.JSON} {
		for _, defaults := range []bool{false, true} {
			synctest.Test(t, func(t *testing.T) {
				var b bytes.Buffer

				l, err := setup.New(&b, setup.WithFormat(format), setup.WithMaskDefaults(defaults), setup.WithBufferSize(100))
				require.NoError(t, err)

				data := map[string]string{"value": "before"}
				slice := []string{"before"}
				pointer := new("before")
				mutableErr := &mutableError{message: "before"}

				for range 30 {
					data["value"] = "before"
					slice[0] = "before"
					*pointer = "before"
					mutableErr.message = "before"
					l.Logger().Info("snapshot", "map", data, "slice", slice, "pointer", pointer, "valuer", mutableValue{data}, "error", mutableErr)
					data["value"] = "after"
					slice[0] = "after"
					*pointer = "after"
					mutableErr.message = "after"
				}

				require.NoError(t, l.Shutdown(t.Context()))
				require.NotContains(t, b.String(), "after")
				require.Contains(t, b.String(), "before")
			})
		}
	}
}

func TestValidationAndCustomSink(t *testing.T) {
	t.Parallel()

	for _, opts := range [][]setup.Option{{setup.WithBufferSize(-1)}, {setup.WithFormat(99)}, {setup.WithColor(99)}, {setup.WithMaskRules(mask.Rule{Pattern: "["})}, {setup.WithLevel(slogx.LevelOff), setup.WithMaskRules(mask.Rule{})}} {
		_, err := setup.New(io.Discard, opts...)
		require.Error(t, err)
	}

	_, err := setup.New(nil)
	require.Error(t, err)
	_, err = setup.New(io.Discard, setup.WithSink(func(io.Writer, *slog.HandlerOptions) slog.Handler { return nil }))
	require.Error(t, err)

	recorder := slogxtest.NewRecorder()
	l, err := setup.New(io.Discard, setup.WithFormat(setup.JSON), setup.WithSink(func(io.Writer, *slog.HandlerOptions) slog.Handler { return recorder }), setup.WithAppName(""), setup.WithAppVersion(""))
	require.NoError(t, err)
	l.Logger().Info("custom")
	require.NoError(t, l.Shutdown(t.Context()))

	rows, err := recorder.Records()
	require.NoError(t, err)
	require.Len(t, rows, 1)
}
func TestConformance(t *testing.T) {
	t.Parallel()

	var recorder *slogxtest.Recorder

	slogxtest.Conformance(t, func(t *testing.T) slog.Handler {
		recorder = slogxtest.NewRecorder()
		l, err := setup.New(io.Discard, setup.WithFormat(setup.JSON), setup.WithAppName(""), setup.WithAppVersion(""), setup.WithSink(func(io.Writer, *slog.HandlerOptions) slog.Handler { return recorder }))
		require.NoError(t, err)

		return l.Logger().Handler()
	}, func(t *testing.T) map[string]any {
		rows, err := recorder.Records()
		require.NoError(t, err)
		require.Len(t, rows, 1)

		return rows[0]
	})
}

func TestBufferedConformance(t *testing.T) {
	t.Parallel()

	var (
		output bytes.Buffer
		logger *setup.Logger
	)

	slogxtest.Conformance(t, func(t *testing.T) slog.Handler {
		output.Reset()

		var err error

		logger, err = setup.New(&output, setup.WithFormat(setup.JSON), setup.WithBufferSize(8), setup.WithAppName(""), setup.WithAppVersion(""))
		require.NoError(t, err)

		return logger.Logger().Handler()
	}, func(t *testing.T) map[string]any {
		require.NoError(t, logger.Shutdown(t.Context()))

		var record map[string]any
		require.NoError(t, json.Unmarshal(output.Bytes(), &record))

		return record
	})
}
