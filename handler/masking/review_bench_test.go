// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"io"
	"log/slog"
	"strconv"
	"testing"

	"github.com/altessa-s/go-slogx/handler/masking"
)

func BenchmarkH2LargeSliceDefaults(b *testing.B) {
	for _, size := range []int{16, 100000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			h, err := masking.New(slog.NewJSONHandler(io.Discard, nil), masking.WithDefaults(true))
			if err != nil {
				b.Fatal(err)
			}
			log := slog.New(h)
			data := make([]int, size)
			b.ReportAllocs()
			for b.Loop() {
				log.Info("record", "payload", data)
			}
		})
	}
}
func BenchmarkM5HighCardinalityMaps(b *testing.B) {
	h, err := masking.New(slog.NewJSONHandler(io.Discard, nil), masking.WithDefaults(true))
	if err != nil {
		b.Fatal(err)
	}
	log := slog.New(h)
	values := make([]map[string]string, 8192)
	for i := range values {
		values[i] = map[string]string{"field" + strconv.Itoa(i): "public"}
	}
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		log.Info("record", "payload", values[i%len(values)])
		i++
	}
}
