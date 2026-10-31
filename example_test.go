// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx_test

import (
	"fmt"

	"github.com/altessa-s/go-slogx"
)

func ExampleFields_Grouped() {
	fields := slogx.Fields{{Key: "request.id", Value: "r-1"}}
	fmt.Println(fields.Grouped()[0]) // Output: request=[id=r-1]
}
