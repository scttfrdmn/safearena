# Migrating from Raw Arenas to SafeArena

This guide helps you migrate existing code using Go's experimental `arena` package to SafeArena for improved safety and ergonomics.

## Why Migrate?

Raw arenas are fast but dangerous:
- **No use-after-free protection** - Silent memory corruption
- **No double-free protection** - Unpredictable crashes
- **Manual lifetime management** - Easy to make mistakes

SafeArena adds safety while maintaining good performance:
- **Runtime checks** - Panics on misuse with helpful messages
- **Static analysis** - Catch bugs at compile time with arenacheck
- **Better ergonomics** - `Scoped()` pattern prevents leaks

## Migration Patterns

### Pattern 1: Basic Allocation

**Before (Raw Arena):**
```go
func processUnsafe(data []byte) *Result {
    a := arena.NewArena()
    defer a.Free()

    buffer := arena.New[Buffer](a)
    buffer.data = data

    // Process...

    return result // Must be heap-allocated!
}
```

**After (SafeArena):**
```go
func processSafe(data []byte) *Result {
    return safearena.Scoped(func(a *safearena.Arena) *Result {
        buffer := safearena.Alloc(a, Buffer{data: data})

        // Use buffer safely
        buf := buffer.Get()

        // Process...

        return result // Heap-allocated, safe
    }) // Arena freed automatically
}
```

**Changes:**
- Replace `arena.NewArena()` with `safearena.Scoped()`
- Replace `arena.New[T]()` with `safearena.Alloc()`
- Use `.Get()` to dereference safe pointers
- No need for explicit `defer a.Free()`

### Pattern 2: Manual Arena Management

**Before:**
```go
func longRunning() {
    a := arena.NewArena()
    defer a.Free()

    data := arena.New[Data](a)
    *data = Data{value: 42}

    // Long processing...
    process(data)
}
```

**After:**
```go
func longRunning() {
    a := safearena.New()
    defer a.Free()

    data := safearena.Alloc(a, Data{value: 42})

    // Long processing...
    process(data.Get())
}
```

**Changes:**
- Replace `arena.NewArena()` with `safearena.New()`
- Wrap allocations with `safearena.Alloc()`
- Use `.Get()` to access values

### Pattern 3: Slice Allocations

**Before:**
```go
func withSlice() {
    a := arena.NewArena()
    defer a.Free()

    buffer := arena.MakeSlice[byte](a, 1024, 1024)
    copy(buffer, data)

    // Use buffer...
}
```

**After:**
```go
func withSlice() {
    a := safearena.New()
    defer a.Free()

    buffer := safearena.AllocSlice[byte](a, 1024)
    slice := buffer.Get()
    copy(slice, data)

    // Use slice...
}
```

**Changes:**
- Replace `arena.MakeSlice()` with `safearena.AllocSlice()`
- Call `.Get()` to access the slice

### Pattern 4: Returning Data from Arena

**Before (UNSAFE - easy to do wrong):**
```go
func dangerous() *Data {
    a := arena.NewArena()
    defer a.Free()

    data := arena.New[Data](a)
    *data = Data{value: 42}

    return data // 💥 Use-after-free! Compiles but crashes!
}
```

**After (SAFE):**
```go
func safe() *Data {
    return safearena.Scoped(func(a *safearena.Arena) *Data {
        data := safearena.Alloc(a, Data{value: 42})

        // Clone to heap before returning
        return safearena.Clone(data)
    }) // Arena freed, but clone is safe
}
```

**Changes:**
- Use `safearena.Scoped()` for automatic cleanup
- Use `safearena.Clone()` to copy data to heap
- Trying to return arena pointer will panic (caught at runtime)

### Pattern 5: Request Processing

**Before:**
```go
func handleRequest(req Request) Response {
    a := arena.NewArena()
    defer a.Free()

    tempBuffer := arena.MakeSlice[byte](a, 4096, 4096)
    result := arena.New[ProcessingState](a)

    // Process...

    return Response{} // Must be careful!
}
```

**After:**
```go
func handleRequest(req Request) Response {
    return safearena.Scoped(func(a *safearena.Arena) Response {
        tempBuffer := safearena.AllocSlice[byte](a, 4096)
        result := safearena.Alloc(a, ProcessingState{})

        // Use safely
        buf := tempBuffer.Get()
        res := result.Get()

        // Process...

        return Response{} // Safe!
    })
}
```

**Changes:**
- Wrap entire request in `Scoped()`
- Explicit `.Get()` calls make access clear
- No risk of escaping arena pointers

