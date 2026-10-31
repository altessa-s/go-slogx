// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/mask"
)

func TestCatalogPatternRecoversReplacementPanic(t *testing.T) {
	t.Parallel()
	c, err := mask.NewCatalog(mask.WithMasks(map[string]mask.Func{
		"panicking": func(string) string { panic("canary") },
	}))
	require.NoError(t, err)
	f, err := c.Resolve("pattern", mask.Params{"pattern": "secret", "replacementType": "panicking"})
	require.NoError(t, err)
	require.Equal(t, "********", f("secret"))
	require.Equal(t, "public", f("public"))
}
