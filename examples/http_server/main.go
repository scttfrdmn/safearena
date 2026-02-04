package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/scttfrdmn/safearena"
)

// HTTPServer example: Use arenas for request-scoped allocations
// Each request gets its own arena that's freed after the response is sent

type Request struct {
	Method string
	Path   string
	Body   []byte
}

type Response struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// processRequest uses an arena for all temporary allocations
func processRequest(req Request) Response {
	return safearena.Scoped(func(a *safearena.Arena) Response {
		// All temporary data goes in the arena
		tempBuffer := safearena.AllocSlice[byte](a, 4096)
		parseResults := safearena.Alloc(a, struct {
			Params map[string]string
			Data   []string
		}{
			Params: make(map[string]string),
			Data:   make([]string, 0, 10),
		})

		// Simulate processing
		buf := tempBuffer.Get()
		copy(buf, req.Body)

		results := parseResults.Get()
		results.Params["method"] = req.Method
		results.Params["path"] = req.Path
		results.Data = append(results.Data, "processed")

		// Return heap-allocated response
		// Arena will be freed automatically when we return
		return Response{
			StatusCode: 200,
			Body:       fmt.Sprintf("Handled %s %s", req.Method, req.Path),
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}
	})
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Create request object
	req := Request{
		Method: r.Method,
		Path:   r.URL.Path,
		Body:   []byte("request body"),
	}

	// Process with arena (all temp allocs are arena-scoped)
	resp := processRequest(req)

	// Write response
	w.WriteHeader(resp.StatusCode)
	fmt.Fprintf(w, "%s\nProcessed in: %v\n", resp.Body, time.Since(start))
}

func main() {
	http.HandleFunc("/", handler)

	fmt.Println("Server starting on :8080")
	fmt.Println("Try: curl http://localhost:8080/hello")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
