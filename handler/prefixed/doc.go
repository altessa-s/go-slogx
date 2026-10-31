// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package prefixed collects string attributes under one key and emits a single
// formatted prefix attribute. setup inserts it for text and console sinks.
//
// # Usage
//
//	handler, err := prefixed.New(next)
package prefixed
