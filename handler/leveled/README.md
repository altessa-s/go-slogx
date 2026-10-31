# leveled

```go
import "github.com/altessa-s/go-slogx/handler/leveled"
```

Middleware that enables a record from a dynamic default level and from static subsystem overrides.

## Level authority

`MinLevel` is the floor of the current default and the static overrides. The sink uses that floor so a subsystem that is more verbose
than the default is not dropped before this handler sees the record. A subsystem key inside a named group does not select an override. Empty-key groups are inline: their first subsystem string selects the override, including resolved LogValuer values. The resolved representation is forwarded so callbacks run once. Inline traversal is capped at 100 levels; deeper values become a fixed placeholder.

`leveled.New(next, opts...)` returns `(*Handler, error)` and rejects a nil next handler.
