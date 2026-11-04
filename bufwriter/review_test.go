// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter_test

import (
	"errors"
	"io"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/bufwriter"
)

func TestM1FlushResetsJoinedErrors(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		w, err := bufwriter.New(writerFunc(func(p []byte) (int, error) {
			calls++
			if calls == 1 {
				return 0, io.ErrClosedPipe
			}

			if calls == 2 {
				return 0, io.ErrUnexpectedEOF
			}

			return len(p), nil
		}))
		require.NoError(t, err)

		for range 2 {
			_, err = w.Write([]byte("line"))
			require.NoError(t, err)
		}

		err = w.Flush(t.Context())
		require.ErrorIs(t, err, io.ErrClosedPipe)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		_, err = w.Write([]byte("success"))
		require.NoError(t, err)
		require.NoError(t, w.Flush(t.Context()))
		require.NoError(t, w.Close(t.Context()))
	})
}
func TestM2ClosedWriteReportsOnce(t *testing.T) {
	t.Parallel()

	reports := 0
	w, err := bufwriter.New(io.Discard, bufwriter.WithOnError(func(err error) {
		if errors.Is(err, bufwriter.ErrClosed) {
			reports++
		}
	}))
	require.NoError(t, err)
	require.NoError(t, w.Close(t.Context()))

	for range 2 {
		n, err := w.Write([]byte("record"))
		require.NoError(t, err)
		require.Equal(t, 6, n)
	}

	require.Equal(t, 2, reports)
}
