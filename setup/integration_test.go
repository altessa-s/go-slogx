//go:build integration

// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx"
	"github.com/altessa-s/go-slogx/bufwriter"
	"github.com/altessa-s/go-slogx/mask"
	"github.com/altessa-s/go-slogx/setup"
)

func TestIntegrationFile(t *testing.T) {
	t.Parallel()
	file, err := os.Create(filepath.Join(t.TempDir(), "log.json"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	l, err := setup.New(file, setup.WithFormat(setup.JSON), setup.WithBufferSize(16), setup.WithMaskDefaults(true), setup.WithMaskRules(mask.Rule{Field: "private", Mask: mask.Fixed("safe")}), setup.WithAppName("integration"), setup.WithSubsystemLevels(map[string]slog.Level{"verbose": slog.LevelDebug}))
	require.NoError(t, err)
	l.Logger().Debug("override", slogx.Module("verbose"), "password", "canary", "private", "canary")
	l.SetLevel(slog.LevelDebug)
	l.Logger().Debug("runtime")
	require.NoError(t, l.Shutdown(t.Context()))
	_, err = file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	var rows []map[string]any
	for scanner.Scan() {
		var row map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &row))
		rows = append(rows, row)
		require.NotContains(t, scanner.Text(), "canary")
	}
	require.NoError(t, scanner.Err())
	require.Len(t, rows, 2)
	require.Equal(t, "verbose", rows[0]["subsystem"])
	require.Equal(t, "safe", rows[0]["private"])
	require.Equal(t, "integration", rows[0]["app"].(map[string]any)["name"])
}
func TestIntegrationColorAutoPipe(t *testing.T) {
	t.Parallel()
	read, write, err := os.Pipe()
	require.NoError(t, err)
	defer read.Close()
	l, err := setup.New(write, setup.WithBufferSize(8), setup.WithColor(setup.ColorAuto))
	require.NoError(t, err)
	l.Logger().Info("pipe")
	require.NoError(t, l.Shutdown(t.Context()))
	require.NoError(t, write.Close())
	out, err := io.ReadAll(read)
	require.NoError(t, err)
	require.NotContains(t, string(out), "\x1b")
}

type orderedSink struct {
	mu    sync.Mutex
	last  [64]int
	count int
	err   error
}

func (s *orderedSink) Write(p []byte) (int, error) {
	var row struct {
		Worker   int `json:"worker"`
		Sequence int `json:"sequence"`
	}
	if err := json.Unmarshal(p, &row); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if row.Sequence != s.last[row.Worker] {
		s.err = errors.New("out-of-order record")
	}
	s.last[row.Worker]++
	s.count++
	return len(p), nil
}
func TestIntegrationConcurrentSoak(t *testing.T) {
	t.Parallel()
	sink := new(orderedSink)
	// Capacity holds the entire workload: overflow ordering is explicitly unspecified.
	l, err := setup.New(sink, setup.WithFormat(setup.JSON), setup.WithBufferSize(64*10000), setup.WithMaskDefaults(true), setup.WithAppName(""), setup.WithAppVersion(""))
	require.NoError(t, err)
	var wg sync.WaitGroup
	for worker := range 64 {
		wg.Go(func() {
			log := l.Logger().With(slogx.Module("soak"), slog.Int("worker", worker))
			for sequence := range 10000 {
				log.Info("record", "sequence", sequence)
			}
		})
	}
	wg.Wait()
	require.NoError(t, l.Shutdown(t.Context()))
	require.NoError(t, sink.err)
	require.Equal(t, 640000, sink.count)
	for _, count := range sink.last {
		require.Equal(t, 10000, count)
	}
}

type panicOnce struct {
	writer io.Writer
	once   sync.Once
}

func (w *panicOnce) Write(p []byte) (int, error) {
	panics := false
	w.once.Do(func() { panics = true })
	if panics {
		panic("private")
	}
	return w.writer.Write(p)
}
func TestIntegrationSignalShutdown(t *testing.T) {
	if os.Getenv("SLOGX_TEST_CHILD") == "1" {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM)
		defer signal.Stop(signals)
		l, err := setup.New(&panicOnce{writer: os.Stdout}, setup.WithFormat(setup.JSON), setup.WithBufferSize(128), setup.WithAppName(""), setup.WithAppVersion(""))
		if err != nil {
			os.Exit(2)
		}
		l.Logger().Info("panic-write")
		l.Logger().Error("after-panic")
		for i := range 100 {
			l.Logger().Info("drain", "sequence", i)
		}
		fmt.Fprintln(os.Stderr, "READY")
		<-signals
		if err := l.Shutdown(t.Context()); err != nil && !errors.Is(err, bufwriter.ErrPanicked) {
			os.Exit(3)
		}
		os.Exit(0)
	}
	t.Parallel()
	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestIntegrationSignalShutdown$")
	command.Env = append(os.Environ(), "SLOGX_TEST_CHILD=1")
	var output bytes.Buffer
	command.Stdout = &output
	stderr, err := command.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, command.Start())
	t.Cleanup(func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	scanner := bufio.NewScanner(stderr)
	require.True(t, scanner.Scan())
	require.Equal(t, "READY", scanner.Text())
	require.NoError(t, command.Process.Signal(syscall.SIGTERM))
	require.NoError(t, command.Wait())
	require.Equal(t, 101, strings.Count(output.String(), "\n"))
	require.Contains(t, output.String(), "after-panic")
	require.Contains(t, output.String(), `"sequence":99`)
	require.NotContains(t, output.String(), "private")
}
