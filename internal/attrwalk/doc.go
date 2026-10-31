// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package attrwalk reflects through maps, slices, structs, and LogValuer
// values so a mask can be applied inside them. Descent stops at a fixed
// depth and replaces the remainder.
//
// # Usage
//
//	walker := attrwalk.Walker{Match: resolveMask}
package attrwalk
