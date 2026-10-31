// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package slogx is the domain of the structured-logging library: nil-safe
// attributes, a mutable field bag carried in a context, and the off level.
//
// It imports the standard library only. Configuration, sinks, and process
// hooks live in the consumer and in the setup adapter.
//
// # Usage
//
//	attr := slogx.Module("worker")
package slogx
