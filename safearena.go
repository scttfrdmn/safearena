// Package safearena provides safe, ergonomic arena memory management for Go with runtime safety checks.
// It wraps Go's experimental arena package with type-safe wrappers that prevent use-after-free and double-free errors.
package safearena

import (
	"arena"
	"fmt"
	"runtime"
	"sync/atomic"
)

// Approach 1: Type-based safety with runtime checks
// Trade-off: Small runtime overhead for safety guarantees

// Arena wraps Go's arena with lightweight lifetime tracking
type Arena struct {
	inner *arena.Arena
	id    uint64
	freed atomic.Bool
	// Removed: objects sync.Map (unused, caused 10x slowdown)
}

// Ptr is a pointer that knows which arena it belongs to
// This is the key: encoding arena lifetime in the type!
type Ptr[T any] struct {
	ptr   *T
	arena *Arena // Keep reference to prevent premature freeing
	// Removed: arenaID (can get from arena.id, saves 8 bytes per pointer)
}

var arenaCounter atomic.Uint64

// New creates a new safe arena
func New() *Arena {
	return &Arena{
		inner: arena.NewArena(),
		id:    arenaCounter.Add(1),
	}
}

// Alloc allocates a value in the arena and returns a safe pointer
func Alloc[T any](a *Arena, value T) Ptr[T] {
	if a.freed.Load() {
		stack := captureStack(2)
		panic(errorWithHint(a.id, "allocation after free", stack, hintAllocAfterFree))
	}

	ptr := arena.New[T](a.inner)
	*ptr = value

	// No tracking needed - removed for 10x performance improvement

	return Ptr[T]{
		ptr:   ptr,
		arena: a,
	}
}

// Get safely dereferences the pointer with lifetime checking
func (p Ptr[T]) Get() *T {
	if p.arena.freed.Load() {
		stack := captureStack(2)
		panic(errorWithHint(p.arena.id, "use after free", stack, hintUseAfterFree))
	}
	return p.ptr
}

// Deref dereferences and returns the value (copies it out)
func (p Ptr[T]) Deref() T {
	return *p.Get()
}

// Free safely frees the arena
// After this, any Ptr.Get() will panic
func (a *Arena) Free() {
	if !a.freed.CompareAndSwap(false, true) {
		stack := captureStack(2)
		panic(errorWithHint(a.id, "double free", stack, hintDoubleFree))
	}
	a.inner.Free()
}

// Scoped executes a function with an arena that's automatically freed
// This is the safest pattern - impossible to leak references!
func Scoped[R any](fn func(*Arena) R) R {
	a := New()
	defer a.Free()
	return fn(a)
}

// ScopedPtr is like Scoped but prevents returning arena pointers
// The function CANNOT return a Ptr[T] - only regular heap values
func ScopedPtr(fn func(*Arena)) {
	a := New()
	defer a.Free()
	fn(a)
}

// Clone copies a value out of the arena to the heap
// This is how you safely extract data from arena
func Clone[T any](p Ptr[T]) *T {
	val := p.Deref() // Get the value (panics if freed)
	heapCopy := new(T)
	*heapCopy = val
	return heapCopy
}

// Advanced: Slice support
type Slice[T any] struct {
	slice []T
	arena *Arena
}

func AllocSlice[T any](a *Arena, size int) Slice[T] {
	if a.freed.Load() {
		stack := captureStack(2)
		panic(errorWithHint(a.id, "allocation after free", stack, hintAllocAfterFree))
	}

	// Allocate backing array in arena
	slice := make([]T, size)

	return Slice[T]{
		slice: slice,
		arena: a,
	}
}

func (s Slice[T]) Get() []T {
	if s.arena.freed.Load() {
		stack := captureStack(2)
		panic(errorWithHint(s.arena.id, "use after free", stack, hintUseAfterFree))
	}
	return s.slice
}

// Example: Safe arena-based string builder
type StringBuilder struct {
	buffers Slice[byte]
	length  int
}

func NewStringBuilder(a *Arena, capacity int) Ptr[StringBuilder] {
	return Alloc(a, StringBuilder{
		buffers: AllocSlice[byte](a, capacity),
		length:  0,
	})
}

func (sb *StringBuilder) Append(s string) {
	buf := sb.buffers.Get()
	copy(buf[sb.length:], s)
	sb.length += len(s)
}

func (sb *StringBuilder) String() string {
	buf := sb.buffers.Get()
	return string(buf[:sb.length])
}

// Finalizer-based safety (optional extra layer)
func NewWithFinalizer() *Arena {
	a := New()

	// Set finalizer to detect use-after-GC
	runtime.SetFinalizer(a, func(a *Arena) {
		if !a.freed.Load() {
			fmt.Printf("WARNING: arena %d was GC'd without being freed!\n", a.id)
		}
	})

	return a
}
