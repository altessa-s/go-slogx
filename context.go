// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx

import (
	"context"
	"log/slog"
)

type ctxSlog struct{}

// ctxSlog{} is the key used to store the logger in the context.

// ContextWithLogger stores the provided logger in context for retrieval via FromContext.
// Panics if logger is nil. Replaces any previously stored logger.
//
// Example:
//
//	ctx = slogx.ContextWithLogger(ctx, logger.With("request_id", id))
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		panic("logger cannot be nil")
	}

	ctx = context.WithValue(ctx, ctxSlog{}, logger)

	return ctx
}

// LoggerFromContext returns the explicitly stored logger, if present.
func LoggerFromContext(ctx context.Context) (*slog.Logger, bool) {
	if ctx == nil {
		return nil, false
	}
	l, ok := ctx.Value(ctxSlog{}).(*slog.Logger)
	return l, ok && l != nil
}

// FromContext returns the stored logger or slog.Default.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := LoggerFromContext(ctx); ok {
		return l
	}
	return slog.Default()
}

// ContextWithLoggerIfAbsent preserves the first logger stored in ctx.
func ContextWithLoggerIfAbsent(ctx context.Context, l *slog.Logger) context.Context {
	if _, ok := LoggerFromContext(ctx); ok {
		return ctx
	}
	return ContextWithLogger(ctx, l)
}

// BuildLogger creates a logger enriched with fields from context.
// If no fields in context, returns base logger unchanged.
// If base is nil, uses slog.Default().
//
// Example:
//
//	logger := slogx.BuildLogger(ctx, nil) // uses slog.Default() + context fields
func BuildLogger(ctx context.Context, base *slog.Logger) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	fields := FieldsFromContext(ctx)
	if len(fields) == 0 {
		return base
	}
	return base.With(fields.Args()...)
}

// InjectLogger creates enriched logger from base + context fields and stores in context.
// Combines BuildLogger and ContextWithLogger in one call.
// Replaces the logger previously stored in the context.
//
// Example (in interceptor):
//
//	ctx = slogx.InjectLogger(ctx, baseLogger)
func InjectLogger(ctx context.Context, base *slog.Logger) context.Context {
	enriched := BuildLogger(ctx, base)
	return ContextWithLogger(ctx, enriched)
}
