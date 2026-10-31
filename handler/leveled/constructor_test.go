// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package leveled_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/leveled"
)

func mustNew(t testing.TB, next slog.Handler, opts ...leveled.Option) *leveled.Handler {
	t.Helper()
	result, err := leveled.New(next, opts...)
	require.NoError(t, err)
	return result
}

func TestRejectNilHandler(t *testing.T) {
	t.Parallel()
	_, err := leveled.New(nil)
	require.Error(t, err)
}
