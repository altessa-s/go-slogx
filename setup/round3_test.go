// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/mask"
	"github.com/altessa-s/go-slogx/setup"
)

type pointerTextPayload struct {
	Password string
	Public   string
}

func (*pointerTextPayload) MarshalText() ([]byte, error) { return []byte("safe"), nil }

type valueTextPayload struct {
	Password string
	Public   string
}

func (valueTextPayload) MarshalText() ([]byte, error) { return []byte("safe"), nil }

func TestTextMarshalerFieldsRemainMasked(t *testing.T) {
	t.Parallel()
	for _, format := range []setup.Format{setup.JSON, setup.Text} {
		for _, defaults := range []bool{false, true} {
			for _, captured := range []bool{false, true} {
				for _, nested := range []bool{false, true} {
					for _, input := range []any{
						pointerTextPayload{"canary", "visible"},
						&pointerTextPayload{"canary", "visible"},
						valueTextPayload{"canary", "visible"},
						&valueTextPayload{"canary", "visible"},
					} {
						t.Run(fmt.Sprintf("%v/defaults=%v/captured=%v/nested=%v/%T", format, defaults, captured, nested, input), func(t *testing.T) {
							t.Parallel()
							var out bytes.Buffer
							opts := []setup.Option{setup.WithFormat(format), setup.WithMaskDefaults(defaults)}
							if !defaults {
								opts = append(opts, setup.WithMaskRules(mask.Rule{Field: "Password", Mask: mask.Full()}))
							}
							logger, err := setup.New(&out, opts...)
							require.NoError(t, err)
							if nested {
								input = struct{ Child any }{input}
							}
							log := logger.Logger()
							if captured {
								log.With("payload", input).Info("record")
							} else {
								log.Info("record", "payload", input)
							}
							require.NotContains(t, out.String(), "canary")
							require.Contains(t, out.String(), "********")
							require.Contains(t, out.String(), "visible")
							require.NoError(t, logger.Shutdown(t.Context()))
						})
					}
				}
			}
		}
	}
}

// Exported embedded types exercise encoding/json's promoted namespace.
type Credentials struct {
	Password string `json:"password"`
}
type OtherCredentials struct {
	Password string `json:"password"`
}

func TestPromotedEmbeddingsFailClosed(t *testing.T) {
	t.Parallel()
	// Construct intentional JSON tag ambiguity dynamically: go vet correctly
	// rejects this shape in a static declaration.
	conflict := reflect.New(reflect.StructOf([]reflect.StructField{
		{Name: "Credentials", Type: reflect.TypeFor[Credentials](), Anonymous: true},
		{Name: "OtherCredentials", Type: reflect.TypeFor[OtherCredentials](), Anonymous: true},
	})).Elem()
	conflict.Field(0).Set(reflect.ValueOf(Credentials{"canary"}))
	conflict.Field(1).Set(reflect.ValueOf(OtherCredentials{"canary"}))
	for _, format := range []setup.Format{setup.JSON, setup.Text} {
		for _, captured := range []bool{false, true} {
			for _, input := range []any{
				struct{ Credentials }{Credentials{"canary"}},
				struct{ *Credentials }{&Credentials{"canary"}},
				&struct{ Credentials }{Credentials{"canary"}},
				&struct{ *Credentials }{&Credentials{"canary"}},
				struct{ *Credentials }{},
				struct {
					*Credentials `json:",omitempty"`
				}{&Credentials{"canary"}},
				struct {
					Credentials
					Password string `json:"password"`
				}{Credentials{"canary"}, "canary"},
				conflict.Interface(),
			} {
				t.Run(fmt.Sprintf("%v/captured=%v/%T", format, captured, input), func(t *testing.T) {
					t.Parallel()
					var out bytes.Buffer
					logger, err := setup.New(&out, setup.WithFormat(format),
						setup.WithMaskRules(mask.Rule{Field: "payload.password", Mask: mask.Full()}))
					require.NoError(t, err)
					log := logger.Logger()
					if captured {
						log.With("payload", input).Info("record")
					} else {
						log.Info("record", "payload", input)
					}
					require.NotContains(t, out.String(), "canary")
					require.Contains(t, out.String(), "promoted embedding")
					require.NoError(t, logger.Shutdown(t.Context()))
				})
			}
		}
	}
}
