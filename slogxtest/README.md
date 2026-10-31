# slogxtest

```go
import "github.com/altessa-s/go-slogx/slogxtest"
```

Helpers for tests of this module and of consumers. Production code does not import it.

## Helpers

`Recorder` captures resolved records. `Conformance` runs `testing/slogtest`. `Discard` returns a logger that drops records.
`GoroutineLeaks` delegates to go-atlas-core `leakcheck.Check`. `Golden` compares bytes with a file under `testdata`.

Construct recorders with `NewRecorder`; `Records` returns decoded JSON maps and
an error. `Conformance` takes the same handler and result callbacks as slogtest.Run.
`Golden(t, name, bytes, update)` resolves `name` below repository-root testdata.
Consumers can pass their own `-update` flag as the explicit final argument.
