// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/masking"
	"github.com/altessa-s/go-slogx/mask"
	"github.com/altessa-s/go-slogx/slogxtest"
)

type logValue struct {
	calls *int
	value slog.Value
}

func (v logValue) LogValue() slog.Value { *v.calls++; return v.value }

type payload struct {
	Password string `json:"password"`
	Public   string
	Hidden   string `json:"-"`
}
type cyclic struct {
	Password string
	Next     *cyclic
}

func TestNestedMaskingAndDepth(t *testing.T) {
	t.Parallel()

	values := []any{
		payload{Password: "canary", Public: "visible"}, &payload{Password: "canary"},
		map[string]any{"password": "canary", "public": "visible"}, []payload{{Password: "canary"}},
		[1]payload{{Password: "canary"}},
	}
	c := &cyclic{Password: "canary"}
	c.Next = c
	values = append(values, c)

	var pointer any

	pointer = &pointer

	values = append(values, pointer)
	for i, value := range values {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			t.Parallel()

			r := slogxtest.NewRecorder()
			h, err := masking.New(r, masking.WithDefaults(true))
			require.NoError(t, err)
			slog.New(h).Info("record", slog.Any("payload", value))

			rows, err := r.Records()
			require.NoError(t, err)
			require.NotContains(t, fmt.Sprint(rows), "canary")
		})
	}
}
func TestRulesCapturedGroupsAndLogValuer(t *testing.T) {
	t.Parallel()

	r := slogxtest.NewRecorder()
	calls := 0
	h, err := masking.New(r, masking.WithNested(true), masking.WithRules(mask.Rule{Field: "g.password"}, mask.Rule{Pattern: "token"}), masking.WithDefaultMask(mask.Fixed("safe")))
	require.NoError(t, err)
	require.Same(t, h, h.WithGroup(""))
	log := slog.New(h).WithGroup("g").With(slog.Any("password", logValue{&calls, slog.StringValue("canary")}))
	log.Info("unmasked message", slog.String("token", "canary"), slog.Int("public", 3))
	require.Equal(t, 1, calls)

	rows, err := r.Records()
	require.NoError(t, err)

	group := rows[0]["g"].(map[string]any)
	require.Equal(t, "safe", group["password"])
	require.Equal(t, "safe", group["token"])
	require.Equal(t, float64(3), group["public"])
	r.Reset()
	h, err = masking.New(r, masking.WithCaseSensitive(true), masking.WithNested(false), masking.WithRules(mask.Rule{Field: "Password", Mask: mask.Fixed("yes")}))
	require.NoError(t, err)
	slog.New(h).Info("case", slog.String("password", "visible"), slog.String("Password", "canary"))

	rows, err = r.Records()
	require.NoError(t, err)
	require.Equal(t, "visible", rows[0]["password"])
	require.Equal(t, "yes", rows[0]["Password"])

	for _, rule := range []mask.Rule{{}, {Pattern: "["}, {Field: "x", Pattern: "x"}} {
		_, err = masking.New(r, masking.WithRules(rule))
		require.ErrorIs(t, err, mask.ErrInvalidRule)
	}

	_, err = masking.New(nil)
	require.Error(t, err)
}
func TestAtomicValuesAndCachePressure(t *testing.T) {
	t.Parallel()

	r := slogxtest.NewRecorder()
	h, err := masking.New(r, masking.WithDefaults(true))
	require.NoError(t, err)

	log := slog.New(h)
	for _, v := range []any{nil, (*payload)(nil), time.Unix(0, 0), time.Second, []byte("safe"), map[string]any{}, []string{}, struct{}{}, 7, true} {
		log.Info("atomic", slog.Any("value", v))
	}

	for i := range 4200 {
		log.Info("cache", slog.String(fmt.Sprintf("dynamic-%d", i), "visible"))
	}

	log.Info("long", slog.String(strings.Repeat("x", 300), "visible"))

	rows, err := r.Records()
	require.NoError(t, err)
	require.Len(t, rows, 4211)
}
