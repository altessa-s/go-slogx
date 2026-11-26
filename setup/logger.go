// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"

	"github.com/altessa-s/go-slogx/bufwriter"
	"github.com/altessa-s/go-slogx/console"
	"github.com/altessa-s/go-slogx/handler/leveled"
	"github.com/altessa-s/go-slogx/handler/masking"
	"github.com/altessa-s/go-slogx/handler/prefixed"
	"github.com/altessa-s/go-slogx/mask"

	coremaps "github.com/altessa-s/go-atlas-core/collections/maps"
)

// Logger owns a logger's level authority and optional byte buffer.
type Logger struct {
	logger *slog.Logger
	level  *slog.LevelVar
	buffer *bufwriter.Writer
}

// New composes a logger over w. Configuration is validated even at LevelOff.
func New(w io.Writer, opts ...Option) (*Logger, error) {
	o := newOptions(opts...)

	terminal := false
	if file, ok := w.(*os.File); ok {
		terminal = isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
	}

	if w == nil || o.bufferSize < 0 || o.bufferBypassTimeout <= 0 || o.format > JSON || o.color > ColorNever {
		return nil, errors.New("setup: invalid writer or options")
	}
	// Validate rules before starting a worker or invoking a custom sink.
	maskOpts := []masking.Option{
		masking.WithRules(o.maskRules...), masking.WithDefaults(o.maskDefaults),
		masking.WithDefaultMask(mask.Fixed(o.maskString)), masking.WithMessageMask(o.messageMask),
	}
	if _, err := masking.New(slog.DiscardHandler, maskOpts...); err != nil {
		return nil, err
	}

	l := &Logger{level: new(slog.LevelVar)}
	l.level.Set(o.level)
	levelOpts := []leveled.Option{leveled.WithDefaultLevel(l.level), leveled.WithSubsystemKey(o.subsystemKey), leveled.WithSubsystemLevels(o.subsystemLevels)}

	levelPolicy, err := leveled.New(slog.DiscardHandler, levelOpts...)
	if err != nil {
		return nil, err
	}

	floor := levelPolicy.MinLevel()

	var output *bufferOutput

	if o.bufferSize > 0 {
		buffer, bufferErr := bufwriter.New(w, bufwriter.WithSize(o.bufferSize), bufwriter.WithOnError(o.onError))
		if bufferErr != nil {
			return nil, bufferErr
		}

		l.buffer = buffer
		output = &bufferOutput{buffer: buffer, format: make(chan struct{}, 1)}
		w = output
	}

	success := false
	defer func() {
		if !success && l.buffer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), o.bufferBypassTimeout)
			defer cancel()

			_, _ = l.buffer.Close(ctx)
		}
	}()

	var sink slog.Handler
	switch {
	case o.sink != nil:
		sink = o.sink(w, &slog.HandlerOptions{Level: floor, AddSource: o.addSource})
	case o.format == JSON:
		sink = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: floor, AddSource: o.addSource})
	default:
		color := o.color == ColorAlways
		if o.color == ColorAuto {
			color = terminal
		}

		sink, err = console.New(w, console.WithLevel(floor), console.WithColor(color),
			console.WithAddSource(o.addSource), console.WithTimeFormat(o.timeFormat), console.WithPrefixKey(o.subsystemKey))
	}

	if err != nil {
		return nil, err
	}

	if sink == nil {
		return nil, errors.New("setup: custom sink returned nil")
	}

	if l.buffer != nil {
		sink = &buffered{next: sink, output: output, level: o.bufferBypassLevel, timeout: o.bufferBypassTimeout}
	}

	if o.format == Text {
		sink, err = prefixed.New(sink, prefixed.WithKey(o.subsystemKey))
		if err != nil {
			return nil, err
		}
	}

	sink, err = masking.New(sink, maskOpts...)
	if err != nil {
		return nil, err
	}

	sink, err = leveled.New(sink, levelOpts...)
	if err != nil {
		return nil, err
	}

	l.logger = slog.New(sink)

	metadata := []slog.Attr{}
	if o.appName != "" {
		metadata = append(metadata, slog.String("name", o.appName))
	}

	if o.appVersion != "" {
		metadata = append(metadata, slog.String("version", o.appVersion))
	}

	if o.serviceID != "" {
		metadata = append(metadata, slog.String("service_id", o.serviceID))
	}

	if len(metadata) > 0 {
		l.logger = l.logger.With(slog.GroupAttrs(o.appGroup, metadata...))
	}

	if len(o.tags) > 0 {
		l.logger = l.logger.With(coremaps.ToSortedKeyValueSlice(o.tags)...)
	}

	success = true

	return l, nil
}

// Logger returns the composed standard logger.
func (l *Logger) Logger() *slog.Logger { return l.logger }

// SetLevel atomically updates the default threshold.
func (l *Logger) SetLevel(level slog.Level) { l.level.Set(level) }

// Level returns the current default threshold.
func (l *Logger) Level() slog.Level { return l.level.Level() }

// Shutdown drains the logger's byte buffer and reports delivery statistics.
func (l *Logger) Shutdown(ctx context.Context) (bufwriter.Stats, error) {
	if l.buffer == nil {
		return bufwriter.Stats{}, nil
	}

	return l.buffer.Close(ctx)
}
