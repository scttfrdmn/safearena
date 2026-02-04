package safearena_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/scttfrdmn/safearena"
)

// Integration test: Concurrent request processing
func TestIntegrationConcurrentRequests(t *testing.T) {
	const numRequests = 100
	const numWorkers = 10

	type Request struct {
		ID   int
		Data []byte
	}

	type Response struct {
		ID     int
		Result string
	}

	processRequest := func(req Request) Response {
		return safearena.Scoped(func(a *safearena.Arena) Response {
			// Simulate request-scoped processing
			buffer := safearena.AllocSlice[byte](a, 1024)
			temp := safearena.Alloc(a, struct {
				count int
				data  []byte
			}{})

			buf := buffer.Get()
			copy(buf, req.Data)

			t := temp.Get()
			t.count = len(req.Data)
			t.data = buf

			return Response{
				ID:     req.ID,
				Result: fmt.Sprintf("processed %d bytes", t.count),
			}
		})
	}

	var wg sync.WaitGroup
	requests := make(chan Request, numRequests)
	responses := make(chan Response, numRequests)

	// Workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range requests {
				resp := processRequest(req)
				responses <- resp
			}
		}()
	}

	// Generate requests
	go func() {
		for i := 0; i < numRequests; i++ {
			requests <- Request{
				ID:   i,
				Data: []byte(fmt.Sprintf("request-%d", i)),
			}
		}
		close(requests)
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(responses)
	}()

	// Collect responses
	count := 0
	for range responses {
		count++
	}

	if count != numRequests {
		t.Errorf("expected %d responses, got %d", numRequests, count)
	}
}

// Integration test: Nested arena scopes
func TestIntegrationNestedScopes(t *testing.T) {
	result := safearena.Scoped(func(a1 *safearena.Arena) int {
		outer := safearena.Alloc(a1, 10)

		innerResult := safearena.Scoped(func(a2 *safearena.Arena) int {
			inner := safearena.Alloc(a2, 20)
			return *outer.Get() + *inner.Get()
		})

		return innerResult + *outer.Get()
	})

	expected := 10 + 20 + 10
	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

// Integration test: Long-running arena with many allocations
func TestIntegrationLongRunningArena(t *testing.T) {
	const numAllocations = 10000

	result := safearena.Scoped(func(a *safearena.Arena) int {
		ptrs := make([]safearena.Ptr[int], numAllocations)

		// Allocate many objects
		for i := 0; i < numAllocations; i++ {
			ptrs[i] = safearena.Alloc(a, i)
		}

		// Access them all
		sum := 0
		for _, ptr := range ptrs {
			sum += *ptr.Get()
		}

		return sum
	})

	expected := (numAllocations - 1) * numAllocations / 2
	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

// Integration test: Mixed arena and heap allocations
func TestIntegrationMixedAllocations(t *testing.T) {
	heapData := make([]*int, 100)

	safearena.Scoped(func(a *safearena.Arena) int {
		// Mix arena and heap allocations
		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				// Arena allocation
				arenaPtr := safearena.Alloc(a, i)
				// Clone to heap
				heapData[i] = safearena.Clone(arenaPtr)
			} else {
				// Direct heap allocation
				heapData[i] = new(int)
				*heapData[i] = i
			}
		}
		return 0
	})

	// Verify heap data survived
	for i := 0; i < 100; i++ {
		if *heapData[i] != i {
			t.Errorf("index %d: expected %d, got %d", i, i, *heapData[i])
		}
	}
}

// Integration test: Complex data structures
func TestIntegrationComplexStructures(t *testing.T) {
	type Node struct {
		Value int
		Next  *Node
	}

	result := safearena.Scoped(func(a *safearena.Arena) int {
		// Build linked list in arena
		var head *Node
		for i := 0; i < 10; i++ {
			node := safearena.Alloc(a, Node{
				Value: i,
				Next:  head,
			})
			head = node.Get()
		}

		// Traverse and sum
		sum := 0
		for current := head; current != nil; current = current.Next {
			sum += current.Value
		}

		return sum
	})

	expected := 45 // 0+1+2+...+9
	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

// Integration test: Error recovery
func TestIntegrationErrorRecovery(t *testing.T) {
	recovered := false

	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()

		a := safearena.New()
		data := safearena.Alloc(a, 42)
		a.Free()

		// This should panic
		_ = data.Get()
	}()

	if !recovered {
		t.Error("expected panic on use-after-free")
	}
}

