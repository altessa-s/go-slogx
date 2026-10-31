# console

```go
import "github.com/altessa-s/go-slogx/console"
```

Text sink with a private SGR encoder. Keys and values are quoted so a record cannot inject a newline or an ANSI sequence.

## Options

Level is a `slog.Leveler` read on every `Enabled` call. `ReplaceAttr` runs on resolved values, including groups and built-in keys.
Color defaults to false.

`New(writer, opts...)` returns `(*Handler, error)` and rejects a nil writer.
`WithColor(bool)` and `WithAddSource(bool)` are explicit switches. `WithKeyStyles`
uses typed SGR constants. Captured attributes are formatted at WithAttrs time.
Output uses quoted `"key"="value"` pairs; group names qualify keys with dots.
