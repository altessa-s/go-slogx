// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/bufwriter"
	"github.com/altessa-s/go-slogx/setup"
)

type observedPipe struct {
	*io.PipeWriter
	entered chan struct{}
}

func (w *observedPipe) Write(p []byte) (int, error) {
	select {
	case w.entered <- struct{}{}:
	default:
	}

	return w.PipeWriter.Write(p)
}

func TestBufferedBlockedPipeBoundsLoggingAndShutdown(t *testing.T) {
	t.Parallel()
	synctest.Test(t, testBufferedBlockedPipe)
}

func testBufferedBlockedPipe(t *testing.T) {
	t.Helper()

	const (
		bypassTimeout = 40 * time.Millisecond
		testTimeout   = time.Second
		priorityCount = 4
	)

	reader, writer := io.Pipe()

	t.Cleanup(func() { _ = writer.Close() })
	t.Cleanup(func() { _ = reader.Close() })

	sink := &observedPipe{PipeWriter: writer, entered: make(chan struct{}, 1)}
	l, err := setup.New(sink, setup.WithFormat(setup.JSON), setup.WithBufferSize(1), setup.WithBufferBypassTimeout(bypassTimeout))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = reader.Close()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		_, _ = l.Shutdown(ctx)
	})

	h := l.Logger().With("captured", true).WithGroup("group").Handler()
	handle := func(level slog.Level) <-chan error {
		done := make(chan error, 1)
		go func() {
			done <- h.Handle(t.Context(), slog.NewRecord(time.Time{}, level, "blocked", 0))
		}()

		return done
	}
	result := func(done <-chan error) error {
		t.Helper()

		select {
		case err := <-done:
			return err
		case <-time.After(testTimeout):
			t.Fatal("logging blocked while the pipe had no reader")
			return nil
		}
	}

	require.NoError(t, result(handle(slog.LevelInfo)))

	select {
	case <-sink.entered:
	case <-time.After(testTimeout):
		t.Fatal("worker did not reach the pipe")
	}

	require.NoError(t, result(handle(slog.LevelInfo)))

	start := time.Now()

	priorities := make([]<-chan error, priorityCount)
	for i := range priorities {
		priorities[i] = handle(slog.LevelError)
	}

	require.ErrorIs(t, result(handle(slog.LevelInfo)), bufwriter.ErrDropped)
	require.Zero(t, time.Since(start))

	for _, done := range priorities {
		err := result(done)
		require.ErrorIs(t, err, bufwriter.ErrDropped)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	}

	require.Equal(t, bypassTimeout, time.Since(start))

	type shutdownResult struct {
		stats bufwriter.Stats
		err   error
	}

	closed := make(chan shutdownResult, 1)
	start = time.Now()

	ctx, cancel := context.WithTimeout(t.Context(), bypassTimeout)
	defer cancel()

	go func() {
		stats, err := l.Shutdown(ctx)
		closed <- shutdownResult{stats: stats, err: err}
	}()

	select {
	case got := <-closed:
		require.ErrorIs(t, got.err, context.DeadlineExceeded)
		require.Equal(t, bufwriter.Stats{Pending: 2, Dropped: 1 + priorityCount, DroppedPriority: priorityCount}, got.stats)
	case <-time.After(testTimeout):
		t.Fatal("shutdown blocked after its deadline")
	}

	require.Equal(t, bypassTimeout, time.Since(start))

	require.ErrorIs(t, result(handle(slog.LevelInfo)), bufwriter.ErrClosed)
	require.ErrorIs(t, result(handle(slog.LevelError)), bufwriter.ErrClosed)
	require.NoError(t, reader.Close())

	retry, cancelRetry := context.WithTimeout(t.Context(), testTimeout)
	defer cancelRetry()

	stats, err := l.Shutdown(retry)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.Equal(t, bufwriter.Stats{Failed: 2, Dropped: 1 + priorityCount, DroppedPriority: priorityCount}, stats)
}
