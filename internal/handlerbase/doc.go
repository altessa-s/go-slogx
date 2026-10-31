// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package handlerbase stores the next slog.Handler and the open group list
// for middleware. It performs no policy of its own.
//
// # Usage
//
//	base := handlerbase.NewBase(next)
package handlerbase
