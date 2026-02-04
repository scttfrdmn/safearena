package testdata

import "arena"

type TestData struct {
	X int
	Y string
}

// Test 1: Direct return - SHOULD CATCH
func test1DirectReturn() *TestData {
	a := arena.NewArena()
	defer a.Free()
	return arena.New[TestData](a)
}

// Test 2: Variable then return - SHOULD CATCH
func test2VarReturn() *TestData {
	a := arena.NewArena()
	defer a.Free()
	d := arena.New[TestData](a)
	return d
}

// Test 3: Global escape - SHOULD CATCH
var global *TestData

func test3GlobalEscape() {
	a := arena.NewArena()
	defer a.Free()
	global = arena.New[TestData](a)
}

// Test 4: Use after explicit free - SHOULD CATCH
func test4UseAfterFree() int {
	a := arena.NewArena()
	d := arena.New[TestData](a)
	a.Free()
	return d.X
}

// Test 5: Safe - return value copy - SHOULD NOT CATCH
func test5SafeValueReturn() int {
	a := arena.NewArena()
	defer a.Free()
	d := arena.New[TestData](a)
	return d.X
}

// Test 6: Safe - heap copy - SHOULD NOT CATCH
func test6SafeHeapCopy() *TestData {
	a := arena.NewArena()
	defer a.Free()
	d := arena.New[TestData](a)
	return &TestData{X: d.X, Y: d.Y}
}

// Test 7: Safe - no escape - SHOULD NOT CATCH
func test7SafeNoEscape() int {
	a := arena.NewArena()
	defer a.Free()
	d := arena.New[TestData](a)
	d.X = 42
	return d.X
}
