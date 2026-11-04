// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/mask"
	"github.com/altessa-s/go-slogx/setup"

	slogx "github.com/altessa-s/go-slogx"
)

type jsonPayload struct{ mode string }

func (v jsonPayload) MarshalJSON() ([]byte, error) {
	switch v.mode {
	case "panic":
		panic("canary")
	case "error":
		return nil, errors.New("canary")
	case "invalid":
		return []byte("canary"), nil
	default:
		return []byte(`{"password":"canary","public":"visible","number":9007199254740993}`), nil
	}
}

type resolvedJSON struct{}

func (resolvedJSON) LogValue() slog.Value         { return slog.AnyValue(jsonPayload{}) }
func (resolvedJSON) MarshalJSON() ([]byte, error) { panic("LogValue must precede JSON") }

func TestJSONRepresentationBypasses(t *testing.T) {
	t.Parallel()

	cyclic := map[string]any{}
	cyclic["self"] = cyclic

	var pointer any

	pointer = &pointer

	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"R2 integer map", map[int]any{1: map[string]string{"password": "canary"}}, "********"},
		{"R2 unsupported map", map[bool]string{true: "canary"}, "JSON encoding failed"},
		{"R3 omitted", struct {
			Password string `json:"-"`
			Public   string
		}{"canary", "visible"}, "visible"},
		{"R5 private embedding", embeddedValue{credentials{"canary"}}, "********"},
		{"R6 map cycle", cyclic, "max depth exceeded"},
		{"R6 pointer cycle", pointer, "max depth exceeded"},
		{"R8 pointer text value", pointerTextPayload{"canary", "visible"}, "********"},
		{"R8 pointer text pointer", &pointerTextPayload{"canary", "visible"}, "safe"},
		{"R8 value text", valueTextPayload{"canary", "visible"}, "safe"},
		{"R9 promoted", struct{ Credentials }{Credentials{"canary"}}, "********"},
		{"R10 private map", struct {
			credentials map[string]string
			Public      string
		}{map[string]string{"password": "canary"}, "visible"}, "visible"},
		{"R10 private struct", struct {
			credentials credentials
			Public      string
		}{credentials{"canary"}, "visible"}, "visible"},
		{"R11 invalid name", struct {
			Password string `json:"public\\name"`
		}{"canary"}, "JSON encoding failed"},
		{"R11 invalid embedding", struct {
			Credentials `json:"public\\name"`
		}{Credentials{"canary"}}, "JSON encoding failed"},
		{"custom JSON", jsonPayload{}, "9007199254740993"},
		{"custom JSON nested", struct{ Child jsonPayload }{jsonPayload{}}, "9007199254740993"},
		{"JSON panic", jsonPayload{"panic"}, "JSON encoding failed"},
		{"JSON error", jsonPayload{"error"}, "JSON encoding failed"},
		{"invalid JSON", jsonPayload{"invalid"}, "JSON encoding failed"},
		{"unsupported value", make(chan int), "JSON encoding failed"},
		{"resolved first", resolvedJSON{}, "********"},
	}
	for _, tc := range cases {
		for _, format := range []setup.Format{setup.Text, setup.JSON} {
			for _, defaults := range []bool{false, true} {
				for _, captured := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%v/defaults=%v/captured=%v", tc.name, format, defaults, captured), func(t *testing.T) {
						t.Parallel()

						var out bytes.Buffer

						l, err := setup.New(&out, setup.WithFormat(format), setup.WithMaskDefaults(defaults), setup.WithMaskRules(mask.Rule{Field: "password"}))
						require.NoError(t, err)

						attr := slog.Any("payload", tc.value)
						h := l.Logger().Handler()
						r := slog.Record{Message: "record"}

						if captured {
							h = h.WithAttrs([]slog.Attr{attr})
						} else {
							r.AddAttrs(attr)
						}

						require.NoError(t, h.Handle(t.Context(), r))
						require.NotContains(t, out.String(), "canary")
						require.Contains(t, out.String(), tc.want)

						if format == setup.JSON {
							require.True(t, json.Valid(out.Bytes()))
						}

						require.NoError(t, l.Shutdown(t.Context()))
					})
				}
			}
		}
	}
}

func TestReviewCacheAndCallbackBypasses(t *testing.T) {
	t.Parallel()

	for _, format := range []setup.Format{setup.Text, setup.JSON} {
		for _, captured := range []bool{false, true} {
			for _, callback := range []string{"direct", "catalog"} {
				t.Run(fmt.Sprintf("%v/%v/%s", format, captured, callback), func(t *testing.T) {
					t.Parallel()

					fn := mask.Func(func(string) string { panic("canary") })

					if callback == "catalog" {
						var err error

						fn, err = mask.Regexp(".*", fn)
						require.NoError(t, err)
					}

					var out bytes.Buffer

					l, err := setup.New(&out, setup.WithFormat(format), setup.WithMaskDefaults(true), setup.WithMaskRules(mask.Rule{Field: "password", Mask: fn}))
					require.NoError(t, err)
					// R1: warm the path cache with a literal key before a grouped key.
					log := l.Logger()
					log.Info("warm", "account.password", "public")

					attr := slog.Group("account", slog.String("password", "canary"))
					if captured {
						log.With(attr).Info("record")
					} else {
						log.Info("record", attr)
					}

					require.NotContains(t, out.String(), "canary")
					require.Contains(t, out.String(), "********")
				})
			}
		}
	}
}

type subsystemValue string

func (v subsystemValue) LogValue() slog.Value { return slog.StringValue(string(v)) }

func TestR7SubsystemRepresentations(t *testing.T) {
	t.Parallel()

	for _, format := range []setup.Format{setup.Text, setup.JSON} {
		for _, captured := range []bool{false, true} {
			for _, disabled := range []bool{false, true} {
				for _, inline := range []bool{false, true} {
					t.Run(fmt.Sprintf("%v/captured=%v/disabled=%v/inline=%v", format, captured, disabled, inline), func(t *testing.T) {
						t.Parallel()

						var out bytes.Buffer

						threshold := slog.LevelDebug
						if disabled {
							threshold = slogx.LevelOff
						}

						l, err := setup.New(&out, setup.WithFormat(format), setup.WithMaskDefaults(true), setup.WithSubsystemLevels(map[string]slog.Level{"test": threshold}))
						require.NoError(t, err)

						attr := slog.Any("subsystem", subsystemValue("test"))
						if inline {
							attr = slog.GroupAttrs("", attr)
						}

						log := l.Logger()
						if captured {
							log = log.With(attr)
						}

						if captured {
							log.Debug("record")
						} else {
							log.Debug("record", attr)
						}

						if disabled {
							require.Empty(t, out.String())
						} else {
							require.Contains(t, out.String(), "record")
						}
					})
				}
			}
		}
	}
}
