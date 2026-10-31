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
	"github.com/altessa-s/go-slogx/mask"
)

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
