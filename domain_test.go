// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx_test

import (
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx"
	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestHelpers(t *testing.T) {
	t.Parallel()
	type name string
	type pointer *string
	type number int
	type wide int64
	type small int32
	require.Equal(t, slog.Attr{}, slogx.Error(nil))
	require.Equal(t, slogx.ErrorKey, slogx.Error(errors.New("test")).Key)
	for _, a := range []slog.Attr{slogx.String("x", ""), slogx.String("x", new("")), slogx.String("x", name("")), slogx.String("x", pointer(new(""))), slogx.String("x", (*string)(nil))} {
		require.Equal(t, slog.Attr{}, a)
	}
	for _, a := range []slog.Attr{slogx.String("x", "yes"), slogx.String("x", new("yes")), slogx.String("x", name("yes")), slogx.String("x", pointer(new("yes")))} {
		require.Equal(t, "yes", a.Value.String())
	}
	for _, a := range []slog.Attr{slogx.Int("x", 1), slogx.Int("x", new(1)), slogx.Int("x", number(1)), slogx.Int64("x", int64(1)), slogx.Int64("x", new(int64(1))), slogx.Int64("x", int32(1)), slogx.Int64("x", new(int32(1))), slogx.Int64("x", wide(1)), slogx.Int64("x", small(1))} {
		require.Equal(t, int64(1), a.Value.Int64())
	}
	require.Equal(t, slog.Attr{}, slogx.Int("x", (*int)(nil)))
	require.Equal(t, slog.Attr{}, slogx.Int64("x", (*int64)(nil)))
	require.Equal(t, slog.Attr{}, slogx.Int64("x", (*int32)(nil)))
	require.Equal(t, slogx.ModuleKey, slogx.Module("test").Key)
	require.Len(t, slogx.Modules("a", "b"), 2)
}
func TestFieldsAndContextIsolation(t *testing.T) {
	t.Parallel()
	fields := slogx.Fields{}.Append("x", 1).AppendFields(slogx.Fields{{Key: "x", Value: 2}, {Key: "a.b", Value: "nested"}})
	require.Len(t, fields.Unique(), 2)
	require.Len(t, fields.Args(), 6)
	require.Len(t, fields.Attrs(), 3)
	n := 0
	for range fields.All() {
		n++
		break
	}
	require.Equal(t, 1, n)
	require.Equal(t, "a", fields.Grouped()[0].Key)
	backing := slogx.Fields{{Key: "a", Value: "one"}, {Key: "b", Value: "two"}}
	shortened := backing.Delete("a")
	require.Len(t, shortened, 1)
	require.Equal(t, slogx.Field{}, backing[1])
	require.Equal(t, shortened, shortened.Delete("absent"))
	ctx := slogx.InjectFields(t.Context(), fields)
	other := slogx.InjectFields(t.Context(), fields)
	fields[0].Value = 99
	require.Equal(t, 1, slogx.FieldsFromContext(ctx)[0].Value)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() { slogx.AppendField(ctx, "concurrent", true) })
	}
	wg.Wait()
	slogx.AppendFields(ctx, slogx.Fields{{Key: "more", Value: true}})
	require.Len(t, slogx.FieldsFromContext(ctx), 36)
	require.Len(t, slogx.FieldsFromContext(other), 3)
	require.Nil(t, slogx.FieldsFromContext(nil))
	require.Nil(t, slogx.FieldsFromContext(t.Context()))
	slogx.AppendField(t.Context(), "ignored", 1)
	slogx.AppendFields(t.Context(), nil)
	first, second := slogxtest.Discard(), slogxtest.Discard()
	ctx = slogx.ContextWithLogger(ctx, first)
	ctx = slogx.ContextWithLogger(ctx, second)
	require.Same(t, second, slogx.FromContext(ctx))
	require.Same(t, second, slogx.FromContext(slogx.ContextWithLoggerIfAbsent(ctx, first)))
	require.Same(t, first, slogx.FromContext(slogx.ContextWithLoggerIfAbsent(t.Context(), first)))
	require.Same(t, slog.Default(), slogx.FromContext(nil))
	require.Same(t, first, slogx.BuildLogger(t.Context(), first))
	require.Same(t, slog.Default(), slogx.BuildLogger(t.Context(), nil))
	recorder := slogxtest.NewRecorder()
	logger := slog.New(recorder)
	injected := slogx.InjectLogger(other, logger)
	slogx.FromContext(injected).Info("captured")
	records, err := recorder.Records()
	require.NoError(t, err)
	require.Equal(t, float64(2), records[0]["x"])
	require.Panics(t, func() { slogx.ContextWithLogger(t.Context(), nil) })
}
func FuzzString(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) { require.Equal(t, slogx.String("x", s), slogx.String("x", &s)) })
}
func FuzzFieldsGrouped(f *testing.F) {
	f.Add("a.b", "value")
	f.Fuzz(func(t *testing.T, key, value string) {
		fields := slogx.Fields{{Key: key, Value: value}, {Key: key, Value: "later"}}
		require.Equal(t, fields.Grouped(), fields.Grouped())
		require.Equal(t, (slogx.Fields{{Key: key, Value: value}}).Grouped(), fields.Grouped())
	})
}
