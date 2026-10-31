# go-slogx

```go
import "github.com/altessa-s/go-slogx"
```

Structured logging helpers for Go's `log/slog`: nil-safe attributes, context loggers, mask functions, and composable handlers.

The consumer supplies an `io.Writer`. `setup` builds one logger. The library reads no config file and registers no process hook.

## Packages

| Package | Role |
|---------|------|
| `slogx` | Attributes, request fields, context carriers, `LevelOff` |
| `mask` | Mask functions, rules, and an immutable catalog |
| `handler/leveled` | Dynamic level and per-subsystem overrides |
| `handler/masking` | Attribute redaction |
| `handler/prefixed` | Subsystem prefix for text and console output |
| `console` | Text sink with a private SGR encoder |
| `bufwriter` | Asynchronous `io.Writer` under the sink |
| `setup` | Builder: level, format, masks, buffer, and shutdown |
| `slogxtest` | Recorder, conformance helper, golden files |

## Chain

`setup` wires handlers as `leveled → masking → prefixed → sink`. `prefixed` is inserted for text and console output. JSON keeps the raw
`subsystem` attribute. The sink formats each record on the caller goroutine, then writes finished bytes through `bufwriter` when a buffer
is configured.

## Level

One `setup.Logger` owns one `*slog.LevelVar`. `slogx.LevelOff` is the off sentinel. The consumer maps its own `none` spelling onto that
constant before calling `setup`.

## Module

```
github.com/altessa-s/go-slogx
```

Go 1.27. Dependencies point inward: the root and `mask` use only the standard library;
middleware uses domain and internal packages, never sibling middleware or go-atlas-core.
`setup` alone composes the chain. The explicit package tables in `architecture_test.go`
enforce these boundaries across every module, along with dependency purity, generated
options, package documentation, and restrictions on initialization and exported mutable state.

## Usage

```go
logger, err := setup.New(output,
    setup.WithFormat(setup.JSON),
    setup.WithMaskDefaults(true),
    setup.WithBufferSize(100),
)
if err != nil {
    return err
}
logger.Logger().InfoContext(ctx, "ready", slogx.Module("worker"))
return logger.Shutdown(ctx)
```

All public configurable constructors return errors. There are no global level,
shutdown traversal, registry, hash mask, or fanout APIs. Compose multiple sinks
with the standard library's `slog.NewMultiHandler`.

Fields in a context are shallow snapshots; AppendField/AppendFields safely mutate
the request-local bag. Callers still own values contained in fields. Retrieval
from a nil context returns no stored fields/logger; context insertion requires a
valid context. ContextWithLogger replaces; ContextWithLoggerIfAbsent preserves.

## Verification

Run `make vet && make lint && make test-all && make test-integration && make generate`
and `make fuzz-seed`. Integration tests use real files, pipes, PTYs and a child
process; no Docker dependency exists. Overflow may reorder writes, so the
concurrency-order soak provisions capacity for its full workload.
