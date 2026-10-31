// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package masking_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/handler/masking"
	"github.com/altessa-s/go-slogx/mask"
)

type nullJSONMarshaler struct{}

type nullableLogValue struct{ value slog.Value }

func (v nullableLogValue) LogValue() slog.Value { return v.value }

func (nullJSONMarshaler) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

func TestJSONEmptyAttributeKeyPreservesNull(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		attr slog.Attr
		omit bool
	}{
		{name: "nil pointer", attr: slog.Any("", (*string)(nil))},
		{name: "nil map", attr: slog.Any("", map[string]string(nil))},
		{name: "nil slice", attr: slog.Any("", []string(nil))},
		{name: "raw null", attr: slog.Any("", json.RawMessage("null"))},
		{name: "custom marshaler", attr: slog.Any("", nullJSONMarshaler{})},
		{name: "zero attribute", attr: slog.Attr{}, omit: true},
		{name: "untyped nil", attr: slog.Any("", nil), omit: true},
		{name: "resolved zero", attr: slog.Any("", nullableLogValue{}), omit: true},
		{name: "resolved typed null", attr: slog.Any("", nullableLogValue{slog.AnyValue((*string)(nil))})},
	}
	for _, tc := range cases {
		for _, group := range []string{"none", "WithGroup", "attribute"} {
			for _, captured := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/group=%s/captured=%t", tc.name, group, captured), func(t *testing.T) {
					t.Parallel()
					var buf bytes.Buffer
					h := portedHandler(t, slog.NewJSONHandler(&buf, nil), masking.WithDefaults(true))
					logger := slog.New(h)
					attr := tc.attr
					switch group {
					case "WithGroup":
						logger = logger.WithGroup("root")
					case "attribute":
						attr = slog.Group("root", attr)
					}
					attrs := []slog.Attr{attr}
					if captured {
						logger = slog.New(logger.Handler().WithAttrs(attrs))
						attrs = nil
					}
					logger.LogAttrs(t.Context(), slog.LevelInfo, "test", attrs...)
					var record map[string]any
					require.NoError(t, json.Unmarshal(buf.Bytes(), &record))
					delete(record, "time")
					delete(record, "level")
					delete(record, "msg")
					want := map[string]any{}
					if !tc.omit {
						want[""] = nil
						if group != "none" {
							want = map[string]any{"root": want}
						}
					}
					require.Equal(t, want, record)
				})
			}
		}
	}
}

func TestJSONEmptyAttributeKeyPreservesShapeAndMaskPaths(t *testing.T) {
	t.Parallel()
	for _, grouped := range []bool{false, true} {
		for _, captured := range []bool{false, true} {
			for _, matched := range []bool{false, true} {
				t.Run(fmt.Sprintf("grouped=%t/captured=%t/matched=%t", grouped, captured, matched), func(t *testing.T) {
					t.Parallel()
					path := ".private"
					if grouped {
						path = "root." + path
					}
					opts := []masking.Option{masking.WithDefaults(true)}
					wantPrivate := "hidden"
					if matched {
						opts = append(opts, masking.WithRules(mask.Rule{Field: path, Mask: mask.Full()}))
						wantPrivate = "********"
					}
					var buf bytes.Buffer
					h := portedHandler(t, slog.NewJSONHandler(&buf, nil), opts...)
					logger := slog.New(h)
					if grouped {
						logger = logger.WithGroup("root")
					}
					input := map[string]any{"public": "visible", "private": "hidden"}
					attrs := []slog.Attr{slog.Any("", input), slog.String("private", "visible")}
					if captured {
						logger = slog.New(logger.Handler().WithAttrs(attrs))
						attrs = nil
					}
					logger.LogAttrs(t.Context(), slog.LevelInfo, "test", attrs...)
					var record map[string]any
					require.NoError(t, json.Unmarshal(buf.Bytes(), &record))
					delete(record, "time")
					delete(record, "level")
					delete(record, "msg")
					want := map[string]any{
						"":        map[string]any{"public": "visible", "private": wantPrivate},
						"private": "visible",
					}
					if grouped {
						want = map[string]any{"root": want}
					}
					require.Equal(t, want, record)
					require.Equal(t, map[string]any{"public": "visible", "private": "hidden"}, input)
				})
			}
		}
	}
}

func TestJSONEmptyKeysPreserveShapeAndMaskPaths(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  string
		path   string
		masked string
	}{
		{"object", `{"":{"public":"visible","private":"hidden"},"private":"visible"}`, "root.payload..private", `{"":{"public":"visible","private":"********"},"private":"visible"}`},
		{"null", `{"":null,"public":"visible"}`, "root.payload.", `{"":"********","public":"visible"}`},
		{"nested", `{"child":{"":null,"private":"hidden"}}`, "root.payload.child.private", `{"child":{"":null,"private":"********"}}`},
		{"array object", `[{"":{"public":"visible","private":"hidden"},"private":"visible"}]`, "root.payload..private", `[{"":{"public":"visible","private":"********"},"private":"visible"}]`},
		{"array null", `[{"":null,"public":"visible"}]`, "root.payload.", `[{"":"********","public":"visible"}]`},
	}
	for _, tc := range cases {
		for _, captured := range []bool{false, true} {
			for _, matched := range []bool{false, true} {
				name := tc.name + "/record"
				if captured {
					name = tc.name + "/captured"
				}
				if matched {
					name += "/masked"
				} else {
					name += "/preserved"
				}
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					var input any
					require.NoError(t, json.Unmarshal([]byte(tc.input), &input))
					path, want := "unmatched", tc.input
					if matched {
						path, want = tc.path, tc.masked
					}
					var buf bytes.Buffer
					h := portedHandler(t, slog.NewJSONHandler(&buf, nil),
						masking.WithNested(true),
						masking.WithRules(mask.Rule{Field: path, Mask: mask.Full()}))
					logger := slog.New(h).WithGroup("root")
					attr := slog.Any("payload", input)
					if captured {
						logger.With(attr).InfoContext(t.Context(), "test")
					} else {
						logger.LogAttrs(t.Context(), slog.LevelInfo, "test", attr)
					}
					var record map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(buf.Bytes(), &record))
					require.JSONEq(t, `{"payload":`+want+`}`, string(record["root"]))
					unchanged, err := json.Marshal(input)
					require.NoError(t, err)
					require.JSONEq(t, tc.input, string(unchanged))
				})
			}
		}
	}
}
