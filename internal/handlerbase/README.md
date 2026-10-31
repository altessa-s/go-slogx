# handlerbase

```go
import "github.com/altessa-s/go-slogx/internal/handlerbase"
```

Embeddable state for a middleware handler: the next `slog.Handler` and the groups opened by `WithGroup`.

## Contract

`NewBase` panics when the next handler is nil. `WithGroup("")` is the caller's decision; this package only stores a non-empty name.
`GroupPath` joins names with dots.
