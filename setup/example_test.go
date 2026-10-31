// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/altessa-s/go-slogx/setup"
)

func ExampleNew() {
	var output bytes.Buffer
	l, err := setup.New(&output, setup.WithFormat(setup.JSON))
	if err != nil {
		panic(err)
	}
	l.Logger().Info("ready")
	if err := l.Shutdown(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println(output.Len() > 0) // Output: true
}
