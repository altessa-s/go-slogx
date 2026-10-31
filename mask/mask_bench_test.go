// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask_test

import (
	"testing"

	"github.com/altessa-s/go-slogx/mask"
)

func BenchmarkPartialMask(b *testing.B) {
	f, _ := mask.Partial(2, 2, '*')
	for _, input := range []string{"sensitive-value", "секретное-значение"} {
		b.Run(input, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = f(input)
			}
		})
	}
}
func BenchmarkURLMask(b *testing.B) {
	f := mask.URL()
	b.ReportAllocs()
	for b.Loop() {
		_ = f("https://user:secret@example.test/path/file?signature=secret")
	}
}
func BenchmarkEmailMask(b *testing.B) {
	f := mask.Email()
	b.ReportAllocs()
	for b.Loop() {
		_ = f("someone@example.test")
	}
}
