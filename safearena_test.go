package safearena

import (
	"strings"
	"sync"
	"testing"
)

func TestBasicSafety(t *testing.T) {
	a := New()

	// Allocate some data
	p1 := Alloc(a, 42)
	p2 := Alloc(a, "hello")

	// Safe to use
	if *p1.Get() != 42 {
		t.Error("expected 42")
	}
	if *p2.Get() != "hello" {
		t.Error("expected hello")
	}

	// Free the arena
	a.Free()

	// This should panic!
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on use-after-free")
		}
	}()

	_ = p1.Get() // Should panic
}

func TestScoped(t *testing.T) {
	result := Scoped(func(a *Arena) int {
		p := Alloc(a, 100)
		return p.Deref() // Copy value out
	})
	// Arena automatically freed here

	if result != 100 {
		t.Error("expected 100")
	}
}

func TestDoubleFree(t *testing.T) {
	a := New()
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on double free")
		}
	}()

	a.Free() // Should panic
}

func TestClone(t *testing.T) {
	a := New()

	p := Alloc(a, "arena data")
	heapCopy := Clone(p) // Copy to heap

	a.Free()

	// heapCopy is still valid (on heap, not arena)
	if *heapCopy != "arena data" {
		t.Error("expected arena data")
	}
}

func TestSlice(t *testing.T) {
	a := New()

	s := AllocSlice[int](a, 5)
	slice := s.Get()
	slice[0] = 10
	slice[1] = 20

	if slice[0] != 10 || slice[1] != 20 {
		t.Error("slice not working")
	}

	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on use-after-free")
		}
	}()

	_ = s.Get() // Should panic
}

// Example: Processing requests safely
type OldRequest struct {
	ID   int
	Data string
}

type OldResponse struct {
	ID     int
	Result string
}

func processRequest(req OldRequest) OldResponse {
	return Scoped(func(a *Arena) OldResponse {
		// All temp allocations in arena
		_ = Alloc(a, make([]byte, 1024))
		builder := NewStringBuilder(a, 256)

		// Do processing
		sb := builder.Get()
		sb.Append("Processed: ")
		sb.Append(req.Data)

		// Return heap-allocated response
		// Arena automatically freed after this
		return OldResponse{
			ID:     req.ID,
			Result: sb.String(),
		}
	})
}

func TestRequestProcessing(t *testing.T) {
	req := OldRequest{ID: 1, Data: "test"}
	resp := processRequest(req)

	if resp.ID != 1 {
		t.Error("wrong ID")
	}
	if resp.Result != "Processed: test" {
		t.Error("wrong result")
	}
}

func TestReset(t *testing.T) {
	a := New()
	data := Alloc(a, 42)
	buf := AllocSlice[byte](a, 8)

	// Values accessible before reset
	if data.Deref() != 42 {
		t.Fatal("expected 42 before reset")
	}
	if len(buf.Get()) != 8 {
		t.Fatal("expected slice length 8 before reset")
	}

	a.Reset()

	// New allocations after reset work fine
	data2 := Alloc(a, 99)
	if data2.Deref() != 99 {
		t.Fatal("expected 99 after reset")
	}

	a.Free()

	// Pre-reset Ptr panics with "use after reset"
	t.Run("ptr panics after reset", func(t *testing.T) {
		a2 := New()
		old := Alloc(a2, 1)
		a2.Reset()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic")
			}
			if !strings.Contains(r.(string), "use after reset") {
				t.Fatalf("expected 'use after reset', got: %v", r)
			}
		}()
		_ = old.Get()
	})

	// Pre-reset Slice panics with "use after reset"
	t.Run("slice panics after reset", func(t *testing.T) {
		a3 := New()
		old := AllocSlice[int](a3, 4)
		a3.Reset()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic")
			}
			if !strings.Contains(r.(string), "use after reset") {
				t.Fatalf("expected 'use after reset', got: %v", r)
			}
		}()
		_ = old.Get()
	})
}

