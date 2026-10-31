// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package mask holds redaction functions, a rule value, and an immutable
// catalog of named masks. It imports the standard library only.
//
// Message text is outside this package. Callers mask attribute values.
//
// # Usage
//
//	redact, err := mask.Partial(1, 1, '*')
package mask
