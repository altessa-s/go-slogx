# attrwalk

```go
import "github.com/altessa-s/go-slogx/internal/attrwalk"
```

Masks resolved attributes and the `encoding/json` representation of `KindAny` values. JSON is decoded with `UseNumber`, preserving numeric
precision. Objects become slog groups; arrays, empty objects, and objects containing empty keys remain generic JSON values to preserve their shape.
Original objects never reach the sink when rules are active. JSON controls private fields, tags, embedding, omission, and custom encoding.
Attribute-level `LogValuer` is resolved first;
inside a `KindAny` object, JSON's own method semantics apply.

Traversal stops at depth ten and 256 entries per container. Marshal failures and recoverable panics produce `<masking: JSON encoding failed>`;
mask callback panics produce `********`. No error or panic payload is emitted. Messages are outside the masking policy.

A conservative reflection preflight limits source depth to ten and total source nodes to 256 before invoking JSON. It rejects unsupported JSON tag names because the pinned encoder truncates some invalid names instead of falling back.
It does not choose field names or decide embedding promotion. This guards a fatal pointer/interface-cycle stack overflow in the pinned Go 1.27.3 encoder, which cannot be recovered. Source
budget exhaustion replaces the entire value, including large containers or private subtrees JSON might otherwise omit. Custom marshalers
can still allocate, block, or recurse internally; recover only contains ordinary panics, not runtime-fatal errors.
