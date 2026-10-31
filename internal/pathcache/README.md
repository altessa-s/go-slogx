# pathcache

```go
import "github.com/altessa-s/go-slogx/internal/pathcache"
```

Bounded cache of mask decisions keyed by attribute path. Entries above the size cap are not stored.

## Ownership

Each masking handler owns its cache. Nothing in this package is shared across loggers.

Sixteen independent shards limit contention for dynamic map keys. Each shard resets its epoch at 256 entries; paths longer than 256 bytes are not retained.
