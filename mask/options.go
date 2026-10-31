// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask

// options configures an immutable catalog. Maps are copied at construction.
//
//go:generate optgen generate --keep-empty-strings --no-default-modifiers
type options struct {
	builtins  bool `optgen:"default=DefaultBuiltins" optval:"param"`
	masks     map[string]Func
	factories map[string]Factory
}

// DefaultBuiltins enables the standard catalog entries.
const DefaultBuiltins = true