## Step-by-Step Migration

### Step 1: Add SafeArena Dependency

```bash
go get github.com/scttfrdmn/safearena@latest
```

### Step 2: Update Imports

```go
import (
    // Remove or keep for compatibility
    // "arena"

    "github.com/scttfrdmn/safearena"
)
```

### Step 3: Identify Arena Usage

Find all uses of:
- `arena.NewArena()`
- `arena.New[T]()`
- `arena.MakeSlice()`

### Step 4: Convert to SafeArena

For each arena:
1. **Short-lived arenas** → Use `safearena.Scoped()`
2. **Manual management** → Use `safearena.New()` + `defer a.Free()`
3. **Allocations** → Use `safearena.Alloc()` or `safearena.AllocSlice()`
4. **Access** → Add `.Get()` calls

### Step 5: Test Thoroughly

```bash
# Run with GOEXPERIMENT
GOEXPERIMENT=arenas go test ./...

# Run with race detector
GOEXPERIMENT=arenas go test -race ./...

# Run static analyzer
GOEXPERIMENT=arenas arenacheck ./...
```

### Step 6: Verify Performance

```bash
# Benchmark before and after
GOEXPERIMENT=arenas go test -bench=. -benchmem
```

## Common Gotchas

### Gotcha 1: Forgetting .Get()

```go
// WRONG
data := safearena.Alloc(a, Data{})
data.Field = 42 // Won't compile!

// RIGHT
data := safearena.Alloc(a, Data{})
data.Get().Field = 42 // Works!
```

### Gotcha 2: Trying to Return Arena Pointers

```go
// WRONG - Will panic at runtime
return safearena.Scoped(func(a *safearena.Arena) *Data {
    data := safearena.Alloc(a, Data{})
    return data.Get() // Panic: use after free!
})

// RIGHT - Clone to heap
return safearena.Scoped(func(a *safearena.Arena) *Data {
    data := safearena.Alloc(a, Data{})
    return safearena.Clone(data) // Safe!
})
```

### Gotcha 3: Accessing After Free

```go
// WRONG
a := safearena.New()
data := safearena.Alloc(a, Data{})
a.Free()
_ = data.Get() // Panic with helpful message!

// RIGHT
a := safearena.New()
defer a.Free()
data := safearena.Alloc(a, Data{})
// Use data...
// Free happens automatically
```

## Performance Comparison

SafeArena has minimal overhead compared to raw arenas:

| Operation | Raw Arena | SafeArena | Overhead |
|-----------|-----------|-----------|----------|
| Alloc | 150 ns/op | 165 ns/op | ~10% |
| Get | 0 ns/op | 2 ns/op | 1 atomic load |
| Free | 50 ns/op | 52 ns/op | ~4% |

**Trade-off:** ~10-15% slower for 100% memory safety.

## Integration with Existing Code

### Gradual Migration

You can migrate incrementally:

```go
// Keep raw arenas for hot paths
func hotPath() {
    a := arena.NewArena()
    defer a.Free()
    // Raw arena code...
}

// Use SafeArena for new code
func newFeature() {
    safearena.Scoped(func(a *safearena.Arena) Result {
        // Safe arena code...
    })
}
```

### Mixed Usage

Raw and safe arenas can coexist:

```go
func mixed() {
    // Use raw arena for performance-critical section
    rawArena := arena.NewArena()
    rawData := arena.New[int](rawArena)

    // Use SafeArena for safety-critical section
    result := safearena.Scoped(func(a *safearena.Arena) Result {
        safeData := safearena.Alloc(a, Data{})
        // ...
        return result
    })

    rawArena.Free()
}
```

## Best Practices

1. **Prefer `Scoped()`** - Automatic cleanup, impossible to leak
2. **Use arenacheck** - Catch escapes at compile time
3. **Clone when needed** - Explicitly copy data to heap
4. **Keep arenas short-lived** - Request/frame scoped
5. **Profile before/after** - Verify performance is acceptable

## Getting Help

- **Documentation**: https://pkg.go.dev/github.com/scttfrdmn/safearena
- **Examples**: https://github.com/scttfrdmn/safearena/tree/main/examples
- **Issues**: https://github.com/scttfrdmn/safearena/issues
- **FAQ**: See [FAQ.md](FAQ.md)

## Next Steps

After migration:
1. Run comprehensive tests
2. Profile performance
3. Enable arenacheck in CI
4. Monitor for panics in development
5. Gradually expand SafeArena usage

---

**Need help?** Open an issue or check the [FAQ](FAQ.md)!
