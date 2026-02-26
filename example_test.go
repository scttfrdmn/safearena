package safearena_test

import (
	"fmt"

	"github.com/scttfrdmn/safearena"
)

// Example demonstrates basic arena usage with automatic cleanup.
func Example() {
	// Scoped automatically frees the arena when done
	result := safearena.Scoped(func(a *safearena.Arena) string {
		// Allocate in arena
		data := safearena.Alloc(a, "Hello, Arena!")

		// Use the data
		return *data.Get()
	})

	fmt.Println(result)
	// Output: Hello, Arena!
}

// ExampleAlloc shows how to allocate a value in an arena.
func ExampleAlloc() {
	a := safearena.New()
	defer a.Free()

	// Allocate a struct in the arena
	type Config struct {
		Host string
		Port int
	}

	config := safearena.Alloc(a, Config{
		Host: "localhost",
		Port: 8080,
	})

	// Access the value safely
	fmt.Printf("%s:%d", config.Get().Host, config.Get().Port)
	// Output: localhost:8080
}

// ExampleScoped demonstrates the safest pattern for arena usage.
// The arena is automatically freed when the function returns.
func ExampleScoped() {
	result := safearena.Scoped(func(a *safearena.Arena) int {
		// All allocations are arena-scoped
		numbers := make([]int, 100)
		for i := range numbers {
			numbers[i] = i
		}

		sum := 0
		for _, n := range numbers {
			sum += n
		}

		// Return heap-allocated result
		return sum
	})

	fmt.Println(result)
	// Output: 4950
}

// ExampleClone shows how to safely copy arena data to the heap.
func ExampleClone() {
	a := safearena.New()

	// Allocate in arena
	data := safearena.Alloc(a, struct {
		Name  string
		Value int
	}{
		Name:  "example",
		Value: 42,
	})

	// Clone to heap before freeing arena
	heapCopy := safearena.Clone(data)

	// Safe to free arena now
	a.Free()

	// heapCopy is still valid
	fmt.Printf("%s: %d", heapCopy.Name, heapCopy.Value)
	// Output: example: 42
}

// ExampleAllocSlice demonstrates arena-allocated slice usage.
func ExampleAllocSlice() {
	result := safearena.Scoped(func(a *safearena.Arena) int {
		// Allocate a large buffer in the arena
		buffer := safearena.AllocSlice[byte](a, 1024)

		// Use the buffer
		slice := buffer.Get()
		copy(slice, []byte("temporary data"))

		// Return just the length
		return len(slice)
	})

	fmt.Println(result)
	// Output: 1024
}

// ExampleArena_Free shows manual arena management.
func ExampleArena_Free() {
	a := safearena.New()

	// Allocate some data
	data := safearena.Alloc(a, 42)
	fmt.Println(*data.Get())

	// Free the arena manually
	a.Free()

	// Attempting to use data after Free will panic
	// _ = data.Get() // This would panic!

	// Output: 42
}

// ExampleSlice_Get demonstrates accessing an arena-allocated slice.
func ExampleSlice_Get() {
	result := safearena.Scoped(func(a *safearena.Arena) int {
		nums := safearena.AllocSlice[int](a, 5)

		// Get returns the underlying slice, valid while the arena is alive
		slice := nums.Get()
		for i := range slice {
			slice[i] = i + 1 // 1, 2, 3, 4, 5
		}

		sum := 0
		for _, v := range slice {
			sum += v
		}
		return sum
	})

	fmt.Println(result)
	// Output: 15
}

// ExamplePtr_Get demonstrates safe pointer dereferencing.
func ExamplePtr_Get() {
	result := safearena.Scoped(func(a *safearena.Arena) string {
		// Allocate a string
		str := safearena.Alloc(a, "safe access")

		// Get returns a pointer that's valid while arena is alive
		ptr := str.Get()

		return *ptr
	})

	fmt.Println(result)
	// Output: safe access
}

// ExamplePtr_Deref shows how to copy a value out of the arena.
func ExamplePtr_Deref() {
	result := safearena.Scoped(func(a *safearena.Arena) int {
		num := safearena.Alloc(a, 100)

		// Deref copies the value (not just a pointer)
		value := num.Deref()

		return value
	})

	fmt.Println(result)
	// Output: 100
}

// Example_requestProcessing shows a real-world HTTP request pattern.
func Example_requestProcessing() {
	type Request struct {
		ID   int
		Body []byte
	}

	type Response struct {
		Status int
		Result string
	}

	processRequest := func(req Request) Response {
		return safearena.Scoped(func(a *safearena.Arena) Response {
			// Allocate temporary buffers in arena
			workBuffer := safearena.AllocSlice[byte](a, 4096)
			tempData := safearena.Alloc(a, struct {
				parsed map[string]string
			}{
				parsed: make(map[string]string),
			})

			// Simulate processing
			buf := workBuffer.Get()
			copy(buf, req.Body)

			temp := tempData.Get()
			temp.parsed["request_id"] = fmt.Sprintf("%d", req.ID)

			// Return heap-allocated response
			return Response{
				Status: 200,
				Result: "processed",
			}
		})
		// Arena automatically freed here
	}

	req := Request{ID: 1, Body: []byte("data")}
	resp := processRequest(req)

	fmt.Printf("Status: %d, Result: %s", resp.Status, resp.Result)
	// Output: Status: 200, Result: processed
}

// ExampleArena_Reset shows how to reuse an arena without allocating a new one.
func ExampleArena_Reset() {
	a := safearena.New()
	defer a.Free()

	// First use
	p1 := safearena.Alloc(a, 42)
	fmt.Println(p1.Deref())

	// Reset reclaims all memory; existing Ptr[T] values are invalidated
	a.Reset()

	// Arena is ready for reuse; allocate fresh values
	p2 := safearena.Alloc(a, 99)
	fmt.Println(p2.Deref())

	// Output:
	// 42
	// 99
}

// ExamplePool demonstrates arena reuse across multiple calls using a Pool.
func ExamplePool() {
	var pool safearena.Pool

	process := func(n int) int {
		a := pool.Get()
		defer pool.Put(a)

		p := safearena.Alloc(a, n*2)
		return p.Deref()
	}

	fmt.Println(process(10))
	fmt.Println(process(21))
	// Output:
	// 20
	// 42
}

// ExampleNewStringBuilder demonstrates arena-allocated string building.
func ExampleNewStringBuilder() {
	result := safearena.Scoped(func(a *safearena.Arena) string {
		sb := safearena.NewStringBuilder(a, 64)
		sb.Get().Append("Hello")
		sb.Get().Append(", ")
		sb.Get().Append("World!")
		return sb.Get().String()
	})

	fmt.Println(result)
	// Output: Hello, World!
}

// ExampleNewWithFinalizer demonstrates leak detection for arenas.
func ExampleNewWithFinalizer() {
	// NewWithFinalizer prints a warning if the arena is GC'd without Free().
	// Use during development/testing to catch arena leaks.
	a := safearena.NewWithFinalizer()
	defer a.Free() // Without this, a warning would print at GC time

	p := safearena.Alloc(a, 7)
	fmt.Println(p.Deref())
	// Output: 7
}

// Example_safetyCheck demonstrates runtime safety checks.
func Example_safetyCheck() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Caught panic: use after free detected")
		}
	}()

	a := safearena.New()
	data := safearena.Alloc(a, "test")

	// Free the arena
	a.Free()

	// This will panic with a helpful error message
	_ = data.Get()

	// Output: Caught panic: use after free detected
}
