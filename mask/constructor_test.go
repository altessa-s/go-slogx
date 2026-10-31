// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/mask"
)

func mustNew(t testing.TB, opts ...mask.Option) *mask.Catalog {
	t.Helper()
	result, err := mask.NewCatalog(opts...)
	require.NoError(t, err)
	return result
}

func TestRejectInvalidCatalogEntries(t *testing.T) {
	t.Parallel()
	for _, option := range []mask.Option{mask.WithMasks(map[string]mask.Func{"nil": nil}), mask.WithMasks(map[string]mask.Func{"": mask.Full()}), mask.WithFactories(map[string]mask.Factory{"nil": nil}), mask.WithFactories(map[string]mask.Factory{"": func(mask.Params) (mask.Func, error) { return mask.Full(), nil }})} {
		_, err := mask.NewCatalog(option)
		require.ErrorIs(t, err, mask.ErrParams)
	}
}
