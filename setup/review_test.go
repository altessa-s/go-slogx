// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/bufwriter"
	"github.com/altessa-s/go-slogx/mask"
	"github.com/altessa-s/go-slogx/setup"
)

func TestH2DefaultsPreserveJSON(t *testing.T) {
	t.Parallel()
	for _, value := range []any{[]int{1, 2}, [2]int{1, 2}, map[string]any{"z": []int{1, 2}, "a": "ok"}, struct {
		Name  string `json:"name"`
		Empty string `json:"empty,omitempty"`
	}{Name: "ok"}} {
		var plain, masked bytes.Buffer
		a, err := setup.New(&plain, setup.WithFormat(setup.JSON))
		require.NoError(t, err)
		b, err := setup.New(&masked, setup.WithFormat(setup.JSON), setup.WithMaskDefaults(true))
		require.NoError(t, err)
		r := slog.Record{}
		r.AddAttrs(slog.Any("payload", value))
		require.NoError(t, a.Logger().Handler().Handle(t.Context(), r))
		require.NoError(t, b.Logger().Handler().Handle(t.Context(), r))
		require.Equal(t, plain.String(), masked.String())
	}
}
func TestH3PanickingMaskFailsClosed(t *testing.T) {
	t.Parallel()
	for _, value := range []any{"private", map[string]string{"secret": "private"}, []map[string]string{{"secret": "private"}}} {
		var out bytes.Buffer
		l, err := setup.New(&out, setup.WithFormat(setup.JSON), setup.WithMaskRules(mask.Rule{Field: "secret", Mask: func(string) string { panic("private panic") }}))
		require.NoError(t, err)
		key := "payload"
		if _, ok := value.(string); ok {
			key = "secret"
		}
		require.NotPanics(t, func() { l.Logger().Info("record", key, value) })
		require.NotContains(t, out.String(), "private")
		require.Contains(t, out.String(), "********")
	}
}
func TestM2RecordAfterShutdown(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var reports []error
	l, err := setup.New(&out, setup.WithBufferSize(8), setup.WithOnError(func(err error) { reports = append(reports, err) }))
	require.NoError(t, err)
	require.NoError(t, l.Shutdown(t.Context()))
	l.Logger().Info("after shutdown")
	require.Contains(t, out.String(), "after shutdown")
	require.Len(t, reports, 1)
	require.ErrorIs(t, reports[0], bufwriter.ErrClosed)
}
func TestL3FailedSinkClosesWorker(t *testing.T) {
	t.Parallel()
	var writer io.Writer
	var reports []error
	_, err := setup.New(io.Discard, setup.WithBufferSize(8), setup.WithOnError(func(err error) { reports = append(reports, err) }), setup.WithSink(func(w io.Writer, _ *slog.HandlerOptions) slog.Handler { writer = w; return nil }))
	require.EqualError(t, err, "setup: custom sink returned nil")
	buffer, ok := writer.(*bufwriter.Writer)
	require.True(t, ok)
	_, writeErr := buffer.Write([]byte("after failed constructor"))
	require.NoError(t, writeErr)
	require.NoError(t, buffer.Close(t.Context()))
	require.Len(t, reports, 1)
	require.ErrorIs(t, reports[0], bufwriter.ErrClosed)
}

func TestMaskingUnsupportedMapsAndOmittedFields(t *testing.T) {
	t.Parallel()
	for _, format := range []setup.Format{setup.JSON, setup.Text} {
		for _, defaults := range []bool{false, true} {
			for _, value := range []any{
				map[int]any{1: map[string]string{"password": "canary"}},
				struct{ Values map[int]string }{map[int]string{1: "canary"}},
				struct {
					Password string `json:"-"`
				}{"canary"},
				struct {
					Hidden string `json:"-"`
					Public string
				}{"canary", "visible"},
			} {
				var out bytes.Buffer
				l, err := setup.New(&out, setup.WithFormat(format),
					setup.WithMaskDefaults(defaults), setup.WithMaskRules(mask.Rule{Field: "password"}))
				require.NoError(t, err)
				l.Logger().Info("record", "payload", value)
				require.NotContains(t, out.String(), "canary")
				require.Contains(t, out.String(), "record")
			}
		}
	}
}
