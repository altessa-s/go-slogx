// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package prefixed_test

import (
	"io"
	"log/slog"
	"testing"
)

func BenchmarkHandler(b *testing.B) {
	h := mustNew(b, slog.NewJSONHandler(io.Discard, nil))
	log := slog.New(h)
	b.ReportAllocs()
	for b.Loop() {
		log.Info("record", "public", "value")
	}
}
