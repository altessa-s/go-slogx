// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup

//go:generate optgen generate --keep-empty-strings --no-default-modifiers
//go:generate sh ../docs/generated-header.sh options_gen.go
import (
	"io"
	"log/slog"

	"github.com/altessa-s/go-atlas-core/runtime/appinfo"

	"github.com/altessa-s/go-slogx/console"
	"github.com/altessa-s/go-slogx/mask"

	slogx "github.com/altessa-s/go-slogx"
)

// Format selects the built-in sink.
type Format uint8

// Supported output formats.
const (
	Text Format = iota
	JSON
)

// ColorMode controls terminal color detection.
type ColorMode uint8

// Supported color policies.
const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

type options struct {
	level             slog.Level `optgen:"default=DefaultLevel"`
	format            Format
	sink              func(io.Writer, *slog.HandlerOptions) slog.Handler
	color             ColorMode
	addSource         bool   `optval:"param"`
	timeFormat        string `optgen:"default=DefaultTimeFormat"`
	tags              map[string]string
	appGroup          string `optgen:"default=DefaultAppGroup"`
	appName           string `optgen:"default=defaultAppName()"`
	appVersion        string `optgen:"default=defaultAppVersion()"`
	serviceID         string
	subsystemKey      string `optgen:"default=DefaultSubsystemKey"`
	subsystemLevels   map[string]slog.Level
	maskRules         []mask.Rule
	maskDefaults      bool   `optval:"param"`
	maskString        string `optgen:"default=DefaultMaskString"`
	bufferSize        int
	bufferBypassLevel slog.Level `optgen:"default=DefaultBufferBypassLevel"`
	onError           func(error)
}

// DefaultSubsystemKey identifies subsystem attributes.
const DefaultSubsystemKey = slogx.ModuleKey

// DefaultTimeFormat formats console timestamps.
const DefaultTimeFormat = console.DefaultTimeFormat

func defaultAppName() string    { return appinfo.Name }
func defaultAppVersion() string { return appinfo.Version }

// DefaultLevel is the initial default threshold.
const DefaultLevel = slog.LevelInfo

// DefaultAppGroup groups application metadata.
const DefaultAppGroup = "app"

// DefaultMaskString replaces fields whose rule has no explicit mask.
const DefaultMaskString = "********"

// DefaultBufferBypassLevel flushes error records before logging returns.
const DefaultBufferBypassLevel = slog.LevelError
