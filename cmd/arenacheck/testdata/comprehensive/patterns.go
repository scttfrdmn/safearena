package testdata

import "arena"

// Test case 1: Direct return escape
func directReturnEscape() *int {
	a := arena.NewArena()
	defer a.Free()
	return arena.New[int](a) // Want "arena allocation escapes"
}

// Test case 2: Indirect return via variable
func indirectReturnEscape() *int {
	a := arena.NewArena()
	defer a.Free()
	x := arena.New[int](a)
	return x // Want "arena allocation escapes"
}

// Test case 3: Return via struct field
func structFieldEscape() *Data {
	a := arena.NewArena()
	defer a.Free()
	d := &Data{
		ptr: arena.New[int](a), // Want "arena allocation escapes"
	}
	return d
}

// Test case 4: Escape via closure
func closureEscape() func() *int {
	a := arena.NewArena()
	defer a.Free()
	x := arena.New[int](a)
	return func() *int {
		return x // Want "arena allocation escapes"
	}
}

// Test case 5: Use after free
func useAfterFree() int {
	a := arena.NewArena()
	x := arena.New[int](a)
	a.Free()
	return *x // Want "use after free"
}

// Test case 6: Escape to global
var globalPtr *int

func escapeToGlobal() {
	a := arena.NewArena()
	defer a.Free()
	globalPtr = arena.New[int](a) // Want "arena allocation escapes to global"
}

// Test case 7: Escape via channel
func escapeViaChannel(ch chan *int) {
	a := arena.NewArena()
	defer a.Free()
	ch <- arena.New[int](a) // Want "arena allocation escapes"
}

// Test case 8: Safe - value return
func safeValueReturn() int {
	a := arena.NewArena()
	defer a.Free()
	x := arena.New[int](a)
	return *x // OK - returns value, not pointer
}

// Test case 9: Safe - no escape
func safeNoEscape() {
	a := arena.NewArena()
	defer a.Free()
	x := arena.New[int](a)
	_ = *x // OK - used within lifetime
}

// Test case 10: Multiple arenas
func multipleArenas() *int {
	a1 := arena.NewArena()
	a2 := arena.NewArena()
	defer a1.Free()
	defer a2.Free()

	x := arena.New[int](a1)
	y := arena.New[int](a2)

	if *x > *y {
		return x // Want "arena allocation escapes"
	}
	return y // Want "arena allocation escapes"
}

// Test case 11: Nested function call
func nestedCall() *int {
	a := arena.NewArena()
	defer a.Free()
	return helper(a) // Want "arena allocation escapes"
}

func helper(a *arena.Arena) *int {
	return arena.New[int](a)
}

// Test case 12: Interface escape
func interfaceEscape() interface{} {
	a := arena.NewArena()
	defer a.Free()
	return arena.New[int](a) // Want "arena allocation escapes"
}

// Test case 13: Slice of arena pointers
func sliceEscape() []*int {
	a := arena.NewArena()
	defer a.Free()

	result := make([]*int, 3)
	for i := 0; i < 3; i++ {
		result[i] = arena.New[int](a) // Want "arena allocation escapes"
	}
	return result
}

// Test case 14: Map with arena pointers
func mapEscape() map[string]*int {
	a := arena.NewArena()
	defer a.Free()

	m := make(map[string]*int)
	m["key"] = arena.New[int](a) // Want "arena allocation escapes"
	return m
}

// Test case 15: Double free
func doubleFree() {
	a := arena.NewArena()
	a.Free()
	a.Free() // Want "double free"
}

// Test case 16: Safe with proper clone
func safeWithClone() *int {
	a := arena.NewArena()
	defer a.Free()

	x := arena.New[int](a)
	*x = 42

	// Clone to heap
	result := new(int)
	*result = *x
	return result // OK - heap allocated
}

// Test case 17: Complex dataflow
func complexDataflow() *int {
	a := arena.NewArena()
	defer a.Free()

	x := arena.New[int](a)
	y := x
	z := y
	return z // Want "arena allocation escapes"
}

// Test case 18: Conditional escape
func conditionalEscape(flag bool) *int {
	a := arena.NewArena()
	defer a.Free()

	x := arena.New[int](a)

	if flag {
		return x // Want "arena allocation escapes"
	}
	return nil // OK
}

// Test case 19: Loop with arena
func loopWithArena() {
	for i := 0; i < 10; i++ {
		a := arena.NewArena()
		x := arena.New[int](a)
		*x = i
		a.Free() // OK - proper lifetime
	}
}

// Test case 20: Deferred free not called
func noDefer() *int {
	a := arena.NewArena()
	// Missing: defer a.Free()
	return arena.New[int](a) // Want "arena allocation escapes" and maybe "arena not freed"
}

type Data struct {
	ptr *int
}
