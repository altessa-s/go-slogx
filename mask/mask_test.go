// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask_test

import (
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/mask"
)

func TestMasksUnicodeAndValidation(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "a", "abc", "password", "секретный", "😀😁😂🤣😃"} {
		f, err := mask.Partial(1, 1, '●')
		require.NoError(t, err)
		result := f(s)
		require.True(t, utf8.ValidString(result))
		require.Equal(t, utf8.RuneCountInString(s), utf8.RuneCountInString(result))
		if len([]rune(s)) > 2 {
			require.Equal(t, []rune(s)[0], []rune(result)[0])
		}
		for _, m := range []mask.Func{mask.Full(), mask.Fixed("x"), mask.Email(), mask.Phone(), mask.CreditCard(), mask.URL()} {
			require.True(t, utf8.ValidString(m(s)))
		}
	}
	for _, args := range [][2]int{{-1, 0}, {0, -1}, {math.MaxInt, math.MaxInt}} {
		f, err := mask.Partial(args[0], args[1], '*')
		if args[0] < 0 || args[1] < 0 {
			require.ErrorIs(t, err, mask.ErrParams)
		} else {
			require.Equal(t, "***", f("abc"))
		}
	}
	_, err := mask.Partial(0, 0, -1)
	require.ErrorIs(t, err, mask.ErrParams)
	require.Equal(t, "u**r@example.test", mask.Email()("user@example.test"))
	re, err := mask.Regexp("secret", mask.Full())
	require.NoError(t, err)
	require.Equal(t, "********", re("secret"))
	require.Equal(t, "clean", re("clean"))
	_, err = mask.Regexp("[", mask.Full())
	require.ErrorIs(t, err, mask.ErrParams)
	_, err = mask.Regexp("x", nil)
	require.ErrorIs(t, err, mask.ErrParams)
	for _, r := range []mask.Rule{{}, {Field: "x", Pattern: "x"}, {Pattern: "["}} {
		require.ErrorIs(t, r.Validate(), mask.ErrInvalidRule)
	}
	require.NoError(t, (mask.Rule{Field: "x"}).Validate())
	require.NoError(t, (mask.Rule{Pattern: "x"}).Validate())
}
func TestURLNeverLeaksCredentials(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"https://user:canary@host", "https://user:canary@host/a/file?q=canary&z=canary#canary", "mailto:canary", "https://host/?bad=%xx", "://canary", "relative-canary"} {
		result := mask.URL()(input)
		require.NotContains(t, result, "canary")
		require.NotContains(t, result, "user")
	}
	require.Equal(t, "https://host/", mask.URL()("https://host/"))
}
func TestParamsAndCatalog(t *testing.T) {
	t.Parallel()
	for _, v := range []any{1, int64(1), float64(1), "1", json.Number("1")} {
		n, err := (mask.Params{"n": v}).Int("n", 0)
		require.NoError(t, err)
		require.Equal(t, 1, n)
	}
	for _, v := range []any{true, 1.5, math.Inf(1), math.NaN(), "bad", json.Number("1.5"), math.Exp2(63)} {
		_, err := (mask.Params{"n": v}).Int("n", 0)
		require.ErrorIs(t, err, mask.ErrParams)
	}
	n, err := (mask.Params{}).Int("missing", 7)
	require.NoError(t, err)
	require.Equal(t, 7, n)
	for _, v := range []any{true, "true"} {
		b, err := (mask.Params{"b": v}).Bool("b", false)
		require.NoError(t, err)
		require.True(t, b)
	}
	_, err = (mask.Params{"b": 1}).Bool("b", false)
	require.ErrorIs(t, err, mask.ErrParams)
	_, err = (mask.Params{"b": "invalid"}).Bool("b", false)
	require.ErrorIs(t, err, mask.ErrParams)
	b, err := (mask.Params{}).Bool("b", true)
	require.NoError(t, err)
	require.True(t, b)
	s, err := (mask.Params{}).String("s", "default")
	require.NoError(t, err)
	require.Equal(t, "default", s)
	_, err = (mask.Params{"s": 1}).String("s", "")
	require.ErrorIs(t, err, mask.ErrParams)
	masks := map[string]mask.Func{"custom": mask.Fixed("safe")}
	catalog := mustNew(t, mask.WithMasks(masks))
	masks["custom"] = mask.Fixed("changed")
	for name := range catalog.Names() {
		f, err := catalog.Resolve(name, nil)
		if name == "fixed" || name == "pattern" {
			require.ErrorIs(t, err, mask.ErrParams)
		} else {
			require.NoError(t, err)
			require.NotNil(t, f)
		}
	}
	f, err := catalog.Resolve("custom", nil)
	require.NoError(t, err)
	require.Equal(t, "safe", f("input"))
	_, err = catalog.Resolve("absent", nil)
	require.ErrorIs(t, err, mask.ErrUnknownMask)
	f, err = catalog.Resolve("partial", mask.Params{"showFirst": float64(1), "showLast": "1"})
	require.NoError(t, err)
	require.Equal(t, "a**d", f("abcd"))
	for _, p := range []mask.Params{{"showFirst": true}, {"maskChar": ""}, {"maskChar": "ab"}, {"showLast": -1}, {"maskChar": 1}} {
		_, err = catalog.Resolve("partial", p)
		require.ErrorIs(t, err, mask.ErrParams)
	}
	_, err = catalog.Resolve("fixed", mask.Params{"value": 1})
	require.ErrorIs(t, err, mask.ErrParams)
	empty := mustNew(t, mask.WithBuiltins(false))
	require.Empty(t, slices.Collect(empty.Names()))
	custom := mustNew(t, mask.WithFactories(map[string]mask.Factory{"empty": func(mask.Params) (mask.Func, error) { return nil, nil }, "fail": func(mask.Params) (mask.Func, error) { return nil, errors.New("failure") }}))
	for _, name := range []string{"empty", "fail"} {
		_, err = custom.Resolve(name, nil)
		require.ErrorIs(t, err, mask.ErrParams)
	}
}
func FuzzPartial(f *testing.F) {
	f.Add("секрет", 1, 1)
	f.Fuzz(func(t *testing.T, s string, first, last int) {
		m, err := mask.Partial(first, last, '*')
		if first < 0 || last < 0 {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		out := m(s)
		require.True(t, utf8.ValidString(out))
		require.Equal(t, utf8.RuneCountInString(s), utf8.RuneCountInString(out))
	})
}
func FuzzURL(f *testing.F) {
	f.Add("https://user:canary@host/path?q=canary")
	f.Fuzz(func(t *testing.T, s string) {
		out := mask.URL()(s)
		if out == "" || out == "********" {
			return
		}
		u, err := url.Parse(out)
		require.NoError(t, err)
		require.Nil(t, u.User)
		require.Empty(t, u.Fragment)
		for _, values := range u.Query() {
			for _, value := range values {
				require.Equal(t, "********", value)
			}
		}
	})
}
func FuzzParams(f *testing.F) {
	f.Add("123")
	f.Fuzz(func(t *testing.T, s string) {
		n, err := (mask.Params{"n": s}).Int("n", 0)
		if err == nil {
			again, err := (mask.Params{"n": n}).Int("n", 0)
			require.NoError(t, err)
			require.Equal(t, n, again)
		}
	})
}

func TestPatternFactoryAndMaskBoundaries(t *testing.T) {
	t.Parallel()
	catalog := mustNew(t)
	for _, p := range []mask.Params{{}, {"pattern": 1}, {"pattern": "["}, {"pattern": "x", "replacement": 1}, {"pattern": "x", "replacementType": 1}, {"pattern": "x", "replacementValue": 1}, {"pattern": "x", "replacementType": "missing"}} {
		_, err := catalog.Resolve("pattern", p)
		require.ErrorIs(t, err, mask.ErrParams)
	}
	for _, p := range []mask.Params{{"pattern": "x", "replacement": "safe"}, {"pattern": "x"}, {"pattern": "x", "replacementType": "full"}} {
		f, err := catalog.Resolve("pattern", p)
		require.NoError(t, err)
		require.NotEqual(t, "x", f("x"))
		require.Equal(t, "clean", f("clean"))
	}
	f, err := catalog.Resolve("fixed", mask.Params{"value": "safe"})
	require.NoError(t, err)
	require.Equal(t, "safe", f("input"))
	require.Equal(t, "***@host", mask.Email()("ab@host"))
	require.Equal(t, "a*****j@host", mask.Email()("abcdefghij@host"))
	require.Equal(t, "+155******67", mask.Phone()("+15551234567"))
	require.Equal(t, "********", mask.Phone()("123"))
	require.Equal(t, "4111**********11", mask.CreditCard()("4111111111111111"))
}
