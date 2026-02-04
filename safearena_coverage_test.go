package safearena

import (
	"runtime"
	"sync"
	"testing"
)

// Test ScopedPtr function
func TestScopedPtr(t *testing.T) {
	executed := false
	ScopedPtr(func(a *Arena) {
		p := Alloc(a, 42)
		if *p.Get() != 42 {
			t.Error("expected 42")
		}
		executed = true
	})

	if !executed {
		t.Error("ScopedPtr function not executed")
	}
}

// Test NewWithFinalizer
func TestNewWithFinalizer(t *testing.T) {
	a := NewWithFinalizer()
	p := Alloc(a, "test")

	if *p.Get() != "test" {
		t.Error("expected test")
	}

	a.Free()

	// Verify panic after free
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on use-after-free")
		}
	}()
	_ = p.Get()
}

// Test AllocSlice with freed arena (error path)
func TestAllocSliceAfterFree(t *testing.T) {
	a := New()
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on allocation after free")
		}
	}()

	_ = AllocSlice[byte](a, 100)
}

// Test Alloc with freed arena (error path coverage)
func TestAllocAfterFree(t *testing.T) {
	a := New()
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on allocation after free")
		}
	}()

	_ = Alloc(a, 42)
}

// Test optimized version: NewOpt
func TestNewOpt(t *testing.T) {
	a := NewOpt()
	if a == nil {
		t.Error("NewOpt returned nil")
	}
	a.Free()
}

// Test optimized version: AllocOpt
func TestAllocOpt(t *testing.T) {
	a := NewOpt()
	defer a.Free()

	p := AllocOpt(a, 99)
	if *p.Get() != 99 {
		t.Error("expected 99")
	}

	// Test Deref
	val := p.Deref()
	if val != 99 {
		t.Error("Deref expected 99")
	}
}

// Test optimized version: ScopedOpt
func TestScopedOpt(t *testing.T) {
	result := ScopedOpt(func(a *ArenaOpt) string {
		p := AllocOpt(a, "optimized")
		return p.Deref()
	})

	if result != "optimized" {
		t.Error("expected optimized")
	}
}

// Test optimized version: CloneOpt
func TestCloneOpt(t *testing.T) {
	a := NewOpt()

	p := AllocOpt(a, "opt data")
	heapCopy := CloneOpt(p)

	a.Free()

	if *heapCopy != "opt data" {
		t.Error("expected opt data")
	}
}

// Test optimized version: AllocSliceOpt
func TestAllocSliceOpt(t *testing.T) {
	a := NewOpt()
	defer a.Free()

	s := AllocSliceOpt[int](a, 10)
	slice := s.Get()

	if len(slice) != 10 {
		t.Error("expected length 10")
	}

	slice[5] = 42
	if slice[5] != 42 {
		t.Error("expected 42 at index 5")
	}
}

// Test optimized version: UnsafeGet
func TestUnsafeGet(t *testing.T) {
	a := NewOpt()

	s := AllocSliceOpt[int](a, 5)

	// UnsafeGet should work even without checks
	slice := s.UnsafeGet()
	slice[0] = 100

	if slice[0] != 100 {
		t.Error("expected 100")
	}

	a.Free()

	// UnsafeGet still returns slice (unsafe!)
	slice = s.UnsafeGet()
	if slice == nil {
		t.Error("UnsafeGet returned nil")
	}
}

// Test optimized version: SetFinalizer
func TestSetFinalizer(t *testing.T) {
	a := NewOpt()
	a.SetFinalizer()
	a.Free()

	// Force GC to potentially trigger finalizer
	runtime.GC()
}

// Test optimized version: use after free
func TestOptUseAfterFree(t *testing.T) {
	a := NewOpt()
	p := AllocOpt(a, 42)
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on use-after-free")
		}
	}()

	_ = p.Get()
}

// Test optimized version: double free
func TestOptDoubleFree(t *testing.T) {
	a := NewOpt()
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on double free")
		}
	}()

	a.Free()
}

// Test optimized version: alloc after free
func TestOptAllocAfterFree(t *testing.T) {
	a := NewOpt()
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on alloc after free")
		}
	}()

	_ = AllocOpt(a, 42)
}

// Test optimized version: slice after free
func TestOptSliceAfterFree(t *testing.T) {
	a := NewOpt()
	s := AllocSliceOpt[byte](a, 10)
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on slice get after free")
		}
	}()

	_ = s.Get()
}

