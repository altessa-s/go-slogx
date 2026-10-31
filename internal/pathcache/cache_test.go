// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pathcache_test

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/internal/pathcache"
)

func TestBoundedConcurrentCache(t *testing.T) {
	t.Parallel()
	c := new(pathcache.Cache[int])
	_, ok := c.Get("missing")
	require.False(t, ok)
	c.Put("first", 1)
	n, ok := c.Get("first")
	require.True(t, ok)
	require.Equal(t, 1, n)
	for i := range 4096 {
		c.Put(strconv.Itoa(i), i)
	}
	_, ok = c.Get("first")
	require.False(t, ok)
	key := strings.Repeat("a", 257)
	c.Put(key, 1)
	_, ok = c.Get(key)
	require.False(t, ok)
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for i := range 100 {
				c.Put(strconv.Itoa(i), i)
				_, _ = c.Get(strconv.Itoa(i))
			}
		})
	}
	wg.Wait()
}
