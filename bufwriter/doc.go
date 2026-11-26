// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package bufwriter is an asynchronous io.Writer. Callers format a record
// first and hand this writer finished bytes. A full queue drops the newest
// write. Priority writes can wait for capacity within a context budget. Close
// waits for delivery within its context and reports delivery statistics.
//
// # Usage
//
//	writer, err := bufwriter.New(output)
package bufwriter
