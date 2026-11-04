// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/altessa-s/go-slogx/handler/masking"
)

func BenchmarkHandler(b *testing.B) {
	h := func() slog.Handler {
		h, err := masking.New(slog.NewJSONHandler(io.Discard, nil), masking.WithDefaults(true))
		if err != nil {
			b.Fatal(err)
		}

		return h
	}()
	log := slog.New(h)

	b.ReportAllocs()

	for b.Loop() {
		log.Info("record", "public", "value")
	}
}

func BenchmarkMaskingShapes(b *testing.B) {
	cases := []struct {
		name string
		attr slog.Attr
	}{
		{"flat", slog.String("password", "secret")},
		{"group", slog.Group("request", slog.String("password", "secret"))},
		{"struct", slog.Any("request", struct{ Password string }{"secret"})},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			handler, err := masking.New(slog.NewJSONHandler(io.Discard, nil), masking.WithDefaults(true))
			if err != nil {
				b.Fatal(err)
			}

			log := slog.New(handler)
			ctx := b.Context()
			b.ReportAllocs()

			for b.Loop() {
				log.LogAttrs(ctx, slog.LevelInfo, "record", tc.attr)
			}
		})
	}
}
