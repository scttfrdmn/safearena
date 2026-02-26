// Package safearena provides safe arena memory management (stub for arenacheck tests).
// This is a minimal stub that mirrors the public API so that analysistest can load
// user code that imports safearena without requiring GOEXPERIMENT=arenas.
package safearena

// Arena wraps an arena with lifetime tracking.
type Arena struct{}

// New creates a new arena.
func New() *Arena { return nil }

// Free frees the arena.
func (a *Arena) Free() {}

// Reset frees all allocations and prepares the arena for reuse.
func (a *Arena) Reset() {}

// Ptr is a pointer that knows which arena it belongs to.
// Returning a Ptr[T] from a function is unsafe — use Deref() or Clone() instead.
// Note: must be non-zero size so that the Go compiler generates closure bindings for
// captured Ptr[T] values (zero-size types may be elided by the capture analysis).
type Ptr[T any] struct{ arena *Arena }

// Get safely dereferences the pointer.
// The returned raw pointer is only valid while the arena is alive.
func (p Ptr[T]) Get() *T { return nil }

// Deref returns a copy of the value — safe to return from any function.
func (p Ptr[T]) Deref() T { var zero T; return zero }

// Slice is an arena-allocated slice with lifetime tracking.
// Returning a Slice[T] from a function is unsafe — copy the contents instead.
// Note: must be non-zero size for the same reason as Ptr[T].
type Slice[T any] struct{ arena *Arena }

// Get returns the underlying slice.
// The returned slice is only valid while the arena is alive.
func (s Slice[T]) Get() []T { return nil }

// Alloc allocates a value in the arena and returns a safe Ptr[T] wrapper.
func Alloc[T any](a *Arena, value T) Ptr[T] { return Ptr[T]{} }

// AllocSlice creates a lifetime-tracked slice of the given size.
func AllocSlice[T any](a *Arena, size int) Slice[T] { return Slice[T]{} }

// Clone copies an arena value to the heap — safe to return from any function.
func Clone[T any](p Ptr[T]) *T { return nil }

// Scoped executes fn with an automatically freed arena.
func Scoped[R any](fn func(*Arena) R) R { var zero R; return zero }

// ScopedVoid executes fn with an automatically freed arena (no return value).
func ScopedVoid(fn func(*Arena)) {}
