// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package attrwalk_test

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/internal/attrwalk"
	"github.com/altessa-s/go-slogx/mask"
)

type payload struct {
	Password string
	Public   string
	Ignore   string `json:"-"`
	Tagged   string `json:"password,omitempty"`
	Default  string `json:",omitempty"`
	private  string
}
type cycle struct {
	Password string
	Next     *cycle
}
type value struct{ result slog.Value }

func (v value) LogValue() slog.Value { return v.result }

type text string

func (t text) MarshalText() ([]byte, error) { return []byte(t), nil }

type stringer struct{ Password string }

func (s stringer) String() string { return s.Password }
func matcher(name, path string) mask.Func {
	if strings.EqualFold(name, "password") || strings.HasSuffix(strings.ToLower(path), ".password") {
		return mask.Full()
	}
	return nil
}
func TestWalkerKindsAndFailClosed(t *testing.T) {
	t.Parallel()
	w := &attrwalk.Walker{Match: matcher}
	p := payload{Password: "canary", Public: "visible", Tagged: "canary", Default: "kept", Ignore: "canary", private: "hidden"}
	for _, input := range []any{p, &p, map[string]any{"Password": "canary", "Public": "visible"}, []payload{p}, [1]payload{p}, stringer{"canary"}, value{slog.GroupValue(slog.String("Password", "canary"))}, value{slog.AnyValue(map[string]any{"Password": "canary"})}, []any{value{slog.GroupValue(slog.String("Password", "canary"))}}, map[string]any{"child": value{slog.StringValue("visible")}}} {
		a := w.Attr(slog.Any("payload", input), []string{"root"})
		require.NotContains(t, fmt.Sprint(a), "canary")
	}
	c := &cycle{Password: "canary"}
	c.Next = c
	out := w.Attr(slog.Any("cycle", c), nil)
	require.Contains(t, fmt.Sprint(out), "max depth exceeded")
	require.NotContains(t, fmt.Sprint(out), "canary")
	var pointer any
	pointer = &pointer
	require.Contains(t, fmt.Sprint(w.Attr(slog.Any("cycle", pointer), nil)), "max depth exceeded")
	for _, input := range []any{nil, (*payload)(nil), time.Unix(0, 0), time.Second, []byte("safe"), text("safe"), map[string]any{}, []string{}, [0]string{}, struct{}{}, 7, true, make(chan int), func() {}} {
		a := slog.Any("value", input)
		out := w.Attr(a, nil)
		require.Equal(t, a.Value.Kind(), out.Value.Kind())
	}
	require.Equal(t, "<masking: unsupported map keys>", w.Attr(slog.Any("value", map[int]string{1: "safe"}), nil).Value.String())
	require.Equal(t, slog.Attr{}, w.Attr(slog.Attr{}, nil))
	for _, a := range []slog.Attr{slog.String("password", "canary"), slog.Int("password", 1), slog.Uint64("password", 1), slog.Float64("password", 1), slog.Bool("password", true), slog.Duration("password", time.Second), slog.Time("password", time.Now()), slog.Any("password", p), slog.Any("password", value{slog.StringValue("canary")})} {
		require.Equal(t, "********", w.Attr(a, nil).Value.String())
	}
	group := slog.Group("", slog.String("Password", "canary"))
	require.NotContains(t, fmt.Sprint(w.Attr(group, nil)), "canary")
	deep := slog.String("Password", "canary")
	for range 15 {
		deep = slog.GroupAttrs("g", deep)
	}
	require.Contains(t, fmt.Sprint(w.Attr(deep, nil)), "max depth exceeded")
	array := make([]any, 70)
	for i := range array {
		array[i] = i
	}
	require.Equal(t, array, w.Attr(slog.Any("array", array), nil).Value.Any())
	var nested any = p
	for range 15 {
		nested = []any{nested}
	}
	require.NotContains(t, fmt.Sprint(w.Attr(slog.Any("deep", nested), nil)), "canary")
}
func TestTypeVerdictsAndResolvedPointers(t *testing.T) {
	t.Parallel()
	w := &attrwalk.Walker{Match: matcher, Candidate: func(name string) bool { return strings.EqualFold(name, "password") }}
	for range 2 {
		for _, v := range []any{struct{ Clean string }{"safe"}, struct{ Public map[string]string }{map[string]string{"Password": "canary"}}, []payload{{Password: "canary"}}, [1]payload{{Password: "canary"}}, struct{ Value any }{payload{Password: "canary"}}, map[int]string{1: "safe"}, time.Now(), value{slog.StringValue("safe")}} {
			out := w.Attr(slog.Any("v", v), nil)
			require.NotContains(t, fmt.Sprint(out), "canary")
		}
	}
	type recursive struct{ Next *recursive }
	r := &recursive{}
	r.Next = r
	// An unmaskable recursive type can be passed through; do not format it here.
	require.Equal(t, slog.KindAny, w.Attr(slog.Any("v", r), nil).Value.Kind())
	deep := any(payload{Password: "canary"})
	for range 15 {
		copy := deep
		deep = &copy
	}
	require.NotContains(t, fmt.Sprint(w.Attr(slog.Any("deep", deep), nil)), "canary")
	v := value{slog.StringValue("safe")}
	ptr := &v
	require.Equal(t, "safe", w.Attr(slog.Any("pointer", &ptr), nil).Value.String())
}

func TestMatchedCompoundConversionBudget(t *testing.T) {
	t.Parallel()
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	wide := make([]string, 300)
	for _, input := range []any{cyclic, wide} {
		for _, nested := range []bool{false, true} {
			calls := 0
			w := &attrwalk.Walker{Match: func(name, _ string) mask.Func {
				if strings.EqualFold(name, "password") {
					return func(s string) string { calls++; return s }
				}
				return nil
			}}
			attr := slog.Any("password", input)
			if nested {
				attr = slog.Any("payload", map[string]any{"password": input})
			}
			out := w.Attr(attr, nil)
			require.Contains(t, fmt.Sprint(out), "exceeded")
			require.Zero(t, calls)
		}
	}
	w := &attrwalk.Walker{Match: func(_, _ string) mask.Func {
		return func(s string) string { return "masked:" + s }
	}}
	for _, input := range []any{nil, (*string)(nil), []string{"safe"}, map[string]int{"safe": 1}, struct{ Public string }{"safe"}} {
		require.Equal(t, "masked:"+fmt.Sprint(input), w.Attr(slog.Any("password", input), nil).Value.String())
	}
}

func TestUnexportedEmbeddingTypeCacheFailsClosed(t *testing.T) {
	t.Parallel()
	type credentials struct{ Password string }
	type valueEmbedding struct{ credentials }
	type pointerEmbedding struct{ *credentials }
	w := &attrwalk.Walker{
		Match:     matcher,
		Candidate: func(name string) bool { return strings.EqualFold(name, "password") },
	}
	for range 2 {
		for _, input := range []any{valueEmbedding{credentials{"canary"}}, pointerEmbedding{&credentials{"canary"}}} {
			out := w.Attr(slog.Any("payload", input), nil)
			require.Equal(t, "<masking: unexported embedding>", out.Value.String())
		}
	}
}
