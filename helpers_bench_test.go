// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx_test

import (
	"testing"

	"github.com/altessa-s/go-slogx"
)

func BenchmarkString(b *testing.B) {
	s := "value"

	b.ReportAllocs()

	for b.Loop() {
		_ = slogx.String("key", &s)
	}
}
func BenchmarkFields(b *testing.B) {
	f := slogx.Fields{{Key: "a.b", Value: 1}, {Key: "x", Value: "value"}}

	b.ReportAllocs()

	for b.Loop() {
		_ = f.Grouped()
		_ = f.Attrs()
		_ = f.Args()
	}
}
func BenchmarkContext(b *testing.B) {
	ctx := slogx.InjectFields(b.Context(), slogx.Fields{{Key: "request", Value: "id"}})
	b.ReportAllocs()

	for b.Loop() {
		_ = slogx.FieldsFromContext(ctx)
	}
}

func BenchmarkModule(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = slogx.Module("worker")
	}
}
func BenchmarkError(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = slogx.Error(nil)
	}
}
