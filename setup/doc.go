// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package setup composes one slog.Logger from options and an injected writer.
// It reads no configuration file and registers no process shutdown hook.
//
// # Usage
//
//	logger, err := setup.New(output, setup.WithFormat(setup.JSON))
package setup
