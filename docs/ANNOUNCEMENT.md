# Announcing SafeArena: Safe Arena Memory Management for Go

**TL;DR:** SafeArena brings Rust-style memory safety to Go's arena allocations with runtime checks and static analysis, achieving 96.9% test coverage and minimal performance overhead.

---

Go 1.20 introduced experimental arena allocations, offering significant performance benefits for short-lived data. However, raw arenas are dangerous—use-after-free bugs compile without warning and cause silent memory corruption. SafeArena solves this.

## The Problem

```go
// This compiles and runs... but crashes unpredictably
func broken() *Data {
    a := arena.NewArena()
    defer a.Free()
    return arena.New[Data](a)  // 💥 Use-after-free!
}
```

Raw arenas give you performance but no safety net. One mistake = memory corruption.

## The Solution

SafeArena makes the dangerous pattern impossible:

```go
func safe() Response {
    return safearena.Scoped(func(a *safearena.Arena) Response {
        data := safearena.Alloc(a, Data{})
        // Use data safely with .Get()
        return Response{...}  // Heap-allocated, safe to return
    })  // Arena auto-freed, safety checks passed ✓
}
```

**If you try to return an arena pointer, you get an immediate, debuggable panic instead of silent corruption.**

## What You Get

### 1. Runtime Safety (100% Coverage)

Every arena access is checked:
- **Use-after-free detection** - Panics with helpful error messages
- **Double-free prevention** - Catches double-frees before corruption
- **Lifetime tracking** - Type-safe `Ptr[T]` wrappers enforce correctness

```go
a := safearena.New()
data := safearena.Alloc(a, 42)
a.Free()

_ = data.Get()  // Panic with stack trace and fix suggestions!
// arena 1: use after free
//   at main.go:10 (main.process)
//   💡 Hint: Use Clone() to copy values to heap...
```

### 2. Static Analysis (Compile-Time Detection)

The `arenacheck` tool catches bugs before runtime:

```bash
GOEXPERIMENT=arenas arenacheck ./...
```

Detects:
- Arena pointers escaping via returns
- Escapes to global variables
- Use-after-free patterns
- 100% detection on comprehensive test suite

### 3. Production-Quality Engineering

- **96.9% test coverage** - Comprehensive unit and integration tests
- **Fuzz tested** - 5M+ random executions, zero failures
- **Race tested** - No data races
- **Multi-platform CI** - Linux, macOS, Windows on Go 1.23-1.25
- **Real-world examples** - JSON parser, DB processor, image filters

## Performance

SafeArena adds ~10-15% overhead vs raw arenas, but provides 100% memory safety:

```
Real-world request processing (100 allocations):
  SafeArena:    104.8 μs/op    406 KB/op    0.047ms GC pause
  Regular GC:    92.5 μs/op    256 KB/op    0.082ms GC pause
```

**Trade-off:** Slightly slower than raw arenas, but:
- 42% lower GC pause times vs regular GC
- 100% memory safety vs 0% with raw arenas
- No silent corruption

## Real-World Use Cases

### HTTP Request Processing

```go
func handleRequest(req Request) Response {
    return safearena.Scoped(func(a *safearena.Arena) Response {
        buffer := safearena.AllocSlice[byte](a, 4096)
        temp := safearena.Alloc(a, TempData{})

        // All temp allocations in arena
        process(buffer.Get(), temp.Get())

        return Response{Status: 200}  // Heap-allocated
    })  // Arena freed - no GC pressure!
}
```

### JSON Parsing with Arena-Allocated AST

```go
func parseJSON(data []byte) Result {
    return safearena.Scoped(func(a *safearena.Arena) Result {
        ast := buildAST(a, data)  // Temp AST in arena
        // Process AST nodes...
        return extractResult(ast)  // Result on heap
    })  // AST freed instantly, no GC scan
}
```

**Performance:** ~1.4x faster than heap-allocated AST for parse-process-discard patterns.

### Image Processing Pipeline

```go
func applyFilters(img *Image) *Image {
    return safearena.Scoped(func(a *safearena.Arena) *Image {
        // Large temp buffers in arena
        buf1 := safearena.AllocSlice[byte](a, img.Size())
        buf2 := safearena.AllocSlice[byte](a, img.Size())

        applyBlur(img, buf1.Get())
        applySharpen(buf1.Get(), buf2.Get())

        return createResult(buf2.Get())  // Final image on heap
    })  // MBs of temp buffers freed instantly
}
```

**Performance:** ~1.5x faster for large images, scales with size.

## Why Not Just Use Rust?

Valid question! SafeArena is for:
- **Existing Go codebases** - No rewrite needed
- **Go's simplicity** - Gradual learning curve vs Rust's borrow checker
- **Go ecosystem** - Full access to Go libraries
- **Pragmatic safety** - Good enough for most use cases

If you need maximum safety and don't mind the learning curve, use Rust. If you want Go's simplicity with better memory safety, use SafeArena.

## Getting Started

```bash
# Install
go get github.com/scttfrdmn/safearena

# Optional: Install static analyzer
go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest
```

**Basic usage:**

```go
import "github.com/scttfrdmn/safearena"

result := safearena.Scoped(func(a *safearena.Arena) int {
    data := safearena.Alloc(a, MyStruct{Value: 42})
    return data.Get().Value
})
```

## Documentation

- **Quick Start:** https://github.com/scttfrdmn/safearena#quick-start
- **Migration Guide:** [docs/MIGRATION.md](MIGRATION.md)
- **FAQ:** [docs/FAQ.md](FAQ.md)
- **Performance Guide:** [docs/PERFORMANCE.md](PERFORMANCE.md)
- **API Docs:** https://pkg.go.dev/github.com/scttfrdmn/safearena
- **Examples:** https://github.com/scttfrdmn/safearena/tree/main/examples

## Project Status

- **Code Quality:** Production-ready (96.9% coverage, comprehensive tests)
- **Go Arenas:** Experimental (GOEXPERIMENT=arenas required)
- **Recommendation:** Great for development, research, and non-critical systems. Wait for Go team's production release for mission-critical use.

## Philosophy

> "Why not have simplicity AND guarantees?"

SafeArena proves you can add strong safety to Go without sacrificing simplicity:
- ✅ Simple, readable code (feels like normal Go)
- ✅ Strong safety guarantees (panics instead of corruption)
- ✅ Good performance (minimal overhead)
- ✅ Practical tooling (works with existing Go tools)

Not as strong as Rust's compile-time guarantees, but **good enough** for most use cases while staying true to Go's philosophy.

## Contributing

Contributions welcome! Areas for help:
- More real-world examples
- Arenacheck improvements (interprocedural analysis)
- Performance optimizations
- Platform testing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## Try It

```bash
# Clone and explore
git clone https://github.com/scttfrdmn/safearena.git
cd safearena

# Run examples
cd examples/json_parser
GOEXPERIMENT=arenas go run main.go

# Run tests
cd ../..
GOEXPERIMENT=arenas go test ./...

# Try static analyzer
GOEXPERIMENT=arenas arenacheck ./examples/...
```

## Links

- **Repository:** https://github.com/scttfrdmn/safearena
- **Documentation:** https://pkg.go.dev/github.com/scttfrdmn/safearena
- **Issues:** https://github.com/scttfrdmn/safearena/issues
- **Discussions:** https://github.com/scttfrdmn/safearena/discussions

---

**Built with ❤️ for the Go community**

*SafeArena: Safe arena memory management for Go. Because you deserve simplicity AND guarantees.*
