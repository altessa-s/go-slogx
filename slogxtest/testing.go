// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogxtest

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas-core/runtime/leakcheck"
)

// Conformance runs the standard library handler contract suite.
func Conformance(t *testing.T, newHandler func(*testing.T) slog.Handler, result func(*testing.T) map[string]any) {
	t.Helper()
	slogtest.Run(t, newHandler, result)
}

// GoroutineLeaks checks the runtime's goroutine-leak profile.
func GoroutineLeaks() error { return leakcheck.Check() }

// Golden compares output to a repository-root testdata file.
// Set update explicitly from the caller's test flag to replace the fixture.
func Golden(t testing.TB, name string, got []byte, update bool) {
	t.Helper()
	root, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(root)
		require.NotEqual(t, root, parent, "repository root not found")
		root = parent
	}
	require.True(t, filepath.IsLocal(name), "golden name must be local")
	file := filepath.Join(root, "testdata", name)
	if update {
		require.NoError(t, os.MkdirAll(filepath.Dir(file), goldenDirMode))
		require.NoError(t, os.WriteFile(file, got, goldenFileMode))
	}
	expected, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, string(expected), string(got))
}

const goldenDirMode = 0750
const goldenFileMode = 0600
