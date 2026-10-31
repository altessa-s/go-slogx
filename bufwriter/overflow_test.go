// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter

import (
	"context"
	"runtime"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

// This white-box test needs the admission state to force overflow without sleeps.
// The test helper observes private state; no production test hooks are exposed.
func waitOverflow(w *Writer) {
	for {
		w.mu.Lock()
		pending := len(w.pending)
		w.mu.Unlock()
		if pending > 0 {
			return
		}
		runtime.Gosched()
	}
}

type gateWriter struct {
	entered chan struct{}
	release chan struct{}
	writes  int
}

func (w *gateWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == 1 {
		close(w.entered)
		<-w.release
	}
	return len(p), nil
}
func TestOverflowCloseHonorsContext(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		sink := &gateWriter{entered: make(chan struct{}), release: make(chan struct{})}
		w, err := New(sink, WithSize(1))
		require.NoError(t, err)
		_, err = w.Write([]byte("first"))
		require.NoError(t, err)
		<-sink.entered
		_, err = w.Write([]byte("queued"))
		require.NoError(t, err)
		complete := make(chan error, 1)
		go func() { _, err := w.Write([]byte("overflow")); complete <- err }()
		waitOverflow(w)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, w.Close(ctx), context.Canceled)
		close(sink.release)
		require.NoError(t, <-complete)
		require.NoError(t, w.Close(t.Context()))
		require.Equal(t, 3, sink.writes)
	})
}
func TestFlushWaitsForOverflow(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		sink := &gateWriter{entered: make(chan struct{}), release: make(chan struct{})}
		w, err := New(sink, WithSize(1))
		require.NoError(t, err)
		_, err = w.Write([]byte("first"))
		require.NoError(t, err)
		<-sink.entered
		_, err = w.Write([]byte("queued"))
		require.NoError(t, err)
		complete := make(chan error, 1)
		go func() { _, err := w.Write([]byte("overflow")); complete <- err }()
		waitOverflow(w)
		flushed := make(chan error, 1)
		go func() { flushed <- w.Flush(t.Context()) }()
		close(sink.release)
		require.NoError(t, <-flushed)
		require.NoError(t, <-complete)
		require.NoError(t, w.Close(t.Context()))
		require.Equal(t, 3, sink.writes)
	})
}
