// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package attrwalk masks the encoding/json representation of arbitrary values.
// Depth and breadth limits replace uninspectable data with fixed placeholders.
//
// # Usage
//
//	walker := attrwalk.Walker{Match: resolveMask}
package attrwalk
