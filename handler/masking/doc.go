// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package masking applies mask rules to attribute values before they reach the
// next handler. New rejects a rule that does not validate.
//
// # Usage
//
//	handler, err := masking.New(next, masking.WithDefaults(true))
package masking
