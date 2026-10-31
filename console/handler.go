// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/altessa-s/go-slogx"
)

// Handler emits one quoted text line per record, with optional per-instance SGR.
type Handler struct {
	writer   io.Writer
	mu       *sync.Mutex
	opts     *options
	groups   []string
	captured []string
}

// New constructs a console sink. A nil writer returns an error.
func New(w io.Writer, opts ...Option) (*Handler, error) {
	if w == nil {
		return nil, errors.New("console: nil writer")
	}
	o := newOptions(opts...)
	if o.level == nil {
		o.level = slog.LevelInfo
	}
	o.keyStyles = maps.Clone(o.keyStyles)
	return &Handler{writer: w, mu: new(sync.Mutex), opts: o}, nil
}

// Enabled reads the supplied leveler on every call.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	threshold := h.opts.level.Level()
	return threshold != slogx.LevelOff && level >= threshold
}

// Handle formats every value on the calling goroutine before writing bytes.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	parts := make([]string, 0, 4+len(h.captured)+r.NumAttrs())
	if !r.Time.IsZero() {
		parts = h.appendAttr(parts, nil, slog.Time(slog.TimeKey, r.Time))
	}
	parts = h.appendAttr(parts, nil, slog.Any(slog.LevelKey, r.Level))
	if h.opts.addSource && r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		parts = h.appendAttr(parts, nil, slog.Any(slog.SourceKey, &slog.Source{Function: frame.Function, File: frame.File, Line: frame.Line}))
	}
	parts = h.appendAttr(parts, nil, slog.String(slog.MessageKey, r.Message))
	parts = append(parts, h.captured...)
	r.Attrs(func(a slog.Attr) bool { parts = h.appendAttr(parts, h.groups, a); return true })
	line := []byte(strings.Join(parts, " ") + "\n")
	h.mu.Lock()
	defer h.mu.Unlock()
	n, err := h.writer.Write(line)
	if err == nil && n != len(line) {
		return io.ErrShortWrite
	}
	return err
}
func (h *Handler) appendAttr(parts, groups []string, a slog.Attr) []string {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() != slog.KindGroup && h.opts.replaceAttr != nil {
		a = h.opts.replaceAttr(slices.Clone(groups), a)
		a.Value = a.Value.Resolve()
	}
	if a.Equal(slog.Attr{}) {
		return parts
	}
	if a.Value.Kind() == slog.KindGroup {
		if a.Key != "" {
			groups = slices.Concat(groups, []string{a.Key})
		}
		for _, child := range a.Value.Group() {
			parts = h.appendAttr(parts, groups, child)
		}
		return parts
	}
	key := a.Key
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}
	value := a.Value.String()
	if a.Value.Kind() == slog.KindTime {
		value = a.Value.Time().Format(h.opts.timeFormat)
	}
	text := strconv.Quote(key) + "=" + strconv.Quote(value)
	if h.opts.color {
		style := h.opts.keyStyles[a.Key]
		if style == Plain && a.Key == slogx.ErrorKey {
			style = Red
		}
		if style == Plain && a.Key == slog.LevelKey {
			style = Cyan
		}
		if style == Plain && a.Key == h.opts.prefixKey {
			style = Blue
		}
		switch style {
		default: // Unknown styles emit no escape sequence.
		case Red, Green, Yellow, Blue, Cyan:
			text = fmt.Sprintf("\x1b[%dm%s\x1b[0m", style, text)
		}
	}
	return append(parts, text)
}

// WithAttrs formats captured attributes immediately, as slog handlers require.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.captured = slices.Clone(h.captured)
	for _, a := range attrs {
		c.captured = h.appendAttr(c.captured, h.groups, a)
	}
	return &c
}

// WithGroup qualifies future attributes; an empty name returns the receiver.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.groups = slices.Concat(h.groups, []string{name})
	return &c
}
