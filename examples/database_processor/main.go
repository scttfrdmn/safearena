package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/scttfrdmn/safearena"
)

// Database query processor example: Arena-scoped query processing
// Perfect for request-scoped allocations in database middleware

// QueryResult represents a database row
type QueryResult struct {
	ID     int
	Name   string
	Email  string
	Active bool
}

// ProcessingState holds temporary state during query processing
type ProcessingState struct {
	FilterBuffer safearena.Slice[byte]
	TempStrings  []string
	RowCount     int
}

// processQueryWithArena processes database results using arena for temporaries
func processQueryWithArena(rows []QueryResult, filterTerm string) []QueryResult {
	return safearena.Scoped(func(a *safearena.Arena) []QueryResult {
		// Allocate processing state in arena
		state := safearena.Alloc(a, ProcessingState{
			FilterBuffer: safearena.AllocSlice[byte](a, 1024),
			TempStrings:  make([]string, 0, 100),
			RowCount:     0,
		})

		// Allocate working buffers
		workBuffer := safearena.AllocSlice[byte](a, 4096)
		resultBuffer := make([]QueryResult, 0, len(rows))

		s := state.Get()
		buf := workBuffer.Get()

		// Process each row
		for _, row := range rows {
			// Use arena for temporary string processing
			s.RowCount++

			// Temporary string manipulation in arena buffer
			normalized := strings.ToLower(row.Name)
			copy(buf, []byte(normalized))
			s.TempStrings = append(s.TempStrings, normalized)

			// Filter logic
			if strings.Contains(normalized, filterTerm) && row.Active {
				// Add to results (heap-allocated)
				resultBuffer = append(resultBuffer, row)
			}
		}

		fmt.Printf("Processed %d rows, found %d matches\n",
			s.RowCount, len(resultBuffer))

		// Return heap-allocated results
		// Arena with all temporary allocations is freed here
		return resultBuffer
	})
}

// processQueryWithoutArena is traditional approach with GC
func processQueryWithoutArena(rows []QueryResult, filterTerm string) []QueryResult {
	// All allocations go through GC
	filterBuffer := make([]byte, 1024)
	tempStrings := make([]string, 0, 100)
	workBuffer := make([]byte, 4096)
	resultBuffer := make([]QueryResult, 0, len(rows))

	rowCount := 0
	for _, row := range rows {
		rowCount++

		normalized := strings.ToLower(row.Name)
		copy(workBuffer, []byte(normalized))
		tempStrings = append(tempStrings, normalized)

		if strings.Contains(normalized, filterTerm) && row.Active {
			resultBuffer = append(resultBuffer, row)
		}
	}

	_ = filterBuffer // avoid unused warning
	fmt.Printf("Processed %d rows, found %d matches\n", rowCount, len(resultBuffer))
	return resultBuffer
}

// Simulate database query results
func generateMockData(count int) []QueryResult {
	results := make([]QueryResult, count)
	names := []string{"Alice Johnson", "Bob Smith", "Charlie Brown", "Diana Prince", "Eve Anderson"}

	for i := 0; i < count; i++ {
		results[i] = QueryResult{
			ID:     i + 1,
			Name:   names[i%len(names)],
			Email:  fmt.Sprintf("user%d@example.com", i),
			Active: i%3 != 0, // 2/3 active
		}
	}
	return results
}

func main() {
	fmt.Println("Database Query Processor Example\n")

	// Generate test data
	rowCount := 1000
	mockData := generateMockData(rowCount)
	filterTerm := "johnson"

	fmt.Printf("Processing %d database rows...\n\n", rowCount)

	// Benchmark with arena
	iterations := 1000
	start := time.Now()
	var arenaResults []QueryResult
	for i := 0; i < iterations; i++ {
		arenaResults = processQueryWithArena(mockData, filterTerm)
	}
	arenaTime := time.Since(start)

	// Benchmark without arena
	start = time.Now()
	var gcResults []QueryResult
	for i := 0; i < iterations; i++ {
		gcResults = processQueryWithoutArena(mockData, filterTerm)
	}
	gcTime := time.Since(start)

	fmt.Printf("\nBenchmark Results (%d iterations):\n", iterations)
	fmt.Printf("With Arena:    %v (%v per query)\n", arenaTime, arenaTime/time.Duration(iterations))
	fmt.Printf("Without Arena: %v (%v per query)\n", gcTime, gcTime/time.Duration(iterations))
	fmt.Printf("Speedup: %.2fx\n", float64(gcTime)/float64(arenaTime))

	fmt.Printf("\nResults: Found %d matching rows\n", len(arenaResults))
	fmt.Printf("Sample: %+v\n", arenaResults[0])

	// Verify results match
	if len(arenaResults) != len(gcResults) {
		fmt.Println("WARNING: Result mismatch!")
	}
}
