// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/console"
)

func mustNew(t testing.TB, writer io.Writer, opts ...console.Option) *console.Handler {
	t.Helper()
	result, err := console.New(writer, opts...)
	require.NoError(t, err)
	return result
}
