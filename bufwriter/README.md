# bufwriter

```go
import "github.com/altessa-s/go-slogx/bufwriter"
```

Bounded asynchronous `io.Writer` placed under a slog sink. The sink formats on the caller goroutine. This package never sees a `slog.Record`.

## Lifecycle

Overflow writes on the caller. A panic inside one write is reported and does not stop later writes. `Flush` returns after the worker has
exited. `Close` drains, honors the context, and is idempotent.

`New(writer, opts...)` returns `(*Writer, error)`. `WithSize` requires a positive
capacity. `WithOnError` receives every asynchronous or synchronous write failure;
the callback must be concurrency-safe and must not reenter the writer. Flush and
Close return the first write failure. Panic payloads are not included in errors.
Overflow can reorder output; ordering checks must avoid overflow. Underlying
writes cannot be forcibly canceled, but lifecycle waits honor their contexts.
