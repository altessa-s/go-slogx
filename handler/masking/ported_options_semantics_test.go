// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/masking"
)

// logJSON logs a single record with attrs through a masking handler built
// from opts and returns the decoded JSON record.
func logJSON(t *testing.T, opts []masking.Option, attrs ...any) map[string]any {
	t.Helper()

	var buf bytes.Buffer

	logger := slog.New(portedHandler(t, slog.NewJSONHandler(&buf, nil), opts...))
	logger.Info("test", attrs...)

	var record map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &record))

	return record
}

// TestDefaultMask_AppliesToNilMaskRules pins the semantics of WithDefaultMask:
// it is the mask for rules registered without a MaskFunc. Before the fix the
// option was stored but never read, and a nil-mask rule leaked its value.
func TestDefaultMask_AppliesToNilMaskRules(t *testing.T) {
	t.Parallel()

	fixed := fixedMask("[X]")

	tests := []struct {
		name string
		opts []masking.Option
		key  string
		want string
	}{
		{
			name: "field with nil mask uses WithDefaultMask",
			opts: []masking.Option{masking.WithDefaultMask(fixed), fieldOption("secret", nil)},
			key:  "secret",
			want: "[X]",
		},
		{
			name: "pattern with nil mask uses WithDefaultMask",
			opts: []masking.Option{masking.WithDefaultMask(fixed), patternOption(`(?i)^ssn$`, nil)},
			key:  "SSN",
			want: "[X]",
		},
		{
			name: "nil mask without WithDefaultMask falls back to FullMask",
			opts: []masking.Option{fieldOption("secret", nil)},
			key:  "secret",
			want: fullMask()("value-123"),
		},
		{
			name: "WithDefaults keeps an earlier WithDefaultMask",
			opts: []masking.Option{masking.WithDefaultMask(fixed), masking.WithDefaults(true), fieldOption("pin", nil)},
			key:  "pin",
			want: "[X]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			record := logJSON(t, tc.opts, tc.key, "value-123")
			require.Equal(t, tc.want, record[tc.key])
		})
	}
}

// TestDefaultMask_DoesNotMaskUnmatchedFields guards the other half of the
// contract: the default mask is not a catch-all for fields no rule names.
func TestDefaultMask_DoesNotMaskUnmatchedFields(t *testing.T) {
	t.Parallel()

	record := logJSON(t,
		[]masking.Option{masking.WithDefaultMask(fixedMask("[X]")), fieldOption("secret", nil)},
		"user", "alice",
	)
	require.Equal(t, "alice", record["user"])
}

// TestWithDefaults_StaysCaseInsensitive is the regression for WithDefaults
// silently switching exact field matching to case-sensitive: mixed-case keys
// that only the exact-name defaults cover (no default pattern matches them)
// leaked verbatim.
func TestWithDefaults_StaysCaseInsensitive(t *testing.T) {
	t.Parallel()

	const (
		bearer = "Bearer abcdefghijklmnop"
		email  = "someone@example.com"
	)

	record := logJSON(t, []masking.Option{masking.WithDefaults(true)},
		"Authorization", bearer,
		"EMAIL", email,
	)
	require.NotEqual(t, bearer, record["Authorization"])
	require.NotEqual(t, email, record["EMAIL"])
}

// TestWithDefaults_CredentialsRevealNothing is the regression for the default
// credential masks revealing most of a short secret: SmartMask kept the first
// and last two characters and the length, so a 6-character password showed 4
// of its characters and a bearer token showed its scheme and tail. Credential
// keys, the credential patterns and nil-mask rules must all mask to a
// fixed-length placeholder that carries nothing of the value.
func TestWithDefaults_CredentialsRevealNothing(t *testing.T) {
	t.Parallel()

	full := fullMask()
	tests := []struct {
		name  string
		opts  []masking.Option
		key   string
		value string
	}{
		{name: "short password", key: "password", value: "hunt3r"},
		{name: "long password", key: "password", value: "correct-horse-battery-staple"},
		{name: "token", key: "token", value: "tok_0123456789"},
		{name: "secret", key: "secret", value: "s3cr3t"},
		{name: "api key", key: "api_key", value: "AKIA1234567890"},
		{name: "bearer authorization", key: "authorization", value: "Bearer abcdefghijklmnop"},
		{name: "password pattern", key: "db_password", value: "hunt3r"},
		{name: "token pattern", key: "refreshToken", value: "rt-987654"},
		{name: "secret pattern", key: "client_secret", value: "cs-1234"},
		{name: "key pattern", key: "signing_key", value: "k-123456"},
		{
			name:  "nil-mask rule",
			opts:  []masking.Option{masking.WithDefaults(true), fieldOption("pin", nil)},
			key:   "pin",
			value: "4321",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := tc.opts
			if opts == nil {
				opts = []masking.Option{masking.WithDefaults(true)}
			}

			record := logJSON(t, opts, tc.key, tc.value)
			require.Equal(t, full(tc.value), record[tc.key])
		})
	}
}

