<div align="center">
  <img src="docs/logo.png" alt="SafeArena Logo" width="400"/>

# SafeArena

**Safe, ergonomic arena memory management for Go with compile-time and runtime safety checks.**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/safearena)](https://goreportcard.com/report/github.com/scttfrdmn/safearena)
[![codecov](https://codecov.io/gh/scttfrdmn/safearena/branch/main/graph/badge.svg)](https://codecov.io/gh/scttfrdmn/safearena)
[![Go Reference](https://pkg.go.dev/badge/github.com/scttfrdmn/safearena.svg)](https://pkg.go.dev/github.com/scttfrdmn/safearena)
[![Release](https://img.shields.io/github/v/release/scttfrdmn/safearena?style=flat)](https://github.com/scttfrdmn/safearena/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/scttfrdmn/safearena/ci.yml?branch=main&label=CI&style=flat)](https://github.com/scttfrdmn/safearena/actions)

</div>

Go's experimental arena package provides performance benefits but requires careful manual lifetime management. SafeArena makes arenas safe and easy to use by combining:

1. **Runtime Safety** - Type-safe wrappers that prevent use-after-free and double-free
2. **Static Analysis** - Compile-time checks to catch arena escapes and leaks

## The Problem

Raw Go arenas are fast but dangerous:

```go
func unsafe() *Data {
    a := arena.NewArena()
    defer a.Free()
    return arena.New[Data](a)  // 💥 Use-after-free! Silent corruption!
}
```

## The Solution

SafeArena makes this impossible:

```go
func safe() Response {
    return safearena.Scoped(func(a *safearena.Arena) Response {
        data := safearena.Alloc(a, Data{})
        // Use data safely...
        return Response{...}  // Heap-allocated, safe to return
    })  // Arena auto-freed, all safety checks passed ✓
}
```

## Features

### Runtime Safety (safearena package)

- ✅ **Panics on use-after-free** - No silent memory corruption
- ✅ **Panics on double-free** - Prevents crashes
- ✅ **Scoped pattern** - Impossible to leak references
- ✅ **Type-safe wrappers** - `Ptr[T]` and `Slice[T]` track lifetimes
- ✅ **Zero config** - Just import and use

### Static Analysis (arenacheck tool)

- ✅ **Catches escapes at compile time** - Find bugs before runtime
- ✅ **Integrates with go vet** - Standard tooling
- ✅ **SSA-based analysis** - Leverages Go compiler infrastructure

## Installation

```bash
# Get the library
go get github.com/scttfrdmn/safearena

# Install the analyzer (optional)
go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest
```

## Quick Start

### Basic Usage

```go
package main

import "github.com/scttfrdmn/safearena"

func processRequest(req Request) Response {
    // Scoped automatically frees arena on return
    return safearena.Scoped(func(a *safearena.Arena) Response {
        // Allocate temporary data in arena
        buffer := safearena.AllocSlice[byte](a, 4096)
        temp := safearena.Alloc(a, TempData{...})

        // Use them safely
        process(buffer.Get(), temp.Get())

        // Return heap-allocated response
        return Response{Status: 200}
    })
}
```

### Advanced: Manual Arena Management

```go
func advanced() {
    a := safearena.New()
    defer a.Free()

    // Allocate
    data := safearena.Alloc(a, MyStruct{...})

    // Use safely
    data.Get().DoSomething()

    // Copy to heap if needed
    heapCopy := safearena.Clone(data)

    // Arena freed automatically
}
```

### Static Analysis

Run arenacheck to catch bugs at compile time:

```bash
# As a standalone tool
GOEXPERIMENT=arenas arenacheck ./...

# With go vet
GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...
```

## Benchmarks

Real-world request processing (100 allocations per request):

```
BenchmarkSafeArena    104.8 μs/op    406 KB/op    0.047ms GC pause
BenchmarkRegularGC     92.5 μs/op    256 KB/op    0.082ms GC pause
```

**SafeArena trades ~13% performance for:**
- 42% lower GC pause times
- 100% memory safety guarantees
- No risk of silent corruption

## Why SafeArena?

### vs Raw Arenas

| Feature | Raw Arena | SafeArena |
|---------|-----------|-----------|
| Performance | ⚡ Fastest | ⚡ Fast (90%) |
| Safety | ❌ Manual | ✅ Automatic |
| Use-after-free | 💥 Crashes | ✅ Panics |
| Learning curve | 📚 High | 📖 Low |

### vs Rust

| Feature | Rust | SafeArena |
|---------|------|-----------|
| Safety guarantees | ✅ Compile-time | ⚠️ Runtime + Static |
| Learning curve | 📚 Steep | 📖 Gentle |
| Go compatibility | ❌ Different language | ✅ Pure Go |
| Ergonomics | ⚠️ Borrow checker | ✅ Simple patterns |

## Examples

See the [examples/](examples/) directory for complete examples:

- `go_arena_example.go` - Comparison of GC vs SafeArena
- Advanced patterns and use cases

## Documentation

- [Migration Guide](docs/MIGRATION.md) - Migrating from raw arenas to SafeArena
- [FAQ](docs/FAQ.md) - Frequently asked questions
- [Performance Guide](docs/PERFORMANCE.md) - Performance characteristics and optimization
- [Creative Solution](docs/CREATIVE_SOLUTION.md) - Deep dive into the design
- [Arenacheck Results](docs/ARENACHECK_RESULTS.md) - Static analyzer evaluation
- [Arenacheck README](cmd/arenacheck/README.md) - Static analyzer documentation

## Requirements

- Go 1.20+ with `GOEXPERIMENT=arenas`
- Currently experimental - not for production use yet

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) to get started.

**Quick Links:**
- [Contributing Guide](CONTRIBUTING.md) - Development setup, coding standards, PR process
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community guidelines
- [Security Policy](SECURITY.md) - Reporting vulnerabilities

**Areas for Contribution:**
- Improve arenacheck detection rates
- Add more test cases and examples
- Performance optimizations
- Documentation improvements
- Cross-platform testing

See [open issues](https://github.com/scttfrdmn/safearena/issues) for specific tasks.

## License

MIT License - see [LICENSE](LICENSE) file

## Credits

Inspired by the desire to have both simplicity (Go) and guarantees (Rust).

Built with ❤️ by [@scttfrdmn](https://github.com/scttfrdmn)

## Status

⚠️ **Experimental** - This project demonstrates safe arena memory management in Go. The arena package itself is experimental in Go. Use at your own risk.

## Philosophy

> "Why not have simplicity AND guarantees?"

SafeArena shows it's possible to add strong safety guarantees to Go without sacrificing simplicity. You get:

- ✅ Simple, readable code (feels like normal Go)
- ✅ Strong safety guarantees (panics instead of corruption)
- ✅ Good performance (minimal overhead)
- ✅ Practical tooling (works with existing Go tools)

Not as strong as Rust's compile-time guarantees, but **good enough** for most use cases while staying true to Go's philosophy.
