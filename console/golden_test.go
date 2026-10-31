// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console_test

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/console"
	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestGolden(t *testing.T) {
	t.Parallel()
	for _, color := range []bool{false, true} {
		var b bytes.Buffer
		h := mustNew(t, &b, console.WithColor(color))
		record := slog.NewRecord(time.Time{}, slog.LevelInfo, "ready", 0)
		record.AddAttrs(slog.String("key", "value"))
		require.NoError(t, h.Handle(t.Context(), record))
		name := "console/plain.golden"
		if color {
			name = "console/color.golden"
		}
		slogxtest.Golden(t, name, b.Bytes(), false)
	}
}
