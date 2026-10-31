// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogxtest

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sync"
)

type recording struct {
	mu    sync.Mutex
	bytes bytes.Buffer
}

func (r *recording) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bytes.Write(p)
}

// Recorder captures synchronously resolved records using the standard JSON sink.
// Its zero value is not usable; construct it with NewRecorder.
type Recorder struct {
	state   *recording
	handler slog.Handler
}

// NewRecorder creates an all-levels, concurrency-safe recording handler.
func NewRecorder() *Recorder {
	state := new(recording)
	return &Recorder{state: state, handler: slog.NewJSONHandler(state, &slog.HandlerOptions{Level: slog.Level(math.MinInt32)})}
}

// Enabled admits all records supported by the recording sink.
func (r *Recorder) Enabled(ctx context.Context, level slog.Level) bool {
	return r.handler.Enabled(ctx, level)
}

// Handle captures the record on the calling goroutine.
func (r *Recorder) Handle(ctx context.Context, record slog.Record) error {
	return r.handler.Handle(ctx, record)
}

// WithAttrs captures attributes for a derived recorder.
func (r *Recorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Recorder{state: r.state, handler: r.handler.WithAttrs(attrs)}
}

// WithGroup derives a recorder sharing the captured records.
func (r *Recorder) WithGroup(name string) slog.Handler {
	if name == "" {
		return r
	}
	return &Recorder{state: r.state, handler: r.handler.WithGroup(name)}
}

// Records returns independent decoded snapshots in write order.
func (r *Recorder) Records() ([]map[string]any, error) {
	r.state.mu.Lock()
	data := bytes.Clone(r.state.bytes.Bytes())
	r.state.mu.Unlock()
	decoder := json.NewDecoder(bytes.NewReader(data))
	var out []map[string]any
	for decoder.More() {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

// Reset clears recorded output, including output from derived handlers.
func (r *Recorder) Reset() { r.state.mu.Lock(); r.state.bytes.Reset(); r.state.mu.Unlock() }

// Discard returns a logger that drops every record.
func Discard() *slog.Logger { return slog.New(slog.DiscardHandler) }
