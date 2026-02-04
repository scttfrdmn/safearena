package testdata

import "arena"

type MyData struct {
	Value int
}

// Simple direct return test
func directReturn() *MyData {
	a := arena.NewArena()
	defer a.Free()
	return arena.New[MyData](a) // This should be caught!
}