// TestWithDefaults_PatternBoundaries pins which keys the default credential
// patterns cover: password, secret and token anywhere in the key in any
// case, and keys ending in "_key". Keys that only resemble them stay clear.
func TestWithDefaults_PatternBoundaries(t *testing.T) {
	t.Parallel()

	const value = "value-123"

	masked := fullMask()(value)
	tests := []struct {
		key  string
		want string
	}{
		{key: "OldPassword", want: masked},
		{key: "x_SECRET_y", want: masked},
		{key: "accessTokenTTL", want: masked},
		{key: "Signing_Key", want: masked},
		{key: "monkey", want: value},
		{key: "key_id", want: value},
		{key: "tok", want: value},
		{key: "request_id", want: value},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()

			record := logJSON(t, []masking.Option{masking.WithDefaults(true)}, tc.key, value)
			require.Equal(t, tc.want, record[tc.key])
		})
	}
}

// TestWithDefaults_CaseSensitiveStillOptIn checks that an explicit
// WithCaseSensitive is honored together with WithDefaults, in either order.
func TestWithDefaults_CaseSensitiveStillOptIn(t *testing.T) {
	t.Parallel()

	orders := map[string][]masking.Option{
		"defaults first":       {masking.WithDefaults(true), masking.WithCaseSensitive(true)},
		"case-sensitive first": {masking.WithCaseSensitive(true), masking.WithDefaults(true)},
	}

	for name, opts := range orders {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			record := logJSON(t, opts, "EMAIL", "someone@example.com", "email", "someone@example.com")
			require.Equal(t, "someone@example.com", record["EMAIL"], "case-sensitive matching must not fold EMAIL")
			require.NotEqual(t, "someone@example.com", record["email"])
		})
	}
}

// TestCaseFoldCollision_LastRegistrationWins pins deterministic precedence
// for keys that differ only by case. Folding used to range over the fields
// map, so an explicit "Email" rule could randomly lose to the default
// "email" mask. Repetition defeats a lucky map iteration order.
func TestCaseFoldCollision_LastRegistrationWins(t *testing.T) {
	t.Parallel()

	const value = "someone@example.com"

	fixed := fixedMask("[X]")

	for range 50 {
		record := logJSON(t,
			[]masking.Option{masking.WithDefaults(true), fieldOption("Email", fixed)},
			"email", value, "Email", value,
		)
		require.Equal(t, "[X]", record["email"])
		require.Equal(t, "[X]", record["Email"])

		record = logJSON(t,
			[]masking.Option{fieldOption("Email", fixed), masking.WithDefaults(true)},
			"Email", value,
		)
		require.Equal(t, "[X]", record["Email"], "explicit rules override the defaults policy")
	}
}

// TestNew_ReportsInvalidPatterns is the regression for invalid WithPattern
// rules being dropped without trace: New reports each one, wrapping
// ErrInvalidPattern, and returns no handler.
func TestNew_ReportsInvalidPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "does not compile", pattern: "(unclosed"},
		{name: "empty", pattern: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, err := masking.New(slog.DiscardHandler,
				patternOption(`(?i)token`, fullMask()),
				patternOption(tc.pattern, fullMask()),
			)
			require.ErrorIs(t, err, invalidRule)
			require.Nil(t, h)
		})
	}
}

// TestNew_ReportsEveryInvalidPattern checks that all invalid patterns are
// reported, not just the first.
func TestNew_ReportsEveryInvalidPattern(t *testing.T) {
	t.Parallel()

	_, err := masking.New(slog.DiscardHandler,
		patternOption("(a", fullMask()),
		patternOption("[b", fullMask()),
	)
	require.ErrorIs(t, err, invalidRule)
	require.ErrorContains(t, err, `"(a"`)
	require.ErrorContains(t, err, `"[b"`)
}

// TestNew_ValidOptionsMasks is the positive control: valid options build a
// handler that masks like NewHandler.
func TestNew_ValidOptionsMasks(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	h, err := masking.New(slog.NewJSONHandler(&buf, nil),
		patternOption(`(?i)token`, fixedMask("[T]")),
	)
	require.NoError(t, err)
	slog.New(h).Info("x", "AccessToken", "abc")

	var record map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	require.Equal(t, "[T]", record["AccessToken"])
}
