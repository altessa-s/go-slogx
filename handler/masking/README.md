# masking

```go
import "github.com/altessa-s/go-slogx/handler/masking"
```

Middleware that redacts attribute values. `New` is the only constructor and returns an error for an invalid rule.

## Fail closed

A value the walker cannot inspect inside the depth budget is replaced. The log message is left unchanged. Built-in masks come from `mask`.

`WithRules(rules...)` appends validated rule values. Exact dotted paths take
precedence over bare fields; patterns are considered after exact fields.
`WithNested(true)` enables dotted-path matching; `WithDefaults(true)` also enables
it and installs common credential and contact masks. Explicit rules override
defaults regardless of option order. Case-insensitive matching is the default.
The generated bool options accept an explicit bool.

With rules active, arbitrary values are normalized through JSON before masking. Text structs therefore use JSON-shaped groups, including JSON
field names and omission rules. Marshal errors and ordinary panics fail closed.
Traversal stops at depth 10 and limits each decoded container to 256 elements.
A defensive preflight also bounds the original value to 256 visited values and
depth 10 before encoding, preventing recursive pointer/interface input from
exhausting the encoder's stack. Unsupported JSON field names fail closed.
Values exceeding these limits become fixed placeholders without their contents.
Custom encoding callbacks and message text remain the consumer's responsibility.
