package safearena

import (
	"testing"
)

// Benchmark the optimized version
func BenchmarkOptimizedAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ScopedOpt(func(a *ArenaOpt) int {
			sum := 0
			for j := 0; j < 100; j++ {
				p := AllocOpt(a, j)
				sum += p.Deref()
			}
			return sum
		})
	}
}

// Benchmark original version for comparison
func BenchmarkOriginalAlloc(b *testing.B) {
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

// Realistic workload - optimized
func BenchmarkRealisticOptimized(b *testing.B) {
	type Request struct {
		ID      int
		Headers map[string]string
		Body    []byte
	}

	type Response struct {
		Status int
		Body   string
	}

	processRequest := func(req Request) Response {
		return ScopedOpt(func(a *ArenaOpt) Response {
			// Allocate lots of temporary data structures
			for i := 0; i < 100; i++ {
				temp := AllocOpt(a, struct {
					Buffers  [][]byte
					Metadata map[string]interface{}
					Scratch  [1024]byte
				}{
					Buffers:  make([][]byte, 10),
					Metadata: make(map[string]interface{}),
				})

				td := temp.Get()
				for j := 0; j < 10; j++ {
					td.Buffers[j] = make([]byte, 256)
				}
				td.Metadata["key"] = i
			}

			return Response{
				Status: 200,
				Body:   "processed",
			}
		})
	}

	req := Request{
		ID:      1,
		Headers: map[string]string{"User-Agent": "test"},
		Body:    []byte("test body"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processRequest(req)
	}
}

// Direct comparison: allocate single int
func BenchmarkSingleIntOptimized(b *testing.B) {
	for i := 0; i < b.N; i++ {
		a := NewOpt()
		p := AllocOpt(a, 42)
		_ = p.Deref()
		a.Free()
	}
}

func BenchmarkSingleIntOriginal(b *testing.B) {
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
