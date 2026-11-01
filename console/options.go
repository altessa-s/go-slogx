// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package console

//go:generate optgen generate --keep-empty-strings --no-default-modifiers
import (
	"log/slog"
	"time"
)

// DefaultTimeFormat is used for built-in and attribute timestamps.
const DefaultTimeFormat = time.RFC3339Nano

// Style selects a trusted SGR foreground color.
type Style uint8

// Supported foreground styles.
const (
	Plain  Style = 0
	Red    Style = 31
	Green  Style = 32
	Yellow Style = 33
	Blue   Style = 34
	Cyan   Style = 36
)

// Format selects the record layout used by the console handler.
type Format uint8

// Supported record formats.
const (
	FormatKeyValue Format = iota
	FormatText
)

type options struct {
	level       slog.Leveler `optgen:"default=DefaultLevel"`
	addSource   bool         `optval:"param"`
	color       bool         `optval:"param"`
	format      Format       `optgen:"default=FormatKeyValue" optval:"param"`
	timeFormat  string       `optgen:"default=DefaultTimeFormat"`
	prefixKey   string       `optgen:"default=DefaultPrefixKey"`
	keyStyles   map[string]Style
	replaceAttr func([]string, slog.Attr) slog.Attr
}

// DefaultLevel is the console admission threshold.
const DefaultLevel = slog.LevelInfo

// DefaultPrefixKey identifies the formatted subsystem prefix.
const DefaultPrefixKey = "subsystem"
