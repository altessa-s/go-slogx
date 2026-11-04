// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter_test

import (
	"io"
	"testing"

	"github.com/altessa-s/go-slogx/bufwriter"
)

func BenchmarkWriter(b *testing.B) {
	w, err := bufwriter.New(io.Discard)
	if err != nil {
		b.Fatal(err)
	}

	data := []byte("encoded record\n")

	b.ReportAllocs()

	for b.Loop() {
		if _, err := w.Write(data); err != nil {
			b.Fatal(err)
		}
	}

	if err := w.Close(b.Context()); err != nil {
		b.Fatal(err)
	}
}
