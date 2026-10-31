// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/bufwriter"
	"github.com/altessa-s/go-slogx/setup"
)

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { panic("private") }
func TestOptionsAndBypassDerivedHandlers(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		for _, pointer := range []*string{nil, new("custom")} {
			var b bytes.Buffer
			opts := []setup.Option{setup.WithAddSource(true), setup.WithColor(setup.ColorAlways), setup.WithAppGroup(pointer), setup.WithAppName(pointer), setup.WithAppVersion(pointer), setup.WithServiceID(pointer), setup.WithSubsystemKey(pointer), setup.WithTimeFormat(pointer), setup.WithMaskString(pointer), setup.WithBufferSize(8), setup.WithBufferBypassLevel(slog.LevelInfo)}
			l, err := setup.New(&b, opts...)
			require.NoError(t, err)
			log := l.Logger().With("captured", "yes").WithGroup("nested")
			log.Info("flush", "value", 1)
			require.Contains(t, b.String(), "flush")
			require.Contains(t, b.String(), "\x1b[")
			require.NoError(t, l.Shutdown(t.Context()))
		}
		var b bytes.Buffer
		l, err := setup.New(&b, setup.WithAppGroup("meta"), setup.WithSubsystemKey("component"), setup.WithTimeFormat("2006"), setup.WithMaskString("hidden"), setup.WithColor(setup.ColorNever))
		require.NoError(t, err)
		l.Logger().Info("custom")
		require.NoError(t, l.Shutdown(t.Context()))
		require.NotContains(t, b.String(), "\x1b[")
		var callback error
		l, err = setup.New(failWriter{}, setup.WithFormat(setup.JSON), setup.WithBufferSize(8), setup.WithOnError(func(err error) { callback = err }))
		require.NoError(t, err)
		l.Logger().Error("failure")
		require.ErrorIs(t, callback, bufwriter.ErrPanicked)
		require.NoError(t, l.Shutdown(t.Context()))
		_, err = setup.New(io.Discard, setup.WithBufferSize(8), setup.WithSink(func(io.Writer, *slog.HandlerOptions) slog.Handler { return nil }))
		require.Error(t, err)
	})
}
func TestBypassConformance(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var b bytes.Buffer
		l, err := setup.New(&b, setup.WithFormat(setup.JSON), setup.WithBufferSize(8))
		require.NoError(t, err)
		h := l.Logger().Handler()
		require.Same(t, h, h.WithGroup(""))
		require.True(t, h.Enabled(t.Context(), slog.LevelError))
		require.NoError(t, h.Handle(t.Context(), slog.Record{Level: slog.LevelInfo, Message: "normal"}))
		require.NoError(t, l.Shutdown(t.Context()))
		require.Contains(t, b.String(), "normal")
	})
	require.False(t, errors.Is(io.EOF, bufwriter.ErrPanicked))
}
