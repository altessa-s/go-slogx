// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package prefixed_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/prefixed"
	"github.com/altessa-s/go-slogx/slogxtest"
)

func TestCustomKeyAndDelimiter(t *testing.T) {
	t.Parallel()

	for _, ptr := range []*string{nil, new("/")} {
		r := slogxtest.NewRecorder()
		h := mustNew(t, r, prefixed.WithKey(ptr), prefixed.WithDelimiter(ptr))
		key := "subsystem"
		delimiter := ":"

		if ptr != nil {
			key = *ptr
			delimiter = *ptr
		}

		slog.New(h).Info("prefix", slog.String(key, "a"), slog.String(key, "b"))

		rows, err := r.Records()
		require.NoError(t, err)
		require.Equal(t, "[a"+delimiter+"b]", rows[0][key])
	}

	r := slogxtest.NewRecorder()
	slog.New(mustNew(t, r, prefixed.WithKey("module"), prefixed.WithDelimiter("/"))).Info("prefix", slog.String("module", "a"), slog.String("module", "b"))
	rows, err := r.Records()
	require.NoError(t, err)
	require.Equal(t, "[a/b]", rows[0]["module"])
}
