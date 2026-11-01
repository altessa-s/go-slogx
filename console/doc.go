// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package console is a text slog.Handler with quoted key/value and compact
// human-readable formats. It writes its own SGR sequences; color is a
// per-handler option and the package sets no process-wide color flag.
//
// # Usage
//
//	handler, err := console.New(output)
package console
