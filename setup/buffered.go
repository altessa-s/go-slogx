// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/altessa-s/go-slogx/bufwriter"
)

// bufferOutput captures bytes during a priority record's formatting. The
// format gate keeps records separate across all derived handlers.
type bufferOutput struct {
	format  chan struct{}
	mu      sync.Mutex
	buffer  *bufwriter.Writer
	capture *[]byte
}

func (o *bufferOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.capture != nil {
		*o.capture = append(*o.capture, p...)
		return len(p), nil
	}

	return o.buffer.Write(p)
}

type buffered struct {
	next    slog.Handler
	output  *bufferOutput
	level   slog.Level
	timeout time.Duration
}

func (h *buffered) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *buffered) Handle(ctx context.Context, r slog.Record) error {
	o := h.output
	if r.Level < h.level {
		o.format <- struct{}{}
		defer func() { <-o.format }()

		return h.next.Handle(ctx, r)
	}

	wait, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	select {
	case o.format <- struct{}{}:
	case <-wait.Done():
		return o.buffer.DropPriority(wait.Err())
	}

	if err := wait.Err(); err != nil {
		<-o.format
		return o.buffer.DropPriority(err)
	}

	var data []byte

	err := h.capture(ctx, r, &data)
	if len(data) == 0 {
		return err
	}

	// Formatters can lock their writers internally. Wait only after releasing
	// formatting locks, so a stalled sink cannot block other loggers.
	return errors.Join(err, o.buffer.WritePriority(wait, data))
}

func (h *buffered) capture(ctx context.Context, r slog.Record, data *[]byte) error {
	o := h.output
	o.mu.Lock()
	o.capture = data
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.capture = nil
		o.mu.Unlock()
		<-o.format
	}()

	return h.next.Handle(ctx, r)
}

func (h *buffered) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.next = h.next.WithAttrs(attrs)

	return &c
}

func (h *buffered) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	c := *h
	c.next = h.next.WithGroup(name)

	return &c
}
