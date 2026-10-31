// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx"
)

func TestL5NilContextAppends(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		slogx.AppendField(nil, "key", "value")
		slogx.AppendFields(nil, slogx.Fields{{Key: "key", Value: "value"}})
	})
	require.Nil(t, slogx.FieldsFromContext(nil))
}
