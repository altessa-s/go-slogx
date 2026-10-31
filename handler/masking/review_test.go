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
	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestCacheSeparatesLiteralAndGroupedKeys(t *testing.T) {
	t.Parallel()
	for _, captured := range []bool{false, true} {
		for _, groupedFirst := range []bool{false, true} {
			r := slogxtest.NewRecorder()
			h, err := masking.New(r, masking.WithNested(true), masking.WithRules(mask.Rule{Field: "password", Mask: mask.Fixed("safe")}))
			require.NoError(t, err)
			log := slog.New(h)
			literal := func() {
				if captured {
					log.With("account.password", "visible").Info("literal")
				} else {
					log.Info("literal", "account.password", "visible")
				}
			}
			grouped := func() {
				if captured {
					log.WithGroup("account").With("password", "canary").Info("grouped")
				} else {
					log.Info("grouped", slog.Group("account", "password", "canary"))
				}
			}
			if groupedFirst {
				grouped()
				literal()
			} else {
				literal()
				grouped()
			}
			rows, err := r.Records()
			require.NoError(t, err)
			literalIndex, groupedIndex := 0, 1
			if groupedFirst {
				literalIndex, groupedIndex = 1, 0
			}
			require.Equal(t, "visible", rows[literalIndex]["account.password"])
			require.Equal(t, map[string]any{"password": "safe"}, rows[groupedIndex]["account"])
		}
	}
}
