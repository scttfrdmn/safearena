# arenacheck - Static Analyzer for Arena Safety

arenacheck is a static analysis tool that detects unsafe arena usage patterns at compile time.

## Features

- ✅ Detects arena allocations escaping via return statements
- ✅ Detects use-after-free patterns
- ✅ Detects escapes to global variables
- ✅ Tracks allocations through local variables
- ✅ Integrates with `go vet`

## Installation

```bash
go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest
```

## Usage

### Standalone

```bash
GOEXPERIMENT=arenas arenacheck ./...
GOEXPERIMENT=arenas arenacheck path/to/file.go
```

### With go vet

```bash
GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...
```

### In CI/CD

Add to your GitHub Actions workflow:

```yaml
- name: Run arenacheck
  run: |
    go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest
    GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...
  env:
    GOEXPERIMENT: arenas
```

## What it Detects

### 1. Direct Return Escape

```go
func bad() *int {
    a := arena.NewArena()
    defer a.Free()
    return arena.New[int](a) // ERROR: escapes via return
}
```

### 2. Indirect Return Escape

```go
func bad() *int {
    a := arena.NewArena()
    defer a.Free()
    x := arena.New[int](a)
    return x // ERROR: escapes via return
}
```

### 3. Global Variable Escape

```go
var global *int

func bad() {
    a := arena.NewArena()
    defer a.Free()
    global = arena.New[int](a) // ERROR: escapes to global
}
```

### 4. Use After Free

```go
func bad() int {
    a := arena.NewArena()
    x := arena.New[int](a)
    a.Free()
    return *x // ERROR: use after free
}
```

## Current Detection Rate

Tested on comprehensive suite of 20 patterns:
- **Direct escapes:** 100%
- **Indirect escapes:** 100%
- **Use-after-free:** 100%
- **Global escapes:** 100%

See [testdata/comprehensive/](testdata/comprehensive/) for test cases.

## Limitations

Current limitations (future improvements):

1. **Interprocedural analysis:** Doesn't track across function boundaries
   ```go
   func helper(a *arena.Arena) *int {
       return arena.New[int](a) // Not detected yet
   }
   ```

2. **Interface escapes:** Limited detection through interfaces
   ```go
   func bad() interface{} {
       a := arena.NewArena()
       return arena.New[int](a) // May not detect
   }
   ```

3. **Closure escapes:** Limited detection through closures
   ```go
   func bad() func() *int {
       a := arena.NewArena()
       x := arena.New[int](a)
       return func() *int { return x } // May not detect
   }
   ```

4. **Complex dataflow:** Very complex paths may not be tracked

## How It Works

arenacheck uses SSA (Static Single Assignment) analysis:

1. **Arena Tracking:** Identifies `arena.NewArena()` calls
2. **Allocation Tracking:** Finds `arena.New[T]()` calls and associates with arenas
3. **Dataflow Analysis:** Tracks allocations through variables using store/load chains
4. **Escape Detection:** Checks if allocations escape via returns, globals, or outlive arena
5. **Use-After-Free:** Detects accesses after `arena.Free()` calls

## Examples

### Safe Code

```go
// ✅ Returns value, not pointer
func safe1() int {
    a := arena.NewArena()
    defer a.Free()
    x := arena.New[int](a)
    return *x
}

// ✅ Doesn't escape
func safe2() {
    a := arena.NewArena()
    defer a.Free()
    x := arena.New[int](a)
    fmt.Println(*x)
}

// ✅ Clone to heap
func safe3() *int {
    a := arena.NewArena()
    defer a.Free()
    x := arena.New[int](a)
    result := new(int)
    *result = *x
    return result
}
```

### Unsafe Code

```go
// ❌ Returns arena pointer
func unsafe1() *int {
    a := arena.NewArena()
    defer a.Free()
    return arena.New[int](a)
}

// ❌ Use after free
func unsafe2() int {
    a := arena.NewArena()
    x := arena.New[int](a)
    a.Free()
    return *x
}

// ❌ Escapes to global
var global *int
func unsafe3() {
    a := arena.NewArena()
    defer a.Free()
    global = arena.New[int](a)
}
```

## Configuration

Currently no configuration options. The analyzer is designed to be conservative to avoid false positives.

## Performance

arenacheck typically adds:
- ~100-500ms to build time for small projects
- ~1-3s for medium projects
- Scales linearly with codebase size

## Debugging

Enable verbose output:

```bash
GOEXPERIMENT=arenas arenacheck -v ./...
```

## Contributing

Improvements welcome! Areas for contribution:

1. **Interprocedural analysis** - Track across function calls
2. **Better interface handling** - Detect escapes through interfaces
3. **Closure analysis** - Track through closures
4. **Error messages** - More helpful suggestions
5. **Test coverage** - More edge cases

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## References

- [SSA Package Documentation](https://pkg.go.dev/golang.org/x/tools/go/ssa)
- [Writing Custom Analyzers](https://pkg.go.dev/golang.org/x/tools/go/analysis)
- [Go Vet Guide](https://go.dev/doc/cmd/vet)

## License

MIT License - see [LICENSE](../../LICENSE)
