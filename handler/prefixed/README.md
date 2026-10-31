# prefixed

```go
import "github.com/altessa-s/go-slogx/handler/prefixed"
```

Middleware that turns string values of the subsystem key into one prefix attribute. The default formatter is `Bracketed`.

## Placement

`setup` adds this handler on the text and console path only. JSON output keeps the original subsystem attribute. `WithGroup("")` returns
the same handler.

`prefixed.New(next, opts...)` returns `(*Handler, error)` and rejects a nil next handler.
