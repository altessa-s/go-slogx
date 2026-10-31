// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package prefixed

//go:generate optgen generate --keep-empty-strings --no-default-modifiers
//go:generate sh ../../docs/generated-header.sh options_gen.go
import (
	"log/slog"
	"strings"

	"github.com/altessa-s/go-slogx"
)

// DefaultKey identifies subsystem attributes.
const DefaultKey = slogx.ModuleKey

// DefaultDelimiter separates prefix values.
const DefaultDelimiter = ":"

// Formatter combines values; false suppresses the prefix.
type Formatter func([]slog.Value, string) (slog.Value, bool)
type options struct {
	key       string `optgen:"default=DefaultKey"`
	delimiter string `optgen:"default=DefaultDelimiter"`
	formatter Formatter
}

// Plain joins nonempty prefix values.
func Plain() Formatter {
	return func(values []slog.Value, delimiter string) (slog.Value, bool) {
		var parts []string
		for _, v := range values {
			if s := v.String(); s != "" {
				parts = append(parts, s)
			}
		}
		return slog.StringValue(strings.Join(parts, delimiter)), len(parts) > 0
	}
}

// Bracketed surrounds the joined prefixes with square brackets.
func Bracketed() Formatter {
	return func(values []slog.Value, delimiter string) (slog.Value, bool) {
		v, ok := Plain()(values, delimiter)
		if !ok {
			return v, false
		}
		return slog.StringValue("[" + v.String() + "]"), true
	}
}
