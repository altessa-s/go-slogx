// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas-core/runtime/leakcheck"

	"github.com/altessa-s/go-slogx/bufwriter"
)

func TestMain(m *testing.M) {
	code := m.Run()
	if err := leakcheck.Check(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
func TestDrainCopiesAndClosedFlush(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var sink bytes.Buffer
		w, err := bufwriter.New(&sink)
		require.NoError(t, err)
		data := []byte("original")
		_, err = w.Write(data)
		require.NoError(t, err)
		copy(data, "modified")
		require.NoError(t, w.Flush(t.Context()))
		require.Equal(t, "original", sink.String())
		require.NoError(t, w.Close(t.Context()))
		require.NoError(t, w.Close(t.Context()))
		require.NoError(t, w.Flush(t.Context()))
		_, err = w.Write(data)
		require.NoError(t, err)
		require.Equal(t, "originalmodified", sink.String())
	})
}
func TestPanicRecoveryAndWriteErrors(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var calls, reported atomic.Int32
		var observed error
		sink := writerFunc(func(p []byte) (int, error) {
			switch calls.Add(1) {
			case 1:
				panic("private panic payload")
			case 2:
				return 0, io.ErrClosedPipe
			case 3:
				return 0, nil
			default:
				return len(p), nil
			}
		})
		w, err := bufwriter.New(sink, bufwriter.WithOnError(func(err error) { observed = errors.Join(observed, err); reported.Add(1); panic("callback") }))
		require.NoError(t, err)
		for range 4 {
			_, err = w.Write([]byte("line"))
			require.NoError(t, err)
		}
		err = w.Flush(t.Context())
		require.ErrorIs(t, err, bufwriter.ErrPanicked)
		require.ErrorIs(t, observed, io.ErrClosedPipe)
		require.ErrorIs(t, observed, io.ErrShortWrite)
		require.NotContains(t, err.Error(), "payload")
		require.NoError(t, w.Close(t.Context()))
		require.Equal(t, int32(3), reported.Load())
	})
}
func TestOverflowAndDeadline(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		entered := make(chan struct{})
		var count atomic.Int32
		sink := writerFunc(func(p []byte) (int, error) {
			if count.Add(1) == 1 {
				close(entered)
				<-release
			}
			return len(p), nil
		})
		w, err := bufwriter.New(sink, bufwriter.WithSize(1))
		require.NoError(t, err)
		_, err = w.Write([]byte("first"))
		require.NoError(t, err)
		<-entered
		_, err = w.Write([]byte("queued"))
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		require.ErrorIs(t, w.Flush(ctx), context.DeadlineExceeded)
		completed := make(chan struct{})
		go func() { _, _ = w.Write([]byte("overflow")); close(completed) }()

		close(release)
		<-completed
		require.NoError(t, w.Close(t.Context()))
		require.Equal(t, int32(3), count.Load())
	})
}
func TestCloseTimeoutStillDrains(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		entered := make(chan struct{})
		w, err := bufwriter.New(writerFunc(func(p []byte) (int, error) { close(entered); <-release; return len(p), nil }))
		require.NoError(t, err)
		_, err = w.Write([]byte("x"))
		require.NoError(t, err)
		<-entered
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, w.Close(ctx), context.Canceled)
		require.ErrorIs(t, w.Flush(ctx), context.Canceled)
		close(release)
		require.NoError(t, w.Close(t.Context()))
	})
	_, err := bufwriter.New(nil)
	require.Error(t, err)
	_, err = bufwriter.New(io.Discard, bufwriter.WithSize(0))
	require.Error(t, err)
	require.False(t, errors.Is(err, bufwriter.ErrClosed))
}

func TestConcurrentFlushDoesNotBlockCloseDeadline(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		w, err := bufwriter.New(writerFunc(func(p []byte) (int, error) {
			select {
			case entered <- struct{}{}:
				<-release
			default:
			}
			return len(p), nil
		}), bufwriter.WithSize(1))
		require.NoError(t, err)
		written := make(chan error, 1)
		go func() { _, err := w.Write([]byte("first")); written <- err }()
		<-entered
		require.NoError(t, <-written)
		_, err = w.Write([]byte("queued"))
		require.NoError(t, err)
		flushed := make(chan error, 1)
		go func() { flushed <- w.Flush(t.Context()) }()
		synctest.Wait()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, w.Close(ctx), context.Canceled)
		close(release)
		require.NoError(t, <-flushed)
		require.NoError(t, w.Close(t.Context()))
	})
}
