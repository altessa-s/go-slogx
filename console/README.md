# console

```go
import "github.com/altessa-s/go-slogx/console"
```

Text sink with a private SGR encoder. The default format keeps the quoted key/value representation; `FormatText` emits a compact, human-readable line.

## Options

Level is a `slog.Leveler` read on every `Enabled` call. `ReplaceAttr` runs on resolved values, including groups and built-in keys.
Color defaults to false.

`New(writer, opts...)` returns `(*Handler, error)` and rejects a nil writer.
`WithColor(bool)` and `WithAddSource(bool)` are explicit switches. `WithKeyStyles`
uses typed SGR constants. Captured attributes are formatted at WithAttrs time.
`WithFormat(FormatKeyValue)` is the default and preserves the existing output.
`WithFormat(FormatText)` writes the optional timestamp, a fixed-width level,
the unquoted message, then `key=value` attributes. Values are quoted and escaped
when needed; group names qualify keys with dots. Text timestamps appear when a
record has a nonzero time. Color styles apply to the level and attribute keys.
