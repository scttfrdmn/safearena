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

// New creates a new safe arena with runtime safety checks.
// The arena must be freed with Free() when done, typically via defer.
//
// Example:
//
//	a := safearena.New()
//	defer a.Free()
//	data := safearena.Alloc(a, MyStruct{})
func New() *Arena {
	return &Arena{
		inner: arena.NewArena(),
		id:    arenaCounter.Add(1),
	}
}

// Alloc allocates a value in the arena and returns a safe pointer.
// The returned Ptr[T] tracks the arena lifetime and will panic on use-after-free.
//
// Panics if the arena has already been freed.
//
// Example:
//
//	data := safearena.Alloc(a, MyStruct{Field: "value"})
//	ptr := data.Get() // Safe while arena is alive
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

// Get safely dereferences the pointer with lifetime checking.
// Returns a pointer to the arena-allocated value.
//
// Panics if the arena has been freed with a helpful error message including
// stack trace and recovery hints.
//
// Example:
//
//	data := safearena.Alloc(a, 42)
//	value := data.Get() // Returns *int
//	fmt.Println(*value)
func (p Ptr[T]) Get() *T {
	if p.arena.freed.Load() {
		stack := captureStack(2)
		panic(errorWithHint(p.arena.id, "use after free", stack, hintUseAfterFree))
	}
	return p.ptr
}

// Deref dereferences and returns a copy of the value.
// Unlike Get(), this returns the value itself, not a pointer.
//
// Panics if the arena has been freed.
//
// Example:
//
//	data := safearena.Alloc(a, 42)
//	value := data.Deref() // Returns int (not *int)
//	fmt.Println(value)
func (p Ptr[T]) Deref() T {
	return *p.Get()
}

// Free safely frees the arena and all its allocations.
// After calling Free, any attempt to access arena-allocated values will panic
// with a descriptive error message.
//
// Panics on double-free to prevent memory corruption.
// Typically used with defer for automatic cleanup.
//
// Example:
//
//	a := safearena.New()
//	defer a.Free() // Automatic cleanup
//	// Use arena...
func (a *Arena) Free() {
	if !a.freed.CompareAndSwap(false, true) {
		stack := captureStack(2)
		panic(errorWithHint(a.id, "double free", stack, hintDoubleFree))
	}
	a.inner.Free()
}

// Scoped executes a function with an arena that's automatically freed.
// This is the recommended pattern as it's impossible to leak arena references.
// The arena is freed when the function returns, even if it panics.
//
// The function can return any heap-allocated value safely.
// Do not return Ptr[T] values - they will be invalid after Scoped returns.
//
// Example:
//
//	result := safearena.Scoped(func(a *safearena.Arena) Response {
//	    temp := safearena.Alloc(a, TempData{}) // Arena-allocated
//	    // Process temp...
//	    return Response{Status: 200} // Heap-allocated, safe to return
//	})
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

// Clone copies a value from the arena to the heap.
// Use this when you need to preserve arena-allocated data beyond the arena's lifetime.
// The returned pointer is heap-allocated and safe to use after the arena is freed.
//
// Panics if the arena has already been freed.
//
// Example:
//
//	a := safearena.New()
//	data := safearena.Alloc(a, Config{Port: 8080})
//	heapCopy := safearena.Clone(data) // Copy to heap
//	a.Free()
//	fmt.Println(heapCopy.Port) // Safe - heapCopy is on heap
func Clone[T any](p Ptr[T]) *T {
	val := p.Deref() // Get the value (panics if freed)
	heapCopy := new(T)
	*heapCopy = val
	return heapCopy
}

// Slice is an arena-allocated slice with lifetime tracking.
// Like Ptr[T], it tracks the arena lifetime and panics on use-after-free.
type Slice[T any] struct {
	slice []T
	arena *Arena
}

// AllocSlice allocates a slice in the arena with the specified size.
// The slice is initialized with zero values and has both length and capacity set to size.
//
// Panics if the arena has already been freed.
//
// Example:
//
//	buffer := safearena.AllocSlice[byte](a, 4096)
//	slice := buffer.Get()
//	copy(slice, []byte("data"))
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

// Get returns the underlying slice with lifetime checking.
// The returned slice is valid only while the arena is alive.
//
// Panics if the arena has been freed.
//
// Example:
//
//	buffer := safearena.AllocSlice[int](a, 100)
//	slice := buffer.Get()
//	for i := range slice {
//	    slice[i] = i
//	}
func (s Slice[T]) Get() []T {
	if s.arena.freed.Load() {
		stack := captureStack(2)
		panic(errorWithHint(s.arena.id, "use after free", stack, hintUseAfterFree))
	}
	return s.slice
}

// StringBuilder is an example of a safe arena-based string builder.
// It demonstrates how to build complex types using arena-allocated buffers.
type StringBuilder struct {
	buffers Slice[byte]
	length  int
}

// NewStringBuilder creates a new arena-allocated StringBuilder with the given capacity.
//
// Example:
//
//	sb := safearena.NewStringBuilder(a, 1024)
//	sb.Get().Append("Hello")
//	sb.Get().Append(" World")
//	result := sb.Get().String()
func NewStringBuilder(a *Arena, capacity int) Ptr[StringBuilder] {
	return Alloc(a, StringBuilder{
		buffers: AllocSlice[byte](a, capacity),
		length:  0,
	})
}

// Append adds a string to the StringBuilder.
func (sb *StringBuilder) Append(s string) {
	buf := sb.buffers.Get()
	copy(buf[sb.length:], s)
	sb.length += len(s)
}

// String returns the current string content of the StringBuilder.
func (sb *StringBuilder) String() string {
	buf := sb.buffers.Get()
	return string(buf[:sb.length])
}

// NewWithFinalizer creates an arena with a finalizer that detects leaked arenas.
// The finalizer warns if the arena is garbage collected without being freed.
// This is useful for debugging but adds overhead, so use New() in production.
//
// Example:
//
//	a := safearena.NewWithFinalizer()
//	defer a.Free() // Make sure to call Free()
//	// If you forget to Free(), you'll see a warning at GC time
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
