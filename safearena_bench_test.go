package safearena

import (
	"testing"
)

// BenchmarkScopedAlloc: 100 arena allocations via Scoped
func BenchmarkScopedAlloc(b *testing.B) {
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

// Direct comparison: allocate single int
func BenchmarkSingleIntArena(b *testing.B) {
	for i := 0; i < b.N; i++ {
		a := New()
		p := Alloc(a, 42)
		_ = p.Deref()
		a.Free()
	}
}

func BenchmarkSingleIntHeap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := new(int)
		*p = 42
		_ = *p
	}
}

// BenchmarkPool100Allocs vs BenchmarkNew100Allocs shows pool amortization.
func BenchmarkPool100Allocs(b *testing.B) {
	var pool Pool
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := pool.Get()
		sum := 0
		for j := 0; j < 100; j++ {
			p := Alloc(a, j)
			sum += p.Deref()
		}
		_ = sum
		pool.Put(a)
	}
}

func BenchmarkNew100Allocs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		a := New()
		sum := 0
		for j := 0; j < 100; j++ {
			p := Alloc(a, j)
			sum += p.Deref()
		}
		_ = sum
		a.Free()
	}
}
