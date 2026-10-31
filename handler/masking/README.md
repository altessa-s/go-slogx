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
