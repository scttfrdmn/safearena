package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/scttfrdmn/safearena"
)

// JSON parser example: Arena-allocated AST for temporary parsing
// This pattern is ideal for parsing data that's only needed during processing

// Node represents a JSON AST node (simplified)
type Node struct {
	Type     string
	Key      string
	Value    interface{}
	Children []safearena.Ptr[Node]
}

// parseJSON parses JSON into an arena-allocated AST
func parseJSON(jsonData []byte) map[string]interface{} {
	return safearena.Scoped(func(a *safearena.Arena) map[string]interface{} {
		// Parse JSON into temporary map
		var data map[string]interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			panic(err)
		}

		// Build arena-allocated AST for processing
		// In real use, this could be a complex tree structure
		result := make(map[string]interface{})

		// Use arena for temporary processing buffers
		processBuffer := safearena.AllocSlice[byte](a, 1024)
		tempNodes := make([]safearena.Ptr[Node], 0, 10)

		// Process each key-value pair
		for k, v := range data {
			// Create temporary node in arena
			node := safearena.Alloc(a, Node{
				Type:  "field",
				Key:   k,
				Value: v,
			})
			tempNodes = append(tempNodes, node)

			// Use buffer for temporary operations
			buf := processBuffer.Get()
			copy(buf, []byte(k))

			// Extract final result (heap-allocated)
			result[k] = v
		}

		// Arena freed here, but result is on heap
		return result
	})
}

// parseJSONWithoutArena is the traditional approach
func parseJSONWithoutArena(jsonData []byte) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		panic(err)
	}

	// All temporary allocations go through GC
	result := make(map[string]interface{})
	processBuffer := make([]byte, 1024)
	tempNodes := make([]interface{}, 0, 10)

	for k, v := range data {
		node := map[string]interface{}{
			"type":  "field",
			"key":   k,
			"value": v,
		}
		tempNodes = append(tempNodes, node)

		copy(processBuffer, []byte(k))
		result[k] = v
	}

	return result
}

func main() {
	// Sample JSON data
	jsonData := []byte(`{
		"name": "SafeArena",
		"version": "0.4.0",
		"language": "Go",
		"features": ["safety", "performance", "ergonomics"],
		"benchmarks": {
			"alloc_time": "104.8μs",
			"gc_pause": "0.047ms"
		}
	}`)

	fmt.Println("JSON Parser Example: Arena-allocated AST")
	fmt.Println()

	// Benchmark with arena
	start := time.Now()
	iterations := 10000
	for i := 0; i < iterations; i++ {
		_ = parseJSON(jsonData)
	}
	arenaTime := time.Since(start)

	// Benchmark without arena
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = parseJSONWithoutArena(jsonData)
	}
	gcTime := time.Since(start)

	fmt.Printf("Iterations: %d\n", iterations)
	fmt.Printf("With Arena:    %v (%v per operation)\n", arenaTime, arenaTime/time.Duration(iterations))
	fmt.Printf("Without Arena: %v (%v per operation)\n", gcTime, gcTime/time.Duration(iterations))
	fmt.Printf("Speedup: %.2fx\n", float64(gcTime)/float64(arenaTime))

	// Show parsed result
	result := parseJSON(jsonData)
	fmt.Printf("\nParsed data: %v\n", result)
}
