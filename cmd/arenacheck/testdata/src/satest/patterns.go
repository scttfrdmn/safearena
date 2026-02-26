// Package satest contains test cases for arenacheck's safearena-specific analysis.
// Bad patterns (marked with want comments) should be flagged.
// Good patterns (no want comments) should produce no diagnostic.
package satest

import "github.com/scttfrdmn/safearena"

// --- BAD patterns: should be flagged ---

// Returning Ptr[T] leaks the arena wrapper beyond the function's lifetime.
func badReturnPtr() safearena.Ptr[int] {
	a := safearena.New()
	defer a.Free()
	return safearena.Alloc(a, 42) // want "safearena.Ptr escapes via return"
}

// Returning Slice[T] leaks the arena wrapper beyond the function's lifetime.
func badReturnSlice() safearena.Slice[byte] {
	a := safearena.New()
	defer a.Free()
	return safearena.AllocSlice[byte](a, 100) // want "safearena.Slice escapes via return"
}

// Returning the raw *T from .Get() — arena may be freed when caller uses it.
func badReturnGet() *int {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	return p.Get() // want `\.Get\(\) escapes via return`
}

// Returning the raw []T from Slice.Get().
func badReturnSliceGet() []byte {
	a := safearena.New()
	defer a.Free()
	s := safearena.AllocSlice[byte](a, 64)
	return s.Get() // want `\.Get\(\) escapes via return`
}

// Storing Ptr[T] to a global outlives the arena.
var globalPtr safearena.Ptr[int]

func badGlobalPtr() {
	a := safearena.New()
	defer a.Free()
	globalPtr = safearena.Alloc(a, 42) // want "safearena.Ptr escapes via global variable"
}

// Storing Slice[T] to a global.
var globalSlice safearena.Slice[byte]

func badGlobalSlice() {
	a := safearena.New()
	defer a.Free()
	globalSlice = safearena.AllocSlice[byte](a, 64) // want "safearena.Slice escapes via global variable"
}

// Storing raw *T from .Get() to a global.
var globalRaw *int

func badGlobalGet() {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	globalRaw = p.Get() // want `\.Get\(\) escapes via global variable`
}

// Wrapping Ptr[T] in interface{} and returning it still escapes the arena wrapper.
func badInterfaceEscape() interface{} {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	return interface{}(p) // want "safearena.Ptr escapes via return"
}

// Returning a closure that captures Ptr[T] — callable after arena is freed.
func badClosureCapture() func() int {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	return func() int { // want `safearena\.Ptr captured by closure return`
		return p.Deref()
	}
}

// Goroutine that captures Ptr[T] — races with a.Free().
func badGoroutineCapture() {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	go func() { // want `safearena\.Ptr captured by goroutine launch`
		_ = p.Deref()
	}()
}

// Goroutine that captures the raw *T from .Get() — same hazard.
func badGoroutineCaptureGet() {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	raw := p.Get()
	go func() { // want `\.Get\(\) captured by goroutine launch`
		_ = *raw
	}()
}

// --- BAD patterns: struct field escapes ---

// structHolder is used to test field-store escape detection.
type structHolder struct {
	ptr safearena.Ptr[int]
	sl  safearena.Slice[byte]
	raw *int
}

// Storing Ptr[T] into a field of a heap-allocated struct that escapes.
func badFieldStoreHeapPtr(a *safearena.Arena) *structHolder {
	s := &structHolder{}
	s.ptr = safearena.Alloc(a, 42) // want `safearena\.Ptr escapes via struct field`
	return s
}

// Storing Slice[T] into a field of a heap-allocated struct that escapes.
func badFieldStoreHeapSlice(a *safearena.Arena) *structHolder {
	s := &structHolder{}
	s.sl = safearena.AllocSlice[byte](a, 64) // want `safearena\.Slice escapes via struct field`
	return s
}

// Storing Ptr[T] into a field of a struct passed as a parameter — caller owns the struct.
func badFieldStoreParam(s *structHolder, a *safearena.Arena) {
	s.ptr = safearena.Alloc(a, 99) // want `safearena\.Ptr escapes via struct field`
}

// Storing raw *T from .Get() into a field of a heap-allocated struct.
func badFieldStoreGetHeap(a *safearena.Arena) *structHolder {
	s := &structHolder{}
	p := safearena.Alloc(a, 42)
	s.raw = p.Get() // want `\.Get\(\) escapes via struct field`
	return s
}

// --- GOOD patterns: should NOT be flagged ---

// Deref() returns a value copy — safe.
func goodDeref() int {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	return p.Deref()
}

// Clone() copies to the heap — safe.
func goodClone() *int {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	return safearena.Clone(p)
}

// Using Get() only within the function scope — safe.
func goodLocalGet() int {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	val := p.Get()
	return *val // dereferences the pointer; returns the int value, not the pointer
}

// Using Scoped with a safe return type — safe.
func goodScoped() int {
	return safearena.Scoped(func(a *safearena.Arena) int {
		p := safearena.Alloc(a, 99)
		return p.Deref()
	})
}

// Using Scoped with Clone — safe.
func goodScopedClone() *int {
	return safearena.Scoped(func(a *safearena.Arena) *int {
		p := safearena.Alloc(a, 7)
		return safearena.Clone(p)
	})
}

// Closure called locally before defer fires — not an escape, safe.
func goodLocalClosure() int {
	a := safearena.New()
	defer a.Free()
	p := safearena.Alloc(a, 42)
	f := func() int { return p.Deref() } // captured but closure doesn't escape
	return f()
}

// Storing Ptr[T] into a field of a purely stack-local struct — safe.
// The struct (var s structHolder) is Alloc.Heap=false; it doesn't outlive the function.
func goodFieldStoreLocalStruct(a *safearena.Arena) int {
	var s structHolder
	s.ptr = safearena.Alloc(a, 42)
	return s.ptr.Deref()
}

// Reading slice contents locally — safe.
func goodSliceLocal() int {
	a := safearena.New()
	defer a.Free()
	s := safearena.AllocSlice[int](a, 3)
	slice := s.Get()
	total := 0
	for _, v := range slice {
		total += v
	}
	return total
}
