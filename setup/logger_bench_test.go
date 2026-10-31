// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/altessa-s/go-slogx/setup"
)

func BenchmarkJSONChain(b *testing.B) {
	for _, name := range []string{"stdlib", "setup", "setup-masking"} {
		b.Run(name, func(b *testing.B) {
			log := slog.New(slog.NewJSONHandler(io.Discard, nil))
			if name != "stdlib" {
				l, err := setup.New(io.Discard, setup.WithFormat(setup.JSON), setup.WithAppName(""), setup.WithAppVersion(""), setup.WithMaskDefaults(name == "setup-masking"))
				if err != nil {
					b.Fatal(err)
				}
				log = l.Logger()
			}
			b.ReportAllocs()
			for b.Loop() {
				log.Info("record", "key", "value")
			}
		})
	}
}