// Integration test: Stress test with rapid allocation/free
func TestIntegrationStressTest(t *testing.T) {
	const iterations = 1000

	for i := 0; i < iterations; i++ {
		safearena.Scoped(func(a *safearena.Arena) int {
			// Rapid allocations
			for j := 0; j < 100; j++ {
				_ = safearena.Alloc(a, j)
			}
			return 0
		})
	}

	// Should complete without crashes
}

// Integration test: Memory pressure simulation
func TestIntegrationMemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory pressure test in short mode")
	}

	const numIterations = 100
	const largeSizeBytes = 1024 * 1024 // 1MB

	for i := 0; i < numIterations; i++ {
		safearena.Scoped(func(a *safearena.Arena) int {
			// Allocate large buffers
			buffer := safearena.AllocSlice[byte](a, largeSizeBytes)
			slice := buffer.Get()

			// Use the buffer
			for j := 0; j < len(slice); j += 1024 {
				slice[j] = byte(j % 256)
			}

			return len(slice)
		})
	}

	// Should handle memory pressure without issues
}

// Integration test: Optimized version compatibility
func TestIntegrationOptimizedVersion(t *testing.T) {
	// Test that optimized version works correctly
	result := safearena.ScopedOpt(func(a *safearena.ArenaOpt) int {
		data := safearena.AllocOpt(a, 42)
		return data.Deref()
	})

	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

// Integration test: Real-world HTTP-like scenario
func TestIntegrationHTTPScenario(t *testing.T) {
	type HTTPRequest struct {
		Method  string
		Path    string
		Headers map[string]string
		Body    []byte
	}

	type HTTPResponse struct {
		StatusCode int
		Body       string
	}

	handleRequest := func(req HTTPRequest) HTTPResponse {
		return safearena.Scoped(func(a *safearena.Arena) HTTPResponse {
			// Parse and process request with arena allocations
			parseBuffer := safearena.AllocSlice[byte](a, 4096)
			tempData := safearena.Alloc(a, struct {
				parsed map[string]interface{}
			}{
				parsed: make(map[string]interface{}),
			})

			buf := parseBuffer.Get()
			copy(buf, req.Body)

			temp := tempData.Get()
			temp.parsed["method"] = req.Method
			temp.parsed["path"] = req.Path

			// Return heap-allocated response
			return HTTPResponse{
				StatusCode: 200,
				Body:       fmt.Sprintf("Processed %s %s", req.Method, req.Path),
			}
		})
	}

	req := HTTPRequest{
		Method:  "GET",
		Path:    "/api/test",
		Headers: map[string]string{"User-Agent": "test"},
		Body:    []byte("test body"),
	}

	resp := handleRequest(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// Integration test: Timeout scenario
func TestIntegrationWithTimeout(t *testing.T) {
	done := make(chan bool)

	go func() {
		safearena.Scoped(func(a *safearena.Arena) int {
			// Simulate work
			for i := 0; i < 1000; i++ {
				_ = safearena.Alloc(a, i)
			}
			return 0
		})
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("arena operation timed out")
	}
}

// Benchmark: Integration benchmark for realistic workload
func BenchmarkIntegrationRealisticWorkload(b *testing.B) {
	type WorkItem struct {
		ID   int
		Data []byte
	}

	processWorkItem := func(item WorkItem) int {
		return safearena.Scoped(func(a *safearena.Arena) int {
			buffer := safearena.AllocSlice[byte](a, 1024)
			temp := safearena.Alloc(a, struct{ count int }{})

			buf := buffer.Get()
			copy(buf, item.Data)

			t := temp.Get()
			t.count = len(item.Data)

			return t.count
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := WorkItem{
			ID:   i,
			Data: []byte(fmt.Sprintf("data-%d", i)),
		}
		_ = processWorkItem(item)
	}
}
