// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter

// DefaultSize is the maximum number of queued writes.
//
//go:generate optgen generate --keep-empty-strings --no-default-modifiers
const DefaultSize = 100

type options struct {
	size    int `optgen:"default=DefaultSize"`
	onError func(error)
}
