# pathcache

```go
import "github.com/altessa-s/go-slogx/internal/pathcache"
```

Bounded cache of mask decisions keyed by attribute path. Entries above the size cap are not stored.

## Ownership

Each masking handler owns its cache. Nothing in this package is shared across loggers.
