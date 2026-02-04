package safearena

import (
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
