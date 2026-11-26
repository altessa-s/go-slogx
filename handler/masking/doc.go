// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package masking applies mask rules to attribute values before they reach the
// next handler. Rules match dotted paths and leaf keys. WithMessageMask opts
// in to redacting the record message. New validates all rules.
//
// # Usage
//
//	handler, err := masking.New(next, masking.WithDefaults(true))
package masking
