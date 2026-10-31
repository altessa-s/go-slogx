// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package leveled_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestConformance(t *testing.T) {
	t.Parallel()
	var recorder *slogxtest.Recorder
	slogxtest.Conformance(t, func(t *testing.T) slog.Handler { recorder = slogxtest.NewRecorder(); return mustNew(t, recorder) }, func(t *testing.T) map[string]any {
		records, err := recorder.Records()
		require.NoError(t, err)
		require.Len(t, records, 1)
		return records[0]
	})
}
