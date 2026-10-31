// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package bufwriter is an asynchronous io.Writer. Callers format a record
// first and hand this writer finished bytes. A full queue writes synchronously.
// Close drains and can be called more than once.
//
// # Usage
//
//	writer, err := bufwriter.New(output)
package bufwriter
