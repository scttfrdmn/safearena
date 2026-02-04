package testdata

import "arena"

type Data struct {
	Value int
}

// BAD: Returns arena-allocated value
func badReturn() *Data {
	a := arena.NewArena()
	defer a.Free()
	return arena.New[Data](a) // want "arena-allocated value escapes via return"
}

// BAD: Stores arena allocation to global
var globalData *Data

func badGlobal() {
	a := arena.NewArena()
	defer a.Free()
	globalData = arena.New[Data](a) // want "arena-allocated value may escape"
}

// BAD: Use after free
func badUseAfterFree() int {
	a := arena.NewArena()
	data := arena.New[Data](a)
	data.Value = 42
	a.Free()
	return data.Value // want "use of arena allocation after arena.Free()"
}

// BAD: Arena never freed
func badLeak() {
	a := arena.NewArena() // want "arena is never freed"
	_ = arena.New[Data](a)
	// Forgot to free!
}

// GOOD: Safe usage
func goodScoped() int {
	a := arena.NewArena()
	defer a.Free()
	data := arena.New[Data](a)
	data.Value = 42
	return data.Value // Copy value, not pointer
}

// GOOD: Returns heap-allocated value
func goodReturn() *Data {
	a := arena.NewArena()
	defer a.Free()

	temp := arena.New[Data](a)
	temp.Value = 42

	// Copy to heap
	result := &Data{Value: temp.Value}
	return result
}
