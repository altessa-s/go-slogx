// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking

//go:generate optgen generate --keep-empty-strings --no-default-modifiers
//go:generate sh ../../docs/generated-header.sh options_gen.go
import "github.com/altessa-s/go-slogx/mask"

type options struct {
	rules         []mask.Rule `optgen:"append"`
	defaults      bool        `optval:"param"`
	defaultMask   mask.Func
	nested        bool `optval:"param"`
	caseSensitive bool `optval:"param"`
}
