// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter_test

import (
	"context"
	"io"
	"testing"

	"github.com/altessa-s/go-slogx/bufwriter"
)

func BenchmarkWriter(b *testing.B) {
	const batchSize = 1024

	w, err := bufwriter.New(io.Discard, bufwriter.WithSize(batchSize))
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		// The benchmark context is canceled before cleanup runs.
		if _, err := w.Close(context.Background()); err != nil {
			b.Error(err)
		}
	})

	data := []byte("encoded record\n")
	batchWrites := 0

	b.ReportAllocs()

	for b.Loop() {
		// Drain outside the timer after at most one queue's worth of writes.
		if batchWrites == batchSize {
			b.StopTimer()

			if err := w.Flush(b.Context()); err != nil {
				b.Fatal(err)
			}

			batchWrites = 0

			b.StartTimer()
		}

		if _, err := w.Write(data); err != nil {
			b.Fatal(err)
		}

		batchWrites++
	}
}
