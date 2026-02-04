// Package safearena provides safe, ergonomic arena memory management for Go.
//
// # Overview
//
// Go's experimental arena package provides performance benefits but requires careful
// manual lifetime management. SafeArena wraps arenas with runtime safety checks that
// prevent use-after-free and double-free errors while maintaining good performance.
//
// # Quick Start
//
// The simplest and safest way to use arenas is with Scoped:
//
//	result := safearena.Scoped(func(a *safearena.Arena) Response {
//	    // Allocate temporary data in arena
//	    temp := safearena.Alloc(a, TempData{Size: 1024})
//
//	    // Use it safely
//	    process(temp.Get())
//
//	    // Return heap-allocated response
//	    return Response{Status: 200}
//	}) // Arena automatically freed here
//
// # Core Concepts
//
// Arena: A memory region that can be freed all at once. Create with New() or use Scoped()
// for automatic management.
//
// Ptr[T]: A type-safe pointer that tracks which arena it belongs to. Panics if accessed
// after the arena is freed, preventing silent memory corruption.
//
// Slice[T]: A slice wrapper with the same lifetime tracking as Ptr[T].
//
// # Safety Guarantees
//
// SafeArena prevents three common arena bugs:
//
//  1. Use-after-free: Accessing arena memory after Free() panics with helpful error
//  2. Double-free: Calling Free() twice panics to prevent corruption
//  3. Allocation after free: Allocating in freed arena panics immediately
//
// All panics include stack traces and hints for fixing the issue.
//
// # Performance
//
// SafeArena adds minimal overhead (single atomic load per access) while providing
// strong safety guarantees:
//
//	BenchmarkSafeArena    104.8 μs/op    406 KB/op    0.047ms GC pause
//	BenchmarkRegularGC     92.5 μs/op    256 KB/op    0.082ms GC pause
//
// Trade ~13% performance for 42% lower GC pause times and 100% memory safety.
//
// # Patterns
//
// Request-scoped processing:
//
//	func handleRequest(req Request) Response {
//	    return safearena.Scoped(func(a *safearena.Arena) Response {
//	        buffer := safearena.AllocSlice[byte](a, 4096)
//	        temp := safearena.Alloc(a, TempData{})
//	        // Process with arena allocations...
//	        return Response{} // Heap-allocated
//	    })
//	}
//
// Extracting data from arena:
//
//	a := safearena.New()
//	defer a.Free()
//
//	data := safearena.Alloc(a, ExpensiveStruct{})
//	// ... process ...
//
//	// Copy important results to heap
//	result := safearena.Clone(data)
//	// result is safe to use after Free()
//
// # Requirements
//
// Requires Go 1.23+ with GOEXPERIMENT=arenas environment variable set.
//
// The arena package is currently experimental. Use for research and development,
// not production systems.
//
// # Static Analysis
//
// SafeArena includes arenacheck, a static analyzer that catches arena escapes at
// compile time:
//
//	GOEXPERIMENT=arenas arenacheck ./...
//
// Or integrate with go vet:
//
//	GOEXPERIMENT=arenas go vet -vettool=$(which arenacheck) ./...
//
// # Additional Resources
//
// Repository: https://github.com/scttfrdmn/safearena
//
// Examples: https://github.com/scttfrdmn/safearena/tree/main/examples
package safearena
