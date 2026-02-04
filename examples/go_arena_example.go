package main

import (
	"arena"
	"fmt"
	"time"
)

// Request represents an incoming HTTP request
type Request struct {
	ID      int
	Path    string
	Headers map[string]string
	Body    []byte
}

// Response represents the processed response
type Response struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// ProcessingContext holds temporary data for request processing
type ProcessingContext struct {
	ParsedParams map[string]string
	TempBuffers  [][]byte
	Metadata     []string
}

// Without Arena (traditional Go with GC)
func processRequestWithGC(req *Request) *Response {
	// These allocations will be GC'd later
	ctx := &ProcessingContext{
		ParsedParams: make(map[string]string),
		TempBuffers:  make([][]byte, 0, 10),
		Metadata:     make([]string, 0, 5),
	}

	// Simulate processing with temporary allocations
	for i := 0; i < 100; i++ {
		ctx.TempBuffers = append(ctx.TempBuffers, make([]byte, 1024))
		ctx.Metadata = append(ctx.Metadata, fmt.Sprintf("meta-%d", i))
	}

	// Create response
	return &Response{
		StatusCode: 200,
		Body:       fmt.Sprintf("Processed request %d", req.ID),
		Headers:    map[string]string{"Content-Type": "text/plain"},
	}
}

// With Arena - bulk allocation and deallocation
func processRequestWithArena(req *Request) *Response {
	a := arena.NewArena()
	defer a.Free() // Free all arena allocations at once

	// Allocate in arena - these won't be GC'd
	ctx := arena.New[ProcessingContext](a)
	ctx.ParsedParams = make(map[string]string)
	ctx.TempBuffers = make([][]byte, 0, 10)
	ctx.Metadata = make([]string, 0, 5)

	// Temporary allocations in arena
	for i := 0; i < 100; i++ {
		// These can be arena-allocated too
		buffer := make([]byte, 1024)
		ctx.TempBuffers = append(ctx.TempBuffers, buffer)
		ctx.Metadata = append(ctx.Metadata, fmt.Sprintf("meta-%d", i))
	}

	// Response must be heap-allocated (outlives arena)
	// Cannot use arena.New here as it would be freed
	return &Response{
		StatusCode: 200,
		Body:       fmt.Sprintf("Processed request %d with arena", req.ID),
		Headers:    map[string]string{"Content-Type": "text/plain"},
	}
}

func benchmark(name string, fn func(*Request) *Response) {
	start := time.Now()

	for i := 0; i < 10000; i++ {
		req := &Request{
			ID:      i,
			Path:    "/api/users",
			Headers: map[string]string{"User-Agent": "Go"},
			Body:    []byte("request body"),
		}

		resp := fn(req)
		_ = resp // Use the response
	}

	elapsed := time.Since(start)
	fmt.Printf("%s: %v\n", name, elapsed)
}

func main() {
	fmt.Println("Go Arena vs GC Comparison")
	fmt.Println("Processing 10,000 requests...")

	// Warm up
	processRequestWithGC(&Request{ID: 0})
	processRequestWithArena(&Request{ID: 0})

	benchmark("With GC       ", processRequestWithGC)
	benchmark("With Arena    ", processRequestWithArena)

	fmt.Println("\nKey differences:")
	fmt.Println("- Arena: Bulk free at end of request, no GC pressure")
	fmt.Println("- GC: Objects freed individually by garbage collector")
	fmt.Println("- Arena: Better performance, but requires careful lifetime management")
}
