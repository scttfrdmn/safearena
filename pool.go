package safearena

import "sync"

// Pool is a thread-safe pool of reusable arenas. Put resets the arena and
// returns it to the pool; the next Get reuses it. This amortizes arena-creation
// cost in high-throughput workloads such as per-request HTTP processing.
//
// The zero value is ready to use.
//
// Example:
//
//	var pool safearena.Pool
//
//	func handleRequest(data []byte) Result {
//	    a := pool.Get()
//	    defer pool.Put(a)
//	    tmp := safearena.Alloc(a, WorkData{})
//	    // ... process ...
//	    return result // heap-allocated, safe to return
//	}
type Pool struct {
	p sync.Pool
}

// Get returns an arena from the pool, ready for use with no prior allocations.
// If the pool is empty a new arena is created.
func (p *Pool) Get() *Arena {
	if v := p.p.Get(); v != nil {
		return v.(*Arena)
	}
	return New()
}

// Put resets the arena and returns it to the pool for reuse.
// All Ptr[T] and Slice[T] values previously allocated from this arena are
// invalidated; any subsequent access to them will panic with "use after reset".
//
// The arena must not have been freed with Free() before calling Put.
// Panics if called on a freed arena.
func (p *Pool) Put(a *Arena) {
	a.Reset()
	p.p.Put(a)
}
