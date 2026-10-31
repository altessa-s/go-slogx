# setup

```go
import "github.com/altessa-s/go-slogx/setup"
```

Builder for one logger. The caller passes an `io.Writer` and optgen options. `Shutdown` drains buffers this builder created.

## Chain

`leveled`, then `masking`, then `prefixed` when the format is text or console, then the sink. The sink writes through `bufwriter` when
buffering is on. A record at or above the bypass level is flushed after the sink returns. JSON output does not insert `prefixed`.
