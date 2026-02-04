package main

// Final working version that handles generics properly

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

var AnalyzerFinal = &analysis.Analyzer{
	Name:     "arenacheck",
	Doc:      "checks for unsafe arena usage patterns",
	Run:      runFinal,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
}

func runFinal(pass *analysis.Pass) (interface{}, error) {
	ssaProg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	for _, fn := range ssaProg.SrcFuncs {
		if fn == nil || fn.Blocks == nil {
			continue
		}
		checkFunctionFinal(pass, fn)
	}

	return nil, nil
}

func checkFunctionFinal(pass *analysis.Pass, fn *ssa.Function) {
	arenas := make(map[ssa.Value]bool)
	// Map from SSA value to the arena it was allocated from
	allocToArena := make(map[ssa.Value]ssa.Value)

	// First pass: find all arena.NewArena() calls and arena.New() calls
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok {
				callee := call.Call.StaticCallee()
				if callee == nil {
					continue
				}

				// Get the full function name
				fullName := callee.String()

				// arena.NewArena()
				if strings.Contains(fullName, "arena.NewArena") {
					arenas[call] = true
				}

				// arena.New[T] - matches instantiated generic functions
				if strings.Contains(fullName, "arena.New[") && len(call.Call.Args) > 0 {
					arenaArg := call.Call.Args[0]
					allocToArena[call] = arenaArg
				}
			}
		}
	}

	// Second pass: check for escapes
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			// Check returns
			if ret, ok := instr.(*ssa.Return); ok {
				for _, result := range ret.Results {
					// Check if this result is arena-allocated or points to one
					if escapesViaReturn(result, allocToArena, arenas) {
						pass.Reportf(ret.Pos(),
							"arena-allocated value may escape via return")
					}
				}
			}

			// Check stores to globals
			if store, ok := instr.(*ssa.Store); ok {
				if isGlobalVar(store.Addr) {
					if escapesViaStore(store.Val, allocToArena, arenas) {
						pass.Reportf(store.Pos(),
							"arena-allocated value may escape to global variable")
					}
				}
			}
		}
	}
}

func escapesViaReturn(val ssa.Value, allocToArena map[ssa.Value]ssa.Value, arenas map[ssa.Value]bool) bool {
	// Direct allocation
	if _, ok := allocToArena[val]; ok {
		return true
	}

	// Check referrers - trace back through loads, field accesses, etc.
	return tracesToArenaAlloc(val, allocToArena, make(map[ssa.Value]bool))
}

func escapesViaStore(val ssa.Value, allocToArena map[ssa.Value]ssa.Value, arenas map[ssa.Value]bool) bool {
	if _, ok := allocToArena[val]; ok {
		return true
	}
	return tracesToArenaAlloc(val, allocToArena, make(map[ssa.Value]bool))
}

func tracesToArenaAlloc(val ssa.Value, allocToArena map[ssa.Value]ssa.Value, visited map[ssa.Value]bool) bool {
	if visited[val] {
		return false
	}
	visited[val] = true

	// Direct arena allocation
	if _, ok := allocToArena[val]; ok {
		return true
	}

	// Trace through operations
	switch v := val.(type) {
	case *ssa.UnOp:
		// Dereference - check operand
		return tracesToArenaAlloc(v.X, allocToArena, visited)
	case *ssa.FieldAddr:
		// Field access - check parent struct
		return tracesToArenaAlloc(v.X, allocToArena, visited)
	case *ssa.IndexAddr:
		// Array/slice index - check array
		return tracesToArenaAlloc(v.X, allocToArena, visited)
	case *ssa.Phi:
		// Phi node - check all edges
		for _, edge := range v.Edges {
			if tracesToArenaAlloc(edge, allocToArena, visited) {
				return true
			}
		}
	}

	return false
}

func isGlobalVar(val ssa.Value) bool {
	_, ok := val.(*ssa.Global)
	return ok
}

// Also run a simpler check for debugging
func printSSA(fn *ssa.Function) {
	fmt.Printf("\n=== SSA for %s ===\n", fn.Name())
	for _, block := range fn.Blocks {
		fmt.Printf("Block %d:\n", block.Index)
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok {
				if callee := call.Call.StaticCallee(); callee != nil {
					fmt.Printf("  Call: %s\n", callee.String())
				}
			}
		}
	}
}
