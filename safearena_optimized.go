package safearena

// Optimized version - remove unused tracking, optimize hot paths

import (
	"arena"
	"fmt"
	"runtime"
	"sync/atomic"
)

// Arena wraps Go's arena with lightweight lifetime tracking
type ArenaOpt struct {
	inner *arena.Arena
	id    uint64
	freed atomic.Bool
	// Removed: objects sync.Map (never used!)
}

// PtrOpt is an optimized pointer with minimal overhead
type PtrOpt[T any] struct {
	ptr   *T
	arena *ArenaOpt
	// Removed: arenaID (can get from arena.id if needed)
}

var arenaCounterOpt atomic.Uint64

// NewOpt creates a new optimized arena
func NewOpt() *ArenaOpt {
	return &ArenaOpt{
		inner: arena.NewArena(),
		id:    arenaCounterOpt.Add(1),
	}
}

// AllocOpt allocates a value with minimal overhead
func AllocOpt[T any](a *ArenaOpt, value T) PtrOpt[T] {
	// Single atomic load - inlined by compiler
	if a.freed.Load() {
		panic(fmt.Sprintf("arena %d: allocation after free", a.id))
	}

	ptr := arena.New[T](a.inner)
	*ptr = value

	// No tracking needed!

	return PtrOpt[T]{
		ptr:   ptr,
		arena: a,
	}
}

// Get safely dereferences with minimal overhead
func (p PtrOpt[T]) Get() *T {
	// Fast path: single atomic load
	if p.arena.freed.Load() {
		panic(fmt.Sprintf("arena %d: use after free", p.arena.id))
	}
	return p.ptr
}

// Deref returns the value (copies it out)
func (p PtrOpt[T]) Deref() T {
	return *p.Get()
}

// Free safely frees the arena
func (a *ArenaOpt) Free() {
	if !a.freed.CompareAndSwap(false, true) {
		panic(fmt.Sprintf("arena %d: double free", a.id))
	}
	a.inner.Free()
}

// ScopedOpt executes a function with an arena that's automatically freed
func ScopedOpt[R any](fn func(*ArenaOpt) R) R {
	a := NewOpt()
	defer a.Free()
	return fn(a)
}

// CloneOpt copies a value out of the arena to the heap
func CloneOpt[T any](p PtrOpt[T]) *T {
	val := p.Deref()
	heapCopy := new(T)
	*heapCopy = val
	return heapCopy
}

// SliceOpt is an optimized arena slice
type SliceOpt[T any] struct {
	slice []T
	arena *ArenaOpt
}

// AllocSliceOpt allocates a slice in the arena
func AllocSliceOpt[T any](a *ArenaOpt, size int) SliceOpt[T] {
	if a.freed.Load() {
		panic(fmt.Sprintf("arena %d: allocation after free", a.id))
	}

	slice := make([]T, size)

	return SliceOpt[T]{
		slice: slice,
		arena: a,
	}
}

// Get returns the slice with safety check
func (s SliceOpt[T]) Get() []T {
	if s.arena.freed.Load() {
		panic(fmt.Sprintf("arena %d: use after free", s.arena.id))
	}
	return s.slice
}

// UnsafeGet returns the slice without checking (use carefully!)
func (s SliceOpt[T]) UnsafeGet() []T {
	return s.slice
}

// SetFinalizer adds a finalizer to detect leaked arenas (optional debug mode)
func (a *ArenaOpt) SetFinalizer() {
	runtime.SetFinalizer(a, func(a *ArenaOpt) {
		if !a.freed.Load() {
			fmt.Printf("WARNING: arena %d was GC'd without being freed!\n", a.id)
		}
	})
}
