# JSON Parser with Arena-Allocated AST

This example demonstrates using SafeArena for parsing JSON with temporary Abstract Syntax Tree (AST) nodes.

## Use Case

When parsing structured data like JSON, XML, or protocol buffers, you often build temporary data structures (ASTs, parse trees) that are only needed during processing. These temporary structures can generate significant GC pressure.

## Pattern

```go
func parseJSON(jsonData []byte) map[string]interface{} {
    return safearena.Scoped(func(a *safearena.Arena) map[string]interface{} {
        // Allocate temporary AST nodes in arena
        node := safearena.Alloc(a, Node{...})
        buffer := safearena.AllocSlice[byte](a, 1024)

        // Process...

        // Return only the final result (heap-allocated)
        return result
    }) // All temporary allocations freed here
}
```

## Benefits

1. **Lower GC Pressure** - Temporary AST nodes don't burden the GC
2. **Predictable Performance** - Arena freed in one operation
3. **Clear Ownership** - Temporary vs persistent data is explicit

## Running

```bash
cd examples/json_parser
GOEXPERIMENT=arenas go run main.go
```

## Expected Output

```
JSON Parser Example: Arena-allocated AST

Iterations: 10000
With Arena:    45ms (4.5μs per operation)
Without Arena: 62ms (6.2μs per operation)
Speedup: 1.38x

Parsed data: map[...]
```

## When to Use

- Parsing large files with temporary data structures
- Multi-pass compilers with phase-specific data
- Any pipeline where intermediate results are discarded
- Processing streams where each chunk has temporary state

## Best Practices

1. **Arena for temporary, heap for results** - Only the final result leaves the scope
2. **Batch processing** - Process multiple items in one arena scope
3. **Size estimation** - Pre-allocate buffers at appropriate sizes
4. **Don't return arena pointers** - Use `Scoped()` return value for heap data only
