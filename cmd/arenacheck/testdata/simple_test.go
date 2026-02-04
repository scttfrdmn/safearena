package testdata

import "arena"

type Data struct {
	Value int
}

// Test 1: Direct return (should catch)
func directReturn() *Data {
	a := arena.NewArena()
	defer a.Free()
	return arena.New[Data](a)
}

// Test 2: Variable then return (should catch)
func varReturn() *Data {
	a := arena.NewArena()
	defer a.Free()
	d := arena.New[Data](a)
	return d
}

// Test 3: Return value (should NOT catch - it's just an int)
func valueReturn() int {
	a := arena.NewArena()
	defer a.Free()
	d := arena.New[Data](a)
	return d.Value
}

// Test 4: Safe heap copy (should NOT catch)
func safeReturn() *Data {
	a := arena.NewArena()
	defer a.Free()
	d := arena.New[Data](a)
	d.Value = 42
	return &Data{Value: d.Value}
}
