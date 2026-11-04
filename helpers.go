// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx

import (
	"log/slog"
	"reflect"
)

// ErrorKey is the attribute key used for error values in log records.
const ErrorKey string = "error"

// Error creates a slog.Attr for errors using the standard ErrorKey.
// Returns an empty attribute if err is nil, which slog ignores.
//
// Example:
//
//	logger.Error("failed", slogx.Error(err))
func Error(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}

	return slog.Any(ErrorKey, err)
}

// String creates a slog.Attr from a string or pointer to string.
// Returns an empty attribute (ignored by slog) if value is nil or empty.
//
// Example:
//
//	logger.Info("user", slogx.String("name", namePtr)) // nil-safe
func String[T interface{ ~string | ~*string }](key string, value T) slog.Attr {
	var val string

	switch v := any(value).(type) {
	case *string:
		if v == nil {
			return slog.Attr{}
		}

		val = *v
	case string:
		if v == "" {
			return slog.Attr{}
		}

		val = v
	default:
		rv, ok := derefNamed(value)
		if !ok || rv.String() == "" {
			return slog.Attr{}
		}

		val = rv.String()
	}

	if val == "" {
		return slog.Attr{}
	}

	return slog.Attr{Key: key, Value: slog.StringValue(val)}
}

// Int creates a slog.Attr from an int or pointer to int.
// Returns an empty attribute (ignored by slog) if value is nil.
//
// Example:
//
//	logger.Debug("processed", slogx.Int("count", countPtr)) // nil-safe
func Int[T interface{ ~int | ~*int }](key string, value T) slog.Attr {
	var val int

	switch v := any(value).(type) {
	case int:
		val = v
	case *int:
		if v == nil {
			return slog.Attr{}
		}

		val = *v
	default:
		rv, ok := derefNamed(value)
		if !ok {
			return slog.Attr{}
		}

		val = int(rv.Int())
	}

	return slog.Attr{Key: key, Value: slog.IntValue(val)}
}

// Int64 creates a slog.Attr from int64, int32, or their pointers.
// Returns an empty attribute (ignored by slog) if value is nil.
//
// Example:
//
//	logger.Info("found", slogx.Int64("user_id", userIDPtr)) // nil-safe
func Int64[T interface {
	~int64 | ~*int64 | ~int32 | ~*int32
}](key string, value T) slog.Attr {
	var val int64

	switch v := any(value).(type) {
	case int64:
		val = v
	case *int64:
		if v == nil {
			return slog.Attr{}
		}

		val = *v
	case *int32:
		if v == nil {
			return slog.Attr{}
		}

		val = int64(*v)
	case int32:
		val = int64(v)
	default:
		rv, ok := derefNamed(value)
		if !ok {
			return slog.Attr{}
		}

		val = rv.Int()
	}

	return slog.Attr{Key: key, Value: slog.Int64Value(val)}
}

// derefNamed handles named types accepted by the helpers' ~T constraints
// (e.g. type Name string, type Ptr *int), which the built-in type switch
// cases do not match. It dereferences pointers and reports ok=false for a
// nil pointer.
func derefNamed(value any) (rv reflect.Value, ok bool) {
	rv = reflect.ValueOf(value)
	if rv.Kind() != reflect.Pointer {
		return rv, true
	}

	if rv.IsNil() {
		return reflect.Value{}, false
	}

	return rv.Elem(), true
}

// ModuleKey is the attribute key used for identifying the subsystem or module.
const ModuleKey = "subsystem"

// Module creates a slog.Attr that identifies the subsystem generating the log.
//
// Example:
//
//	logger.Info("starting", slogx.Module("http-server"))
func Module(module string) slog.Attr {
	return slog.Attr{
		Key:   ModuleKey,
		Value: slog.StringValue(module),
	}
}

// Modules creates subsystem attributes for variadic logging methods.
func Modules(names ...string) []any {
	out := make([]any, len(names))
	for i, name := range names {
		out[i] = Module(name)
	}

	return out
}

// LevelOff disables logging, including records above the standard error level.
const LevelOff = slog.Level(1<<31 - 1)
