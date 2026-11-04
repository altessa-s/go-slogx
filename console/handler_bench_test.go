// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/altessa-s/go-slogx/console"
)

func BenchmarkHandler(b *testing.B) {
	for _, color := range []bool{false, true} {
		name := "plain"
		if color {
			name = "color"
		}

		b.Run(name, func(b *testing.B) {
			log := slog.New(mustNew(b, io.Discard, console.WithColor(color)))
			b.ReportAllocs()

			for b.Loop() {
				log.Info("record", "key", "value")
			}
		})
	}
}

func BenchmarkTextHandler(b *testing.B) {
	for _, color := range []bool{false, true} {
		name := "plain"
		if color {
			name = "color"
		}

		b.Run(name, func(b *testing.B) {
			log := slog.New(mustNew(b, io.Discard, console.WithFormat(console.FormatText), console.WithColor(color)))
			b.ReportAllocs()

			for b.Loop() {
				log.Info("record", "key", "value")
			}
		})
	}
}
