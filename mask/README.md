# mask

```go
import "github.com/altessa-s/go-slogx/mask"
```

Redaction functions and an immutable catalog of named masks. A rule names either one field or one pattern.

## Builtins

`Full`, `Fixed`, `Partial`, `Email`, `Phone`, `CreditCard`, `URL`, and `Regexp`. `Partial` counts runes and rejects a negative keep count.
`URL` parses with `net/url`, drops a userinfo password, and masks query values.

## Catalog

`NewCatalog` copies the masks it is given. `Resolve` returns `ErrUnknownMask` or `ErrParams`. There is no process-wide registry.

`NewCatalog(opts...)` returns `(*Catalog, error)` and rejects empty names and nil
entries. `WithMasks(map[string]Func)` and `WithFactories(map[string]Factory)` are
generated setters. Catalog names are sorted. Built-ins include full, email, phone,
credit_card, url, partial, fixed, and pattern. Fixed requires a nonempty `value`;
pattern requires a valid `pattern`, with either `replacement` or a replacement mask.

Phone and credit-card masks retain four leading and two trailing runes. Email
retains the local part's first/last runes, with at most five asterisks between.
Short phone values are fully masked; partial masks fully hide overlapping ranges.

`URL` retains the scheme and host, removes userinfo and fragments, masks query values, and masks the entire non-root path, including its last segment.
