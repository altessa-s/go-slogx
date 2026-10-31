# attrwalk

```go
import "github.com/altessa-s/go-slogx/internal/attrwalk"
```

Bounded reflection walk used by the masking handler. Values past the depth budget are replaced with the fail-closed placeholder.

## Scope

The walk resolves a `slog.LogValuer` once, follows pointers with a cycle bound, and does not mask the log message.

Unmatched containers retain their original value and JSON shape. Each traversed container is limited to 256 entries plus a fail-closed placeholder. Mask callback panics produce `********`.

Non-string-keyed maps are replaced with `<masking: unsupported map keys>`,
including with fields-only rules. Struct fields tagged `json:"-"` are omitted
by rebuilding the container so native text formatting cannot reveal them.

Unexported anonymous fields cause their container to be replaced with
`<masking: unexported embedding>`, so promoted exported fields cannot leak.
Matched compound values are checked before formatting using the depth budget
and a total 256-node budget. Cycles or exhausted budgets emit a fixed
placeholder without invoking the mask callback.

TextMarshaler implementations do not bypass field traversal: JSON and console
sinks do not invoke the same methods, and pointer methods may not apply to
plain values. Matching fields are masked for both receiver forms.

Exported anonymous struct and pointer fields without a JSON name (including
options-only tags) replace their container with
`<masking: promoted embedding>`. This conservative policy avoids approximating
JSON promotion, shadowing, and ambiguous-field selection. Explicitly named
embeddings retain ordinary nested field traversal.
