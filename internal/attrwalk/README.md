# attrwalk

```go
import "github.com/altessa-s/go-slogx/internal/attrwalk"
```

Bounded reflection walk used by the masking handler. Values past the depth budget are replaced with the fail-closed placeholder.

## Scope

The walk resolves a `slog.LogValuer` once, follows pointers with a cycle bound, and does not mask the log message.

Unmatched containers retain their original value and JSON shape. Each traversed container is limited to 256 entries plus a fail-closed placeholder. Mask callback panics produce `********`.
