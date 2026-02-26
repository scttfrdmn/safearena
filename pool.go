package safearena

import (
	"sync"
	"sync/atomic"
)

// Pool is a thread-safe pool of reusable arenas. Put resets the arena and
// returns it to the pool; the next Get reuses it. This amortizes arena-creation
// cost in high-throughput workloads such as per-request HTTP processing.
//
// Pool.Get and Pool.Put are safe for concurrent use by multiple goroutines.
// However, each individual Arena returned by Get must not be shared across
// goroutines — give each goroutine its own arena from the pool.
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
//
// PoolStats holds runtime statistics for a Pool.
// All fields are cumulative since the Pool was created.
type PoolStats struct {
	Gets    int64 // total Pool.Get calls
	Puts    int64 // total Pool.Put calls
	Created int64 // arenas created (pool was empty)
	Reused  int64 // arenas returned from pool (pool hit)
}

type Pool struct {
	p       sync.Pool
	gets    atomic.Int64
	puts    atomic.Int64
	created atomic.Int64
	reused  atomic.Int64
}

// Stats returns cumulative statistics for the pool.
func (p *Pool) Stats() PoolStats {
	return PoolStats{
		Gets:    p.gets.Load(),
		Puts:    p.puts.Load(),
		Created: p.created.Load(),
		Reused:  p.reused.Load(),
	}
}

// Get returns an arena from the pool, ready for use with no prior allocations.
// If the pool is empty a new arena is created.
func (p *Pool) Get() *Arena {
	p.gets.Add(1)
	if v := p.p.Get(); v != nil {
		p.reused.Add(1)
		return v.(*Arena)
	}
	p.created.Add(1)
	return New()
}

// Put resets the arena and returns it to the pool for reuse.
// All Ptr[T] and Slice[T] values previously allocated from this arena are
// invalidated; any subsequent access to them will panic with "use after reset".
//
// The arena must not have been freed with Free() before calling Put.
// Panics if called on a freed arena.
func (p *Pool) Put(a *Arena) {
	if a.freed.Load() {
		stack := captureStack(2)
		panic(errorWithHint(a.id, "Pool.Put on freed arena", stack, hintPoolPutFreed))
	}
	p.puts.Add(1)
	a.Reset()
	p.p.Put(a)
}
