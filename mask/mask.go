// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mask

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

const panicPlaceholder = "********"

// Func redacts a string value.
type Func func(string) string

// ErrParams reports invalid mask parameters.
var ErrParams = errors.New("invalid mask parameters")

// ErrInvalidRule reports an invalid field selector.
var ErrInvalidRule = errors.New("invalid mask rule")

// ErrUnknownMask reports a catalog lookup miss.
var ErrUnknownMask = errors.New("unknown mask")

// DefaultCredentialPattern selects common credential fields.
const DefaultCredentialPattern = `(?i)password|secret|token|_key$`

// Full replaces nonempty values with eight asterisks.
func Full() Func { return Fixed(panicPlaceholder) }

// Fixed replaces nonempty values with replacement.
func Fixed(replacement string) Func {
	return func(s string) string {
		if s == "" {
			return ""
		}
		return replacement
	}
}

// Partial preserves at most first and last runes. Short inputs are fully masked.
func Partial(first, last int, r rune) (Func, error) {
	if first < 0 || last < 0 || !utf8.ValidRune(r) {
		return nil, ErrParams
	}
	return func(s string) string {
		chars := []rune(s)
		start, end := first, len(chars)-last
		if first >= len(chars) || last >= len(chars)-first {
			start, end = 0, len(chars)
		}
		for i := start; i < end; i++ {
			chars[i] = r
		}
		return string(chars)
	}, nil
}

// Email masks the local part and preserves the domain.
func Email() Func {
	return func(s string) string {
		local, domain, ok := strings.Cut(s, "@")
		if !ok {
			return Full()(s)
		}
		runes := []rune(local)
		if len(runes) <= visibleEmailEdges {
			return "***@" + domain
		}
		hidden := min(len(runes)-visibleEmailEdges, maxEmailMaskRunes)
		return string(runes[0]) + strings.Repeat("*", hidden) + string(runes[len(runes)-1]) + "@" + domain
	}
}

// Phone preserves four leading and two trailing runes of a phone number.
// Short nonempty inputs use the fixed full mask.
func Phone() Func {
	partial, _ := Partial(visiblePrefix, visibleSuffix, '*')
	return func(s string) string {
		if utf8.RuneCountInString(s) < minimumPhoneRunes {
			return Full()(s)
		}
		return partial(s)
	}
}

// CreditCard preserves four leading and two trailing runes of a card number.
func CreditCard() Func { partial, _ := Partial(visiblePrefix, visibleSuffix, '*'); return partial }

// URL removes userinfo, query values and fragments and masks the entire path (including the last segment).
// Malformed, relative and opaque URLs fail closed.
func URL() Func {
	return func(s string) string {
		if s == "" {
			return ""
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || u.Host == "" || u.Opaque != "" {
			return Full()(s)
		}
		u.User = nil
		u.Fragment, u.RawFragment = "", ""
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return Full()(s)
		}
		for key, values := range q {
			for i := range values {
				values[i] = panicPlaceholder
			}
			q[key] = values
		}
		u.RawQuery = q.Encode()
		if u.Path != "" && u.Path != "/" {
			u.Path = "/********"
		}
		u.RawPath = ""
		return u.String()
	}
}

// Regexp masks an entire value when pattern matches it.
func Regexp(pattern string, m Func) (Func, error) {
	if m == nil {
		return nil, ErrParams
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParams, err)
	}
	return func(s string) (result string) {
		defer func() {
			if recover() != nil {
				result = panicPlaceholder
			}
		}()
		if re.MatchString(s) {
			return m(s)
		}
		return s
	}, nil
}

// Rule selects either an exact field path or a regular expression.
// A nil Mask uses the handler's default mask.
type Rule struct {
	Field, Pattern string
	Mask           Func
}

// Validate rejects ambiguous, empty and invalid selectors.
func (r Rule) Validate() error {
	if (r.Field == "") == (r.Pattern == "") {
		return ErrInvalidRule
	}
	if r.Pattern != "" {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidRule, err)
		}
	}
	return nil
}

const (
	visiblePrefix     = 4
	visibleSuffix     = 2
	minimumPhoneRunes = 7
	visibleEmailEdges = 2
	maxEmailMaskRunes = 5
)
