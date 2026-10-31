// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"sync"

	"github.com/altessa-s/go-slogx"
)

// Handler emits one quoted text line per record, with optional per-instance SGR.
type Handler struct {
	writer   io.Writer
	mu       *sync.Mutex
	opts     *options
	groups   []string
	captured []byte
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
	buffer := buffers.Get().(*[]byte) //nolint:errcheck // the pool stores only byte slice pointers
	parts := (*buffer)[:0]
	defer func() {
		if cap(parts) <= maxPooledCapacity {
			*buffer = parts[:0]
			buffers.Put(buffer)
		}
	}()
	if !r.Time.IsZero() {
		parts = h.appendAttr(parts, nil, slog.Time(slog.TimeKey, r.Time))
	}
	parts = h.appendAttr(parts, nil, slog.Any(slog.LevelKey, r.Level))
	if h.opts.addSource && r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		parts = h.appendAttr(parts, nil, slog.Any(slog.SourceKey, &slog.Source{Function: frame.Function, File: frame.File, Line: frame.Line}))
	}
	parts = h.appendAttr(parts, nil, slog.String(slog.MessageKey, r.Message))
	if len(h.captured) > 0 {
		if len(parts) > 0 {
			parts = append(parts, ' ')
		}
		parts = append(parts, h.captured...)
	}
	r.Attrs(func(a slog.Attr) bool { parts = h.appendAttr(parts, h.groups, a); return true })
	parts = append(parts, '\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	n, err := h.writer.Write(parts)
	if err == nil && n != len(parts) {
		return io.ErrShortWrite
	}
	return err
}
func (h *Handler) appendAttr(parts []byte, groups []string, a slog.Attr) []byte {
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

	if len(parts) > 0 {
		parts = append(parts, ' ')
	}
	style := Plain
	if h.opts.color {
		style = h.opts.keyStyles[a.Key]
		if style == Plain {
			switch a.Key {
			case slogx.ErrorKey:
				style = Red
			case slog.LevelKey:
				style = Cyan
			case h.opts.prefixKey:
				style = Blue
			}
		}
	}
	colored := style == Red || style == Green || style == Yellow || style == Blue || style == Cyan
	if colored {
		parts = append(parts, "\x1b["...)
		parts = strconv.AppendInt(parts, int64(style), 10)
		parts = append(parts, 'm')
	}
	// Quote each segment into one key without allocating a joined path.
	parts = append(parts, '"')
	for _, group := range groups {
		start := len(parts)
		parts = strconv.AppendQuote(parts, group)
		parts = append(parts[:start], parts[start+1:len(parts)-1]...)
		parts = append(parts, '.')
	}
	start := len(parts)
	parts = strconv.AppendQuote(parts, a.Key)
	parts = append(parts[:start], parts[start+1:]...)
	parts = append(parts, '=')
	value := a.Value.String()
	if a.Value.Kind() == slog.KindTime {
		value = a.Value.Time().Format(h.opts.timeFormat)
	} else if a.Value.Kind() == slog.KindAny {
		if source, ok := a.Value.Any().(*slog.Source); ok && source != nil {
			value = source.File + ":" + strconv.Itoa(source.Line)
		}
	}
	parts = strconv.AppendQuote(parts, value)
	if colored {
		parts = append(parts, "\x1b[0m"...)
	}
	return parts
}

const maxPooledCapacity = 64 * 1024

var buffers = sync.Pool{New: func() any { b := make([]byte, 0, 1024); return &b }}

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
