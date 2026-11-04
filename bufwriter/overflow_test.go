// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/bufwriter"
)

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
		w, err := bufwriter.New(sink, bufwriter.WithSize(1))
		require.NoError(t, err)
		_, err = w.Write([]byte("first"))
		require.NoError(t, err)
		<-sink.entered

		_, err = w.Write([]byte("queued"))
		require.NoError(t, err)

		complete := make(chan error, 1)

		go func() { _, err := w.Write([]byte("overflow")); complete <- err }()

		synctest.Wait()

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
		w, err := bufwriter.New(sink, bufwriter.WithSize(1))
		require.NoError(t, err)
		_, err = w.Write([]byte("first"))
		require.NoError(t, err)
		<-sink.entered

		_, err = w.Write([]byte("queued"))
		require.NoError(t, err)

		complete := make(chan error, 1)

		go func() { _, err := w.Write([]byte("overflow")); complete <- err }()

		synctest.Wait()

		flushed := make(chan error, 1)
		go func() { flushed <- w.Flush(t.Context()) }()

		close(sink.release)
		require.NoError(t, <-flushed)
		require.NoError(t, <-complete)
		require.NoError(t, w.Close(t.Context()))
		require.Equal(t, 3, sink.writes)
	})
}
