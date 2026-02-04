# HTTP Server Example

Demonstrates using SafeArena for request-scoped allocations in an HTTP server.

## Benefits

- **Reduced GC pressure**: Temporary allocations don't trigger GC
- **Predictable performance**: No GC pauses during request handling
- **Clean lifecycle**: Arena freed automatically after each request

## Run

```bash
GOEXPERIMENT=arenas go run main.go
```

Then test:
```bash
curl http://localhost:8080/hello
```

## Pattern

```go
func handler(w http.ResponseWriter, r *http.Request) {
    resp := safearena.Scoped(func(a *safearena.Arena) Response {
        // All temp allocations use arena
        buf := safearena.AllocSlice[byte](a, 4096)

        // Process request...

        // Return heap response
        return Response{...}
    }) // Arena freed here!

    // Write response
    w.WriteHeader(resp.StatusCode)
}
```

## Performance

Compared to regular GC:
- Lower latency (no GC pauses)
- Higher throughput (less GC overhead)
- More predictable response times
