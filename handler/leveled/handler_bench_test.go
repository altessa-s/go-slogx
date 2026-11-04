// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package leveled_test

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

func BenchmarkEnabled(b *testing.B) {
	for _, captured := range []bool{false, true} {
		name := "uncaptured"
		if captured {
			name = "captured"
		}

		b.Run(name, func(b *testing.B) {
			var handler slog.Handler = mustNew(b, slog.NewJSONHandler(io.Discard, nil))
			if captured {
				handler = handler.WithAttrs([]slog.Attr{slog.String("subsystem", "worker")})
			}

			ctx := b.Context()
			b.ReportAllocs()

			for b.Loop() {
				_ = handler.Enabled(ctx, slog.LevelInfo)
			}
		})
	}
}
