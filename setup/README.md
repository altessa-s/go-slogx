# setup

```go
import "github.com/altessa-s/go-slogx/setup"
```

Builder for one logger. The caller passes an `io.Writer` and options. `Shutdown(ctx)` waits for buffered delivery within the context
and returns delivery statistics and errors.

## Chain

`leveled`, then `masking`, then `prefixed` when the format is text or console, then the sink. The sink writes through `bufwriter` when
buffering is on. Ordinary records are dropped when the queue fills. Records at
or above `WithBufferBypassLevel` get priority for newly available slots and can
wait up to `WithBufferBypassTimeout` for admission and delivery (default 100 ms).
A shorter caller deadline takes precedence. The buffer counts admission drops.

A buffered custom sink writes formatted bytes synchronously from `Handle`.
Buffered sink formatting is serialized across derived handlers, so formatting
callbacks must avoid reentering the same logger. JSON output omits `prefixed`.
Use `WithMessageMask` to opt in to redacting the log message.
