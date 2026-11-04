// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package handlerbase_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/internal/handlerbase"
)

func TestImmutableGroups(t *testing.T) {
	t.Parallel()

	b := handlerbase.NewBase(slog.DiscardHandler)
	require.Equal(t, slog.DiscardHandler, b.Inner())
	require.False(t, b.Enabled(t.Context(), slog.LevelInfo))
	require.Empty(t, b.Groups())
	require.Equal(t, "key", b.GroupPath("key"))
	c := b.GroupBase("outer")
	d := c.GroupBase("inner")
	require.Equal(t, "outer.inner.key", d.GroupPath("key"))
	require.Equal(t, []string{"outer"}, c.Groups())

	e := d.AttrsBase([]slog.Attr{slog.String("key", "value")})
	require.Equal(t, d.Groups(), e.Groups())
	require.Panics(t, func() { handlerbase.NewBase(nil) })
}