// Test optimized version: slice alloc after free
func TestOptSliceAllocAfterFree(t *testing.T) {
	a := NewOpt()
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on slice alloc after free")
		}
	}()

	_ = AllocSliceOpt[int](a, 10)
}

// Test large allocations
func TestLargeAllocations(t *testing.T) {
	result := Scoped(func(a *Arena) int {
		// Allocate many items
		ptrs := make([]Ptr[int], 1000)
		for i := 0; i < 1000; i++ {
			ptrs[i] = Alloc(a, i)
		}

		// Verify them all
		sum := 0
		for i := 0; i < 1000; i++ {
			sum += *ptrs[i].Get()
		}
		return sum
	})

	expected := (999 * 1000) / 2 // Sum of 0..999
	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

// Test large slices
func TestLargeSlice(t *testing.T) {
	Scoped(func(a *Arena) int {
		s := AllocSlice[byte](a, 1024*1024) // 1MB
		slice := s.Get()

		if len(slice) != 1024*1024 {
			t.Error("wrong slice size")
		}

		// Fill it
		for i := range slice {
			slice[i] = byte(i % 256)
		}

		// Verify
		if slice[100] != 100 {
			t.Error("wrong value")
		}

		return 0
	})
}

// Test concurrent arena usage (different arenas)
func TestConcurrentArenas(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			result := Scoped(func(a *Arena) int {
				p := Alloc(a, n*10)
				return p.Deref()
			})

			if result != n*10 {
				t.Errorf("expected %d, got %d", n*10, result)
			}
		}(i)
	}

	wg.Wait()
}

// Test complex struct allocation
func TestComplexStruct(t *testing.T) {
	type ComplexData struct {
		ID      int
		Name    string
		Values  []int
		Nested  map[string]interface{}
	}

	result := Scoped(func(a *Arena) string {
		data := Alloc(a, ComplexData{
			ID:     42,
			Name:   "test",
			Values: []int{1, 2, 3},
			Nested: map[string]interface{}{
				"key": "value",
			},
		})

		d := data.Get()
		return d.Name
	})

	if result != "test" {
		t.Error("expected test")
	}
}

// Test StringBuilder edge cases
func TestStringBuilder(t *testing.T) {
	result := Scoped(func(a *Arena) string {
		sb := NewStringBuilder(a, 100)

		// Multiple appends
		builder := sb.Get()
		builder.Append("Hello")
		builder.Append(" ")
		builder.Append("World")
		builder.Append("!")

		return builder.String()
	})

	if result != "Hello World!" {
		t.Errorf("expected 'Hello World!', got '%s'", result)
	}
}

// Test empty slice
func TestEmptySlice(t *testing.T) {
	Scoped(func(a *Arena) int {
		s := AllocSlice[int](a, 0)
		slice := s.Get()

		if len(slice) != 0 {
			t.Error("expected empty slice")
		}

		return 0
	})
}

// Test zero-value allocations
func TestZeroValues(t *testing.T) {
	Scoped(func(a *Arena) int {
		p := Alloc(a, 0)
		if *p.Get() != 0 {
			t.Error("expected 0")
		}

		s := Alloc(a, "")
		if *s.Get() != "" {
			t.Error("expected empty string")
		}

		return 0
	})
}

// Test multiple Free patterns
func TestMultipleArenas(t *testing.T) {
	a1 := New()
	a2 := New()
	a3 := New()

	p1 := Alloc(a1, 1)
	p2 := Alloc(a2, 2)
	p3 := Alloc(a3, 3)

	// Free in different order
	a2.Free()
	a1.Free()
	a3.Free()

	// All should be freed
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	_ = p1.Get()
	_ = p2.Get()
	_ = p3.Get()
}

// Test Clone with complex type
func TestCloneComplex(t *testing.T) {
	type Data struct {
		Values []int
		Text   string
	}

	a := New()

	original := Alloc(a, Data{
		Values: []int{1, 2, 3, 4, 5},
		Text:   "original",
	})

	cloned := Clone(original)

	a.Free()

	// Verify cloned data
	if cloned.Text != "original" {
		t.Error("wrong text")
	}
	if len(cloned.Values) != 5 {
		t.Error("wrong length")
	}
}

// Test Deref with structs
func TestDerefStruct(t *testing.T) {
	type Point struct {
		X, Y int
	}

	result := Scoped(func(a *Arena) Point {
		p := Alloc(a, Point{X: 10, Y: 20})
		return p.Deref() // Returns copy
	})

	if result.X != 10 || result.Y != 20 {
		t.Error("wrong point values")
	}
}
