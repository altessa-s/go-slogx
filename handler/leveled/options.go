// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package leveled

//go:generate optgen generate --keep-empty-strings --no-default-modifiers
import (
	"log/slog"

	"github.com/altessa-s/go-slogx"
)

// DefaultLevel is the default record threshold.
const DefaultLevel = slog.LevelInfo

// DefaultSubsystemKey selects the subsystem field.
const DefaultSubsystemKey = slogx.ModuleKey

type options struct {
	defaultLevel    slog.Leveler `optgen:"default=DefaultLevel"`
	subsystemKey    string       `optgen:"default=DefaultSubsystemKey"`
	subsystemLevels map[string]slog.Level
}
