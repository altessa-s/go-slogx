// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package prefixed_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx"
	"github.com/altessa-s/go-slogx/handler/prefixed"
	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestDefaultKeyGroupsAndFormatter(t *testing.T) {
	t.Parallel()

	r := slogxtest.NewRecorder()
	h := mustNew(t, r)
	require.Same(t, h, h.WithGroup(""))
	log := slog.New(h).With(slogx.Module("a")).With(slogx.Module("b"))
	log.Info("root", slogx.Module("c"), slog.String("", "empty-key"))
	log.WithGroup("g").With(slogx.Module("d")).Info("group", slogx.Module("e"))
	log.Info("non-string", slog.Int(slogx.ModuleKey, 7))

	rows, err := r.Records()
	require.NoError(t, err)
	require.Equal(t, "[a:b:c]", rows[0][slogx.ModuleKey])
	require.Equal(t, "empty-key", rows[0][""])
	require.Equal(t, "[a:b]", rows[1][slogx.ModuleKey])
	require.Equal(t, "[d:e]", rows[1]["g"].(map[string]any)[slogx.ModuleKey])
	r.Reset()
	slog.New(mustNew(t, r, prefixed.WithFormatter(func([]slog.Value, string) (slog.Value, bool) { return slog.Value{}, false }))).Info("suppressed", slogx.Module("x"))
	rows, err = r.Records()
	require.NoError(t, err)
	require.NotContains(t, rows[0], slogx.ModuleKey)

	v, ok := prefixed.Plain()([]slog.Value{slog.StringValue("a"), slog.StringValue(""), slog.StringValue("b")}, "/")
	require.True(t, ok)
	require.Equal(t, "a/b", v.String())

	_, ok = prefixed.Bracketed()(nil, ":")
	require.False(t, ok)
}
func FuzzPrefixedFormatter(f *testing.F) {
	f.Add("a", "b")
	f.Fuzz(func(t *testing.T, a, b string) {
		v, ok := prefixed.Plain()([]slog.Value{slog.StringValue(a), slog.StringValue(b)}, ":")
		require.Equal(t, a != "" || b != "", ok)
		require.Equal(t, slog.KindString, v.Kind())
	})
}
