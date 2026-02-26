# Arenas Without Fear: Safe Arena Memory Management in Go

Go's garbage collector is one of its best features. It is also, occasionally, one of its
biggest sources of latency. For most programs this is a fine trade — you write clean,
pointer-rich code and the GC quietly handles the rest. But for a narrow class of
workloads — request-per-core HTTP servers, real-time game loops, high-throughput
parsers — GC pause times start to show up in your p99s, and the instinct to reach for
manual memory management becomes hard to ignore.

Go 1.20 introduced an experimental answer: the `arena` package. This post explains what
arenas are, why the raw API is quietly dangerous, and how
[SafeArena](https://github.com/scttfrdmn/safearena) wraps it with runtime safety checks
and a static analyzer so you can get the performance benefits without the footguns.

---

## What Is an Arena Allocator?

An arena (sometimes called a *region* or *bump allocator*) is a memory management
strategy built on one observation: **most data has a structured lifetime**.

In a conventional HTTP handler, you allocate dozens of temporary objects — parsed
headers, intermediate buffers, decoded request bodies — use them for the duration of the
request, and then throw them all away. With a GC, each of those allocations is a
separate heap object. The collector has to trace them individually, update its write
barriers, and eventually reclaim them in a future cycle. With an arena, you allocate a
large slab of memory up front, carve objects out of it with a simple pointer bump, and
free the entire slab at the end of the request in a single operation. No tracing, no
write barriers, no per-object overhead.

```
  Regular heap                Arena
  ┌───┐ ┌───┐ ┌───┐         ┌───────────────────┐
  │ A │ │ B │ │ C │         │ A | B | C | D | … │
  └───┘ └───┘ └───┘         └───────────────────┘
    ↑     ↑     ↑                     ↑
  3 GC   roots to           1 free call, zero GC roots
   trace individually
```

The trade-off is that you give up per-object lifetime control. Everything in the arena
lives and dies together. This is a constraint — but for scoped, temporary data it is
exactly the constraint you want.

---

## Go's Experimental Arena Package

Go's `arena` package exposes this pattern directly:

```go
import "arena"

a := arena.NewArena()
defer a.Free()

ptr := arena.New[MyStruct](a)  // allocates inside the arena, not the heap
ptr.Field = "hello"
```

`arena.New[T]` returns a `*T` pointing into the arena's slab. When you call `a.Free()`,
the slab is reclaimed wholesale. The GC never sees the individual allocations — they
simply cease to exist when the arena does.

In practice this means fewer GC roots, smaller heap scans, and lower pause times. For
workloads with many short-lived allocations the improvement can be substantial: 40%+
lower GC pause times in our benchmarks.

The catch: the package is deliberately marked experimental, ships behind
`GOEXPERIMENT=arenas`, and carries no API stability guarantees. More pressingly for
day-to-day use, it provides **no safety guarantees whatsoever**.

---

## The Danger: Two Bugs That Are Hard to See

Raw arenas introduce two memory safety bugs that Go programmers rarely have to think
about.

### Use-after-free

```go
func getData() *MyStruct {
    a := arena.NewArena()
    defer a.Free()

    ptr := arena.New[MyStruct](a)
    ptr.Name = "hello"
    return ptr  // 💥 returning pointer into freed arena
}
```

The function looks reasonable. `defer a.Free()` is idiomatic Go. But the returned
pointer points into memory that was freed when `a.Free()` ran. The caller receives a
dangling pointer. In C this is undefined behaviour; in Go it typically causes silent
data corruption or a delayed crash somewhere entirely unrelated to the bug.

There is no compiler error. There is no runtime check. The code passes `go vet`. You
will discover this bug in production.

### Double-free

```go
func process(a *arena.Arena) {
    defer a.Free()  // Free #1
    // ...
}

func main() {
    a := arena.NewArena()
    defer a.Free()  // Free #2 — will run after process()
    process(a)
}
```

Two callers both call `Free()` on the same arena. Depending on timing and allocator
internals, this can corrupt allocator metadata and cause crashes in completely unrelated
code.

Both bugs are invisible at the call site. They require careful whole-program reasoning
to catch — the kind of reasoning that is easy to get wrong under deadline pressure, and
that code review rarely catches because each individual `defer a.Free()` looks correct
in isolation.

---

## The Solution: Encoding Lifetime in the Type

SafeArena's core idea is to make the arena's lifetime *visible to the type system*. Instead
of returning a raw `*T`, `Alloc` returns a `Ptr[T]`:

```go
type Ptr[T any] struct {
    ptr        *T
    arena      *Arena
    generation uint64
}
```

`Ptr[T]` is a small struct that carries three things: the actual pointer, a reference to
the arena it came from, and a *generation counter* (more on that in a moment). Accessing
the value requires going through `Get()`:

```go
func (p Ptr[T]) Get() *T {
    if p.arena.freed.Load() {
        panic(errorWithHint(..., "use after free", ...))
    }
    if p.arena.generation.Load() != p.generation {
        panic(errorWithHint(..., "use after reset", ...))
    }
    return p.ptr
}
```

Every `Get()` call performs two atomic loads — `freed` and `generation` — before
returning the pointer. If the arena has been freed, or if the arena has been reset since
this pointer was allocated, you get an immediate panic with a descriptive message and a
stack trace, not silent corruption:

```
arena 5: use after free
  at handler.go:42 (myapp.handleRequest)

  [hint] Arena was freed before this access. Use Clone() to copy values
  to heap, or ensure arena lifetime covers all uses.
```

This is the same trade-off Rust makes with its borrow checker — fail loudly and early
rather than silently and late — but enforced at runtime rather than compile time.

### The Generation Counter

Why is there a generation counter at all? `Free()` is easy to check with a boolean flag.
But SafeArena also provides `Reset()`, which frees all allocations and immediately
prepares the arena for reuse — without invalidating the `*Arena` pointer itself. This is
useful for `Pool`-based patterns where arenas are recycled:

```go
a := pool.Get()
defer pool.Put(a)  // Put calls Reset internally

p := safearena.Alloc(a, MyStruct{})
// ... use p ...
// pool.Put resets the arena; p.Get() after this point should panic
```

A simple `freed bool` would not catch this case — the arena is not freed, it is reused.
The generation counter is incremented on every `Reset()`. A `Ptr[T]` stamped with
generation 3 will panic when accessed after the arena has been reset to generation 4, even
though `freed` is still false.

---

## The Recommended Pattern: `Scoped`

The safest way to use SafeArena is `Scoped`, which makes arena leaks structurally
impossible:

```go
func handleRequest(req Request) Response {
    return safearena.Scoped(func(a *safearena.Arena) Response {
        // Allocate temporary data in the arena
        buffer := safearena.AllocSlice[byte](a, 4096)
        temp   := safearena.Alloc(a, ParseState{})

        // Use them safely
        parse(buffer.Get(), temp.Get(), req.Body)

        // Return a heap-allocated value — safe to use after the arena is freed
        return Response{Status: 200, Body: temp.Get().Result}
    })
    // Arena is freed here, even if the function panics
}
```

`Scoped` creates an arena, calls your function with it, and frees it via `defer` before
returning. The function's return type is `R` — a regular heap-allocated Go value. Because
the arena is freed inside `Scoped`, any `Ptr[T]` you might accidentally return would
immediately be caught on first access. You cannot return a `Ptr[T]` to the caller through
the type system alone, but the combination of the pattern and the runtime checks makes the
failure mode immediate and obvious rather than delayed and mysterious.

For fire-and-forget scopes with no return value, `ScopedVoid` is available:

```go
safearena.ScopedVoid(func(a *safearena.Arena) {
    buf := safearena.AllocSlice[byte](a, 1<<20) // 1 MB scratch buffer
    render(buf.Get())
    writeToSocket(buf.Get())
    // arena freed, buffer reclaimed
})
```

---

## Arena Pools for High-Throughput Workloads

Creating a new arena for every request involves a system call to allocate the slab.
For very high request rates, amortising this cost matters. `Pool` wraps `sync.Pool` to
reuse arenas across requests:

```go
var pool safearena.Pool

func handleRequest(req Request) Response {
    a := pool.Get()
    defer pool.Put(a)   // Reset + return to pool

    buffer := safearena.AllocSlice[byte](a, 4096)
    // ...
    return Response{...}
}
```

`Pool.Put` calls `Reset()` before returning the arena, incrementing its generation
counter and invalidating all outstanding `Ptr[T]` values — so any lingering references
from the previous use will panic on access. `Pool` also tracks `Gets`, `Puts`,
`Created`, and `Reused` statistics via atomic counters, useful for observability.

---

## Catching Bugs Before Runtime: `arenacheck`

Runtime checks are valuable, but catching bugs at build time is better. SafeArena ships
`arenacheck`, an SSA-based static analyzer that integrates with `go vet`:

```bash
GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...
```

It detects patterns where arena-allocated values escape their intended scope:

```go
// arenacheck: safearena.Ptr returned from function — use after free
func bad() *int {
    return safearena.Scoped(func(a *safearena.Arena) *int {
        p := safearena.Alloc(a, 42)
        return p.Get()  // ← flagged: raw *int escapes arena scope
    })
}

// arenacheck: safearena.Ptr captured by goroutine launch
func alsobad(a *safearena.Arena) {
    p := safearena.Alloc(a, 42)
    go func() { fmt.Println(p.Get()) }()  // ← flagged: goroutine may outlive arena
}
```

Detection covers direct returns, global variable stores, `interface{}` wrapping, closure
captures, and goroutine launches — roughly 100% of the patterns that appear in practice.
The known gaps are escapes through struct fields, maps/channels, and interprocedural
calls across package boundaries (requiring whole-program analysis).

---

## Honest Performance Numbers

SafeArena adds exactly two atomic loads per `Get()` call. On modern hardware an atomic
load from a cache-hot location costs roughly 1–2 ns. That is the entire runtime overhead
for a safely-dereferenced pointer.

For allocation, `Alloc` pays one atomic load (freed check) plus the `arena.New[T]` call
itself. End-to-end:

| Operation | Raw Arena | SafeArena | Overhead |
|-----------|-----------|-----------|---------|
| Alloc     | 150 ns    | 165 ns    | +10%    |
| Get       | 0 ns      | 2 ns      | +2 ns   |
| Free      | 50 ns     | 52 ns     | +4%     |

In a realistic HTTP request benchmark (100 allocations per request, compared against
regular heap allocation):

```
BenchmarkSafeArena    104.8 μs/op    0.047 ms GC pause
BenchmarkRegularGC     92.5 μs/op    0.082 ms GC pause
```

SafeArena is ~13% slower per operation, but GC pause times drop by 42%. Whether that
trade-off is worthwhile depends on your workload — but for GC-sensitive services it often
is.

One important caveat: `AllocSlice[T]` uses `make([]T, size)` for the backing array. This
is a limitation of Go's arena API — there is no arena-backed slice primitive. The
lifetime safety guarantee still holds (the `Slice[T]` wrapper panics on use-after-free),
but the memory itself lives on the regular heap. Profile before assuming `AllocSlice` is
reducing GC pressure.

---

## When to Reach for Arenas

Arenas are not a universal performance improvement. They are the right tool when:

- **Lifetime is clear and short**: request-scoped, frame-scoped, batch-scoped
- **Allocation rate is high**: many objects per arena, not one
- **GC pressure is the measured bottleneck**: profile first

Arenas are the wrong tool when:

- Data lifetime is unknown or long-lived
- You are already meeting your latency targets
- The added complexity is not worth the benefit

The `Scoped` pattern makes the first condition easy to enforce: if you cannot express
your data's lifetime as a function argument and return value, you are probably not in
arena territory.

---

## Getting Started

```bash
go get github.com/scttfrdmn/safearena
go install github.com/scttfrdmn/safearena/cmd/arenacheck@latest
```

All commands require `GOEXPERIMENT=arenas`.

The repository includes a [Migration Guide](MIGRATION.md) for incrementally introducing
arenas into existing code, a [Performance Guide](PERFORMANCE.md) with profiling
templates, and a [FAQ](FAQ.md) covering common questions and gotchas.

---

## Conclusion

Arena allocators have a long history in systems programming — they appear in compilers,
game engines, databases, and web servers wherever predictable latency matters. Go's
experimental arena package finally makes them accessible to Go programmers, but the raw
API asks you to reason carefully about memory lifetimes in a language that has
deliberately insulated you from that reasoning for years.

SafeArena does not eliminate that reasoning — arenas are fundamentally a manual memory
management technique. What it does is make the failure modes immediate, debuggable, and
catchable before production. Use-after-free becomes a panic with a stack trace. A
forgotten `Free` becomes a GC-time warning. A dangling pointer escaped into a goroutine
becomes a build-time error.

The goal is not to match Rust's compile-time guarantees — it is to bring arena
allocators into reach for Go programmers who want better performance without giving up
the safety that makes Go pleasant to write.

---

*SafeArena is at [github.com/scttfrdmn/safearena](https://github.com/scttfrdmn/safearena).
It requires Go 1.23+ with `GOEXPERIMENT=arenas`. The arena package remains experimental —
evaluate carefully before using in production systems.*
