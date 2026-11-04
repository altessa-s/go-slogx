// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/masking"
	"github.com/altessa-s/go-slogx/mask"
)

// These test-only adapters retain the old regression tables while exercising
// the new validated constructor and rule values. No compatibility API is shipped.
func portedHandler(t *testing.T, next slog.Handler, opts ...masking.Option) slog.Handler {
	t.Helper()

	h, err := masking.New(next, opts...)
	require.NoError(t, err)

	return h
}
func fieldOption(name string, f mask.Func) masking.Option {
	return masking.WithRules(mask.Rule{Field: name, Mask: f})
}
func patternOption(pattern string, f mask.Func) masking.Option {
	return masking.WithRules(mask.Rule{Pattern: pattern, Mask: f})
}
func fullMask() mask.Func          { return mask.Full() }
func fixedMask(s string) mask.Func { return mask.Fixed(s) }

var invalidRule = mask.ErrInvalidRule
