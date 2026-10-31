// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/setup"
)

type credentials struct{ Password string }
type embeddedValue struct{ credentials }
type embeddedPointer struct{ *credentials }

func TestUnexportedEmbeddingsFailClosed(t *testing.T) {
	t.Parallel()
	for _, format := range []setup.Format{setup.JSON, setup.Text} {
		for _, input := range []any{
			embeddedValue{credentials{"canary"}},
			embeddedPointer{&credentials{"canary"}},
			&embeddedValue{credentials{"canary"}},
			&embeddedPointer{&credentials{"canary"}},
		} {
			t.Run(fmt.Sprintf("%v/%T", format, input), func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				logger, err := setup.New(&out, setup.WithFormat(format), setup.WithMaskDefaults(true))
				require.NoError(t, err)
				logger.Logger().Info("record", "payload", input)
				require.NotContains(t, out.String(), "canary")
				require.Contains(t, out.String(), "********")
				require.NoError(t, logger.Shutdown(t.Context()))
			})
		}
	}
}

func TestMatchedContainersAreBounded(t *testing.T) {
	t.Parallel()
	cycle := map[string]any{}
	cycle["self"] = cycle
	slice := make([]any, 1)
	slice[0] = slice
	var pointer any
	pointer = &pointer
	var deep any = "canary"
	for range 20 {
		deep = []any{deep}
	}
	for _, format := range []setup.Format{setup.JSON, setup.Text} {
		for _, input := range []any{cycle, slice, pointer, deep, make([]string, 300)} {
			for _, nested := range []bool{false, true} {
				t.Run(fmt.Sprintf("%v/%T/%v", format, input, nested), func(t *testing.T) {
					t.Parallel()
					var out bytes.Buffer
					logger, err := setup.New(&out, setup.WithFormat(format), setup.WithMaskDefaults(true))
					require.NoError(t, err)
					key, value := "password", input
					if nested {
						key, value = "payload", struct{ Password any }{input}
					}
					logger.Logger().Info("record", key, value)
					require.True(t, strings.Contains(out.String(), "masking:") || strings.Contains(out.String(), "********"))
					require.NotContains(t, out.String(), "canary")
					require.NoError(t, logger.Shutdown(t.Context()))
				})
			}
		}
	}
}
