package safearena

import (
	"runtime"
	"testing"
)

// Realistic data structure
type Request struct {
	ID      int
	Headers map[string]string
	Body    []byte
}

type TempData struct {
	Buffers  [][]byte
	Metadata map[string]interface{}
	Scratch  [1024]byte
}

type Response struct {
	Status int
	Body   string
}

// Realistic workload: process a request with lots of temp allocations
func processWithSafeArena(req Request) Response {
	return Scoped(func(a *Arena) Response {
		// Allocate lots of temporary data structures
		for i := 0; i < 100; i++ {
			temp := Alloc(a, TempData{
				Buffers:  make([][]byte, 10),
				Metadata: make(map[string]interface{}),
			})

			td := temp.Get()
			for j := 0; j < 10; j++ {
				td.Buffers[j] = make([]byte, 256)
			}
			td.Metadata["key"] = i
		}

		// Return heap-allocated response
		return Response{
			Status: 200,
			Body:   "processed",
		}
	})
}

func processWithRegularGC(req Request) Response {
	// Same logic but everything on heap (will be GC'd)
	for i := 0; i < 100; i++ {
		temp := &TempData{
			Buffers:  make([][]byte, 10),
			Metadata: make(map[string]interface{}),
		}

		for j := 0; j < 10; j++ {
			temp.Buffers[j] = make([]byte, 256)
		}
		temp.Metadata["key"] = i
	}

	return Response{
		Status: 200,
		Body:   "processed",
	}
}

func BenchmarkRealisticSafeArena(b *testing.B) {
	req := Request{
		ID:      1,
		Headers: map[string]string{"User-Agent": "test"},
		Body:    []byte("test body"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processWithSafeArena(req)
	}
}

func BenchmarkRealisticRegularGC(b *testing.B) {
	req := Request{
		ID:      1,
		Headers: map[string]string{"User-Agent": "test"},
		Body:    []byte("test body"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processWithRegularGC(req)
	}
}

// Show GC impact
func BenchmarkRealisticWithGCStats(b *testing.B) {
	req := Request{
		ID:      1,
		Headers: map[string]string{"User-Agent": "test"},
		Body:    []byte("test body"),
	}

	b.Run("SafeArena", func(b *testing.B) {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)

		for i := 0; i < b.N; i++ {
			_ = processWithSafeArena(req)
		}

		runtime.ReadMemStats(&after)
		b.ReportMetric(float64(after.NumGC-before.NumGC), "gc-count")
		b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1000000, "gc-pause-ms")
	})

	b.Run("RegularGC", func(b *testing.B) {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)

		for i := 0; i < b.N; i++ {
			_ = processWithRegularGC(req)
		}

		runtime.ReadMemStats(&after)
		b.ReportMetric(float64(after.NumGC-before.NumGC), "gc-count")
		b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1000000, "gc-pause-ms")
	})
}