func TestResetAfterFree(t *testing.T) {
	a := New()
	a.Free()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on reset after free")
		}
		if !strings.Contains(r.(string), "reset after free") {
			t.Fatalf("expected 'reset after free', got: %v", r)
		}
	}()
	a.Reset()
}

func TestResetMultipleTimes(t *testing.T) {
	a := New()
	for i := 0; i < 5; i++ {
		p := Alloc(a, i)
		if p.Deref() != i {
			t.Fatalf("iteration %d: expected %d", i, i)
		}
		a.Reset()
	}
	a.Free()
}

func TestPool(t *testing.T) {
	var pool Pool

	a := pool.Get()
	p := Alloc(a, 42)
	if p.Deref() != 42 {
		t.Fatal("expected 42")
	}

	pool.Put(a) // resets a; p is now invalid

	t.Run("ptr panics after Put", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic after pool.Put")
			}
		}()
		_ = p.Get()
	})

	// Arena retrieved from pool works correctly
	a2 := pool.Get()
	p2 := Alloc(a2, 99)
	if p2.Deref() != 99 {
		t.Fatal("expected 99 from pooled arena")
	}
	pool.Put(a2)
}

func TestPoolConcurrent(t *testing.T) {
	var pool Pool
	const goroutines = 20
	const iters = 50

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				a := pool.Get()
				p := Alloc(a, n*1000+j)
				if p.Deref() != n*1000+j {
					t.Errorf("expected %d", n*1000+j)
				}
				pool.Put(a)
			}
		}(i)
	}
	wg.Wait()
}

func TestPoolPutFreedPanics(t *testing.T) {
	var pool Pool
	a := pool.Get()
	a.Free()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when Put-ing a freed arena")
		}
	}()
	pool.Put(a)
}

func BenchmarkSafeArena(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Scoped(func(a *Arena) int {
			sum := 0
			for j := 0; j < 100; j++ {
				p := Alloc(a, j)
				sum += p.Deref()
			}
			return sum
		})
	}
}

func BenchmarkRegularAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sum := 0
		for j := 0; j < 100; j++ {
			p := new(int)
			*p = j
			sum += *p
		}
	}
}

func TestArenaStats(t *testing.T) {
	// New() has no stats tracking.
	a := New()
	defer a.Free()
	Alloc(a, 42)
	if s := a.Stats(); s.AllocCount != 0 {
		t.Errorf("New() arena: expected AllocCount=0, got %d", s.AllocCount)
	}

	// NewWithStats() tracks allocs.
	b := NewWithStats()
	defer b.Free()
	if s := b.Stats(); s.AllocCount != 0 {
		t.Errorf("initial AllocCount: want 0, got %d", s.AllocCount)
	}
	Alloc(b, 1)
	Alloc(b, 2)
	AllocSlice[byte](b, 10)
	if s := b.Stats(); s.AllocCount != 3 {
		t.Errorf("AllocCount: want 3, got %d", s.AllocCount)
	}
}

func TestPoolStats(t *testing.T) {
	var p Pool

	// Initial state.
	if s := p.Stats(); s.Gets != 0 || s.Puts != 0 || s.Created != 0 || s.Reused != 0 {
		t.Errorf("initial stats non-zero: %+v", p.Stats())
	}

	// First Get: pool empty → new arena created.
	a := p.Get()
	if s := p.Stats(); s.Gets != 1 || s.Created != 1 || s.Reused != 0 {
		t.Errorf("after first Get: %+v", s)
	}

	// Put: arena returned.
	p.Put(a)
	if s := p.Stats(); s.Puts != 1 {
		t.Errorf("after Put: %+v", s)
	}

	// Second Get: pool has arena → reused.
	a2 := p.Get()
	defer p.Put(a2)
	if s := p.Stats(); s.Gets != 2 || s.Reused != 1 || s.Created != 1 {
		t.Errorf("after second Get: %+v", s)
	}
}
