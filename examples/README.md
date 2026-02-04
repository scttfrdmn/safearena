# SafeArena Examples

This directory contains real-world examples demonstrating SafeArena usage patterns and best practices.

## Examples

### 1. HTTP Server (`http_server/`)
**Pattern:** Request-scoped allocations

A simple HTTP server that uses arenas for request-specific temporary allocations. Each request gets its own arena that's automatically freed after the response is sent.

**Best for:**
- Web servers and APIs
- Request/response processing
- Per-connection buffers

[View Example](http_server/) | [View README](http_server/README.md)

---

### 2. JSON Parser (`json_parser/`)
**Pattern:** Arena-allocated AST for parsing

Demonstrates building temporary Abstract Syntax Trees during JSON parsing. The AST nodes live in the arena and are freed after processing, with only the final result on the heap.

**Best for:**
- Parsers and compilers
- Multi-pass data processing
- Temporary tree/graph structures

**Performance:** ~1.4x faster for parse-process-discard patterns

[View Example](json_parser/) | [View README](json_parser/README.md)

---

### 3. Database Query Processor (`database_processor/`)
**Pattern:** Request-scoped query processing

Shows how to use arenas for database middleware and query processing. Temporary buffers for filtering, string manipulation, and aggregation are arena-allocated per query.

**Best for:**
- Database middleware
- ORM/query builders
- Result set processing
- Batch operations

**Performance:** ~1.4x faster with lower GC pressure

[View Example](database_processor/) | [View README](database_processor/README.md)

---

### 4. Image Filter Pipeline (`image_filter/`)
**Pattern:** Multi-pass processing with large buffers

Demonstrates image processing with multiple filter passes. Large temporary buffers (MBs) are allocated in the arena and freed immediately after processing.

**Best for:**
- Image/video processing
- Computer vision pipelines
- Format conversion
- Any multi-pass processing with large buffers

**Performance:** ~1.5x faster, scales with image size

[View Example](image_filter/) | [View README](image_filter/README.md)

---

## Running Examples

All examples require Go 1.23+ with the experimental arena support:

```bash
# Navigate to any example directory
cd examples/json_parser

# Run with GOEXPERIMENT
GOEXPERIMENT=arenas go run main.go

# Or build first
GOEXPERIMENT=arenas go build
./json_parser
```

## Common Patterns

### 1. Scoped Pattern (Recommended)
```go
result := safearena.Scoped(func(a *safearena.Arena) Result {
    temp := safearena.Alloc(a, TempData{})
    // Use temp...
    return Result{} // Heap-allocated
}) // Arena freed automatically
```

### 2. Manual Management
```go
a := safearena.New()
defer a.Free()

data := safearena.Alloc(a, MyData{})
// Use data...
```

### 3. Large Buffers
```go
safearena.Scoped(func(a *safearena.Arena) Result {
    buffer := safearena.AllocSlice[byte](a, 1024*1024) // 1MB
    // Process with buffer...
    return result
})
```

## Performance Guidelines

| Buffer Size | Expected Benefit | Best Use Case |
|-------------|------------------|---------------|
| < 100 KB | Minimal (~5-10%) | Small requests, simple parsing |
| 100 KB - 1 MB | Moderate (~20-40%) | Medium images, JSON docs |
| 1 MB - 10 MB | Good (~40-60%) | Large images, video frames |
| > 10 MB | Significant (>60%) | High-res images, data processing |

Benefits increase with:
- Larger temporary allocations
- Higher allocation frequency
- More complex processing pipelines
- Longer-running processes

## Best Practices

1. **Use `Scoped()` by default** - Automatic cleanup, impossible to leak
2. **Temporary in arena, results on heap** - Clear ownership
3. **One arena per request/frame** - Natural lifecycle boundaries
4. **Profile before optimizing** - Measure to confirm benefits
5. **Don't return arena pointers** - Use return value of `Scoped()`

## Anti-Patterns to Avoid

❌ **Returning arena pointers**
```go
// DON'T DO THIS
func bad() *Data {
    return safearena.Scoped(func(a *safearena.Arena) *Data {
        return safearena.Alloc(a, Data{}).Get() // DANGER!
    })
}
```

❌ **Global/long-lived arenas**
```go
// DON'T DO THIS
var globalArena = safearena.New() // Never freed!
```

❌ **Escaping arena references**
```go
// DON'T DO THIS
var leaked safearena.Ptr[Data]
safearena.Scoped(func(a *safearena.Arena) int {
    leaked = safearena.Alloc(a, Data{}) // Escapes scope!
    return 0
})
```

## Contributing Examples

Have a great use case? Contributions welcome! Each example should:
- Demonstrate a real-world pattern
- Include benchmarks comparing arena vs non-arena
- Have a README explaining when to use the pattern
- Show best practices
- Be well-commented

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## Additional Resources

- [Main README](../README.md) - Project overview
- [API Documentation](https://pkg.go.dev/github.com/scttfrdmn/safearena)
- [Design Document](../docs/CREATIVE_SOLUTION.md)
- [Static Analyzer](../cmd/arenacheck/)
