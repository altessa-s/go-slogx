// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask_test

import (
	"fmt"

	"github.com/altessa-s/go-slogx/mask"
)

func ExamplePartial() {
	f, err := mask.Partial(1, 1, '*')
	if err != nil {
		panic(err)
	}

	fmt.Println(f("секрет")) // Output: с****т
}
