// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pathcache

import "sync"

// Cache stores bounded path decisions. Its zero value is usable.
type Cache[T any] struct {
	mu      sync.RWMutex
	entries map[string]T
}

// Get returns a cached decision.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.entries[key]
	return v, ok
}

// Put stores a decision, resetting the epoch at 4096 entries.
// Paths longer than 256 bytes are deliberately not retained.
func (c *Cache[T]) Put(key string, v T) {
	if len(key) > maxPathBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil || len(c.entries) >= maxEntries {
		c.entries = make(map[string]T)
	}
	c.entries[key] = v
}

const maxPathBytes = 256
const maxEntries = 4096
