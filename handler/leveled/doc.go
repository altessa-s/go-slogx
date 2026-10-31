// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package leveled filters records by a dynamic slog.Leveler and by static
// per-subsystem levels. Enabled re-reads the leveler on every call.
//
// # Usage
//
//	handler, err := leveled.New(next)
package leveled
