// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package pathcache remembers which attribute paths match a mask rule.
// The cache is bounded per handler and is safe for concurrent readers after
// it is built.
//
// # Usage
//
//	var cache pathcache.Cache[int]
package pathcache
