# masking

```go
import "github.com/altessa-s/go-slogx/handler/masking"
```

Middleware that redacts attribute values and optionally log messages. `New` is the only constructor and returns an error for an invalid rule.

## Fail closed

A value the walker cannot inspect inside the depth budget is replaced. Built-in masks come from `mask`.
Use `WithMessageMask(mask.Func)` to opt in to masking the whole log message. The
message mask applies independently of attribute rules. A panicking mask replaces
the message with a fixed placeholder.

`WithRules(rules...)` appends validated rule values. Exact dotted paths take
precedence over bare fields. Patterns are considered after exact fields and match
either the dotted path or leaf key. `WithDefaults(true)` installs common
credential and contact masks. Explicit rules override
defaults regardless of option order. Case-insensitive matching is the default.
The generated bool options accept an explicit bool.

With rules active, arbitrary values are normalized through JSON before masking. Text structs therefore use JSON-shaped groups, including JSON
field names and omission rules. Marshal errors and ordinary panics fail closed.
Traversal stops at depth 10 and limits each decoded container to 256 elements.
A defensive preflight also bounds the original value to 256 visited values and
depth 10 before encoding, preventing recursive pointer/interface input from
exhausting the encoder's stack. Unsupported JSON field names fail closed.
Values exceeding these limits become fixed placeholders without their contents.
Custom encoding and mask callbacks remain the consumer's responsibility.
