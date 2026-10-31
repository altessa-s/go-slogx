// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package console is a text slog.Handler that writes its own SGR sequences.
// Color is a per-handler bool. The package sets no process-wide color flag.
//
// # Usage
//
//	handler, err := console.New(output)
package console
