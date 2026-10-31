// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pathcache

import (
	"hash/maphash"
	"sync"
)

// Cache stores bounded path decisions. Its zero value is usable.
type Cache[T any] struct {
	shards [16]shard[T]
}
type shard[T any] struct {
	mu      sync.RWMutex
	entries map[string]T
}

// Get returns a cached decision.
func (c *Cache[T]) Get(key string) (T, bool) {
	shard := &c.shards[maphash.Comparable(cacheSeed, key)%uint64(len(c.shards))]
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	v, ok := shard.entries[key]
	return v, ok
}

// Put stores a decision, resetting each shard at 256 entries (4096 total).
// Paths longer than 256 bytes are deliberately not retained.
func (c *Cache[T]) Put(key string, v T) {
	if len(key) > maxPathBytes {
		return
	}
	shard := &c.shards[maphash.Comparable(cacheSeed, key)%uint64(len(c.shards))]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if shard.entries == nil || len(shard.entries) >= maxEntries/len(c.shards) {
		shard.entries = make(map[string]T)
	}
	shard.entries[key] = v
}

const maxPathBytes = 256
const maxEntries = 4096

var cacheSeed = maphash.MakeSeed()
