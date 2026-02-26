# Integrating arenacheck into CI

`arenacheck` is a static analyzer for SafeArena usage. It catches unsafe patterns
(escaping `Ptr[T]`/`Slice[T]`, use-after-free, `.Get()` results leaking out of scope)
at compile time, before your tests even run.

## Standalone usage

```bash
# Install
GOEXPERIMENT=arenas go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest

# Run via go vet (recommended)
GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...

# Or run directly
GOEXPERIMENT=arenas arenacheck ./...
```

## GitHub Actions

Add a dedicated step after your existing lint step:

```yaml
- name: Install arenacheck
  run: GOEXPERIMENT=arenas go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest

- name: Run arenacheck
  run: GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...
```

Full example workflow:

```yaml
name: CI

on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6

      - name: Install arenacheck
        run: GOEXPERIMENT=arenas go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest

      - name: Run arenacheck
        run: GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...
```

## golangci-lint

golangci-lint does not natively support `go vet`-style plugin analyzers. The
`go vet -vettool` approach above is simpler and recommended.

If you use golangci-lint's `custom-gcl` plugin system, you can build a plugin binary,
but this requires a separate build step and is significantly more complex. The
`go vet -vettool` approach is the idiomatic way to run external analyzers alongside
golangci-lint.

## Makefile integration

If you have a `Makefile`, add arenacheck as a target:

```makefile
.PHONY: arenacheck

arenacheck:
	GOEXPERIMENT=arenas go vet -vettool=$(shell which arenacheck) ./...
```

Then run it with `make arenacheck`.

## What arenacheck detects

| Pattern | Diagnostic |
|---------|------------|
| Returning `Ptr[T]` or `Slice[T]` from a function | `safearena.Ptr escapes via return` |
| Storing `Ptr[T]` or `Slice[T]` to a global variable | `safearena.Ptr escapes via global variable` |
| Returning raw `*T` from `.Get()` | `raw pointer from safearena .Get() escapes via return` |
| Wrapping `Ptr[T]` in `interface{}` and returning | `safearena.Ptr escapes via return` |
| Using raw arena allocation after `Free()` | `use of arena allocation after Free()` |

## Known limitations

arenacheck uses single-function SSA analysis and does not currently detect:
- Escapes through struct fields
- Escapes through channels or maps
- `Ptr[T]` captured by closures or goroutines
- Interprocedural escape paths

For these patterns, rely on runtime safety checks (SafeArena panics on use-after-free)
and the race detector (`go test -race`).
