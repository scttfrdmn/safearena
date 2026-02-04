# Database Query Result Processor

This example demonstrates using SafeArena for processing database query results with request-scoped memory.

## Use Case

Database query processing often involves:
- Temporary buffers for data transformation
- String manipulation and normalization
- Filtering and aggregation state
- Working memory that's only needed for one query

All these temporary allocations can create GC pressure in high-throughput systems.

## Pattern

```go
func processQuery(rows []QueryResult) []ProcessedResult {
    return safearena.Scoped(func(a *safearena.Arena) []ProcessedResult {
        // Request-scoped allocations
        workBuffer := safearena.AllocSlice[byte](a, 4096)
        state := safearena.Alloc(a, ProcessingState{...})

        // Process rows with arena allocations
        for _, row := range rows {
            // Use arena buffers...
        }

        // Return only final results
        return results
    }) // All temporary state freed
}
```

## Benefits

1. **Request-Scoped Memory** - Each query gets its own arena
2. **Predictable Cleanup** - No lingering allocations between requests
3. **Better Throughput** - Less GC pressure in high-load scenarios
4. **Clear Lifecycle** - Temporary processing state is explicit

## Running

```bash
cd examples/database_processor
GOEXPERIMENT=arenas go run main.go
```

## Expected Output

```
Database Query Processor Example

Processing 1000 database rows...

Processed 1000 rows, found 200 matches

Benchmark Results (1000 iterations):
With Arena:    850ms (850μs per query)
Without Arena: 1.2s (1.2ms per query)
Speedup: 1.41x

Results: Found 200 matching rows
```

## Real-World Applications

1. **ORM/Query Builders** - Temporary structures during query construction
2. **Result Set Processing** - Filtering, mapping, aggregating results
3. **Connection Pooling** - Per-connection working buffers
4. **Cache Layer** - Temporary serialization/deserialization buffers
5. **Batch Processing** - Process N records with one arena scope

## Best Practices

1. **One arena per request/query** - Clear lifecycle boundaries
2. **Reuse patterns** - Similar queries can share allocation patterns
3. **Size appropriately** - Estimate buffer sizes based on query complexity
4. **Profile first** - Measure to ensure arena overhead is worth it
5. **Final results on heap** - Only processed results leave the scope

## Integration Example

```go
// HTTP handler with arena-scoped processing
func (h *Handler) QueryUsers(w http.ResponseWriter, r *http.Request) {
    rows, err := h.db.Query("SELECT * FROM users WHERE active = true")
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    defer rows.Close()

    // Process with arena
    results := processQueryWithArena(rows, r.URL.Query().Get("filter"))

    json.NewEncoder(w).Encode(results)
}
```

## Performance Tips

- **Batch size matters** - Larger batches amortize arena overhead
- **Avoid escaping** - Don't return arena pointers via `Scoped()`
- **Monitor GC** - Compare GC pause times with/without arenas
- **Measure end-to-end** - Arena benefits compound in full request lifecycle
