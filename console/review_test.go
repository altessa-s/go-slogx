// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/console"
)

func TestL1SourceFileLine(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	h, err := console.New(&out, console.WithAddSource(true))
	require.NoError(t, err)

	log := slog.New(h)
	_, file, line, _ := runtime.Caller(0)

	log.Info("source")
	require.Contains(t, out.String(), fmt.Sprintf("%s:%d", file, line+2))
}
