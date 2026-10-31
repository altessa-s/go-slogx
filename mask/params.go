// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask

import (
	"encoding/json"
	"math"
	"strconv"
)

// Params contains consumer-decoded factory parameters.
type Params map[string]any

// Int accepts integers, integral JSON numbers, and numeric strings.
func (p Params) Int(key string, fallback int) (int, error) {
	v, ok := p[key]
	if !ok {
		return fallback, nil
	}
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		if int64(int(n)) == n {
			return int(n), nil
		}
	case float64:
		if !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n && n >= -math.Exp2(strconv.IntSize-1) && n < math.Exp2(strconv.IntSize-1) {
			return int(n), nil
		}
	case string:
		if x, err := strconv.Atoi(n); err == nil {
			return x, nil
		}
	case json.Number:
		if x, err := strconv.Atoi(string(n)); err == nil {
			return x, nil
		}
	}
	return 0, ErrParams
}

// String returns a string parameter or fallback when absent.
func (p Params) String(key, fallback string) (string, error) {
	v, ok := p[key]
	if !ok {
		return fallback, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", ErrParams
	}
	return s, nil
}

// Bool accepts booleans and strings recognized by strconv.ParseBool.
func (p Params) Bool(key string, fallback bool) (bool, error) {
	v, ok := p[key]
	if !ok {
		return fallback, nil
	}
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		if x, err := strconv.ParseBool(b); err == nil {
			return x, nil
		}
	}
	return false, ErrParams
}
