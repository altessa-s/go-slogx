# bufwriter

```go
import "github.com/altessa-s/go-slogx/bufwriter"
```

A bounded asynchronous `io.Writer` over a caller-provided sink. The worker
delivers accepted writes in queue order. When the queue fills, `Write` returns
`ErrDropped` and counts the newest write as dropped.

Use `WritePriority` with a deadline to wait for a queue slot and the delivery
result. Waiting priority writes get first access to newly available capacity.
If the deadline expires before admission, the write is dropped and counted as a
priority drop. An admitted write stays queued if waiting for delivery times out.

`Flush` waits for accepted writes and returns and resets accumulated delivery
errors. `Close(ctx)` stops admission, drains the queue, and returns cumulative
delivery counts and sink errors since the last flush. If the context ends, the
result includes its error and a snapshot of pending writes. A later close can
wait for the remaining writes. Writes after closing return `ErrClosed`.

The caller owns the sink. If its `Write` stalls, the caller must unblock it to
let the worker finish. `WithOnError` receives sink errors on the worker,
including short writes and recovered sink panics.
