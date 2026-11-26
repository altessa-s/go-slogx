// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

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
func TestPriorityGetsNextAvailableSlot(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		sink := &gateWriter{entered: make(chan struct{}), release: make(chan struct{})}
		w, err := bufwriter.New(sink, bufwriter.WithSize(1))

		require.NoError(t, err)
		defer func() { require.NoError(t, deliveryError(w.Close(t.Context()))) }()

		_, err = w.Write([]byte("first"))
		require.NoError(t, err)
		<-sink.entered

		_, err = w.Write([]byte("queued"))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		priority := make(chan error, 1)
		go func() { priority <- w.WritePriority(ctx, []byte("priority")) }()

		synctest.Wait()

		_, err = w.Write([]byte("overflow"))

		close(sink.release)
		require.ErrorIs(t, err, bufwriter.ErrDropped)
		require.NoError(t, <-priority)

		stats, err := w.Close(t.Context())
		require.NoError(t, err)
		require.Equal(t, bufwriter.Stats{Written: 3, Dropped: 1}, stats)
	})
}

func TestAdmittedPriorityStaysQueuedAfterTimeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		sink := &gateWriter{entered: make(chan struct{}), release: make(chan struct{})}
		w, err := bufwriter.New(sink, bufwriter.WithSize(1))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		err = w.WritePriority(ctx, []byte("priority"))

		close(sink.release)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.NotErrorIs(t, err, bufwriter.ErrDropped)

		stats, err := w.Close(t.Context())
		require.NoError(t, err)
		require.Equal(t, bufwriter.Stats{Written: 1}, stats)
	})
}
