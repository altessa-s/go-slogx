// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx"
	"github.com/altessa-s/go-slogx/console"
	"github.com/altessa-s/go-slogx/slogxtest"
)

func parse(t *testing.T, data string) map[string]any {
	t.Helper()
	out := map[string]any{}
	re := regexp.MustCompile(`("(?:\\.|[^"\\])*")=("(?:\\.|[^"\\])*")`)
	for _, m := range re.FindAllStringSubmatch(data, -1) {
		key, err := strconv.Unquote(m[1])
		require.NoError(t, err)
		value, err := strconv.Unquote(m[2])
		require.NoError(t, err)
		parts := strings.Split(key, ".")
		target := out
		for _, part := range parts[:len(parts)-1] {
			child, ok := target[part].(map[string]any)
			if !ok {
				child = map[string]any{}
				target[part] = child
			}
			target = child
		}
		target[parts[len(parts)-1]] = value
	}
	return out
}
func TestConformance(t *testing.T) {
	t.Parallel()
	var buffer bytes.Buffer
	slogxtest.Conformance(t, func(t *testing.T) slog.Handler { buffer.Reset(); return mustNew(t, &buffer) }, func(t *testing.T) map[string]any { return parse(t, buffer.String()) })
}

type value struct{ calls *int }

func (v value) LogValue() slog.Value { *v.calls++; return slog.StringValue("resolved") }
func TestDynamicLevelReplaceAttrAndStyles(t *testing.T) {
	t.Parallel()
	var colored, plain bytes.Buffer
	level := new(slog.LevelVar)
	level.Set(slog.LevelWarn)
	calls := 0
	var keys []string
	h := mustNew(t, &colored, console.WithColor(true), console.WithLevel(level), console.WithTimeFormat("2006"), console.WithAddSource(true), console.WithKeyStyles(map[string]console.Style{"public": console.Green}), console.WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
		keys = append(keys, strings.Join(groups, ".")+"/"+a.Key)
		require.NotEqual(t, slog.KindLogValuer, a.Value.Kind())
		if a.Key == "drop" {
			return slog.Attr{}
		}
		return a
	}))
	require.False(t, h.Enabled(t.Context(), slog.LevelInfo))
	level.Set(slog.LevelDebug)
	require.True(t, h.Enabled(t.Context(), slog.LevelDebug))
	require.Same(t, h, h.WithGroup(""))
	log := slog.New(h).WithGroup("g").With("captured", value{&calls})
	log.Info("message", slog.Any("value", value{&calls}), slog.Time("when", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)), slogx.Error(errors.New("test")), slog.String("public", "ok"), slog.String("drop", "hidden"), slogx.Module("module"))
	require.Equal(t, 2, calls)
	require.Contains(t, colored.String(), "\x1b[31m")
	require.Contains(t, colored.String(), `"g.when"="2026"`)
	require.NotContains(t, colored.String(), "hidden")
	require.Contains(t, keys, "/time")
	require.Contains(t, keys, "/level")
	require.Contains(t, keys, "/msg")
	require.Contains(t, keys, "/source")
	require.Contains(t, keys, "g/value")
	slog.New(mustNew(t, &plain)).Info("plain")
	require.NotContains(t, plain.String(), "\x1b")
	level.Set(slogx.LevelOff)
	require.False(t, h.Enabled(t.Context(), slog.LevelError))
	_, err := console.New(nil)
	require.Error(t, err)
}

type shortWriter struct{}

func (shortWriter) Write([]byte) (int, error) { return 0, nil }
func TestWriteErrors(t *testing.T) {
	t.Parallel()
	h := mustNew(t, shortWriter{})
	require.ErrorIs(t, h.Handle(t.Context(), slog.Record{}), io.ErrShortWrite)
}
func FuzzConsoleQuoting(f *testing.F) {
	f.Add("key\n\x1b", "value\r\n\x1b")
	f.Fuzz(func(t *testing.T, key, value string) {
		var b bytes.Buffer
		slog.New(mustNew(t, &b)).Info(value, slog.String(key, value))
		out := b.String()
		require.Equal(t, 1, strings.Count(out, "\n"))
		require.NotContains(t, out, "\r")
		require.NotContains(t, out, "\x1b")
	})
}
