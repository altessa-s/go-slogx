// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package prefixed_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/prefixed"
)

func mustNew(t testing.TB, next slog.Handler, opts ...prefixed.Option) *prefixed.Handler {
	t.Helper()
	result, err := prefixed.New(next, opts...)
	require.NoError(t, err)
	return result
}

func TestRejectNilHandler(t *testing.T) {
	t.Parallel()
	_, err := prefixed.New(nil)
	require.Error(t, err)
}
