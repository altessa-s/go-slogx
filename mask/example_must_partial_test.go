// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask_test

import (
	"fmt"

	"github.com/altessa-s/go-slogx/mask"
)

// maskAccount is fixed in code, so it is built once with MustPartial.
var maskAccount = mask.MustPartial(0, 4, '*')

func ExampleMustPartial() {
	fmt.Println(maskAccount("4000123412341234")) // Output: ************1234
}
