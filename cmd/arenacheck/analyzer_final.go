package main

// Final working version with store/load tracking

import (
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

var AnalyzerFinal2 = &analysis.Analyzer{
	Name:     "arenacheck",
	Doc:      "checks for unsafe arena usage patterns",
	Run:      runFinal2,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
}

func runFinal2(pass *analysis.Pass) (interface{}, error) {
	ssaProg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	for _, fn := range ssaProg.SrcFuncs {
		if fn == nil || fn.Blocks == nil {
			continue
		}
		checkFunctionFinal2(pass, fn)
	}

	return nil, nil
}

type arenaInfo struct {
	value ssa.Value
}

type allocInfo struct {
	arena    *arenaInfo
	value    ssa.Value
	allocPos string
}

func checkFunctionFinal2(pass *analysis.Pass, fn *ssa.Function) {
	arenas := make(map[ssa.Value]*arenaInfo)
	allocations := make(map[ssa.Value]*allocInfo)
	// Track what arena values are stored into what addresses
	storesTo := make(map[ssa.Value]ssa.Value) // addr -> value
	// Track Free() calls: instruction -> arena
	freeInstrs := make(map[ssa.Instruction]ssa.Value)

	// First pass: find arenas, allocations, and Free() calls
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok {
				callee := call.Call.StaticCallee()
				if callee == nil {
					continue
				}

				fullName := callee.String()

				// arena.NewArena()
				if strings.Contains(fullName, "arena.NewArena") {
					arenas[call] = &arenaInfo{value: call}
				}

				// arena.New[T]
				if strings.Contains(fullName, "arena.New[") && len(call.Call.Args) > 0 {
					arenaArg := call.Call.Args[0]
					if arenaInfo, ok := arenas[arenaArg]; ok {
						allocations[call] = &allocInfo{
							arena:    arenaInfo,
							value:    call,
							allocPos: pass.Fset.Position(call.Pos()).String(),
						}
					}
				}

				// arena.Free() - track explicit Free calls
				if strings.Contains(fullName, ".Free") || (callee.Name() == "Free" && len(call.Call.Args) > 0) {
					// Try to find which arena is being freed
					// For method calls: a.Free()
					if len(call.Call.Args) > 0 {
						arenaArg := call.Call.Args[0]
						if _, ok := arenas[arenaArg]; ok {
							freeInstrs[call] = arenaArg
						}
					}
				}
			}

			// Track stores: *addr = value
			if store, ok := instr.(*ssa.Store); ok {
				storesTo[store.Addr] = store.Val
			}
		}
	}

	// Second pass: check returns, stores, and use-after-free
	for _, block := range fn.Blocks {
		freedArenas := make(map[ssa.Value]bool) // Track which arenas are freed in this block

		for _, instr := range block.Instrs {
			// Track when arenas are freed
			if arena, ok := freeInstrs[instr]; ok {
				freedArenas[arena] = true
			}

			// Check for uses of allocations after their arena was freed
			if len(freedArenas) > 0 {
				checkUseAfterFree(pass, instr, allocations, freedArenas, storesTo)
			}

			// Check returns
			if ret, ok := instr.(*ssa.Return); ok {
				for _, result := range ret.Results {
					if alloc := findAllocation(result, allocations, storesTo); alloc != nil {
						// Type check: only flag pointers
						if isPointerType(result.Type()) {
							pass.Reportf(ret.Pos(),
								"arena-allocated value escapes via return (allocated at %s)",
								alloc.allocPos)
						}
					}
				}
			}

			// Check global stores
			if store, ok := instr.(*ssa.Store); ok {
				if isGlobalVar(store.Addr) {
					if alloc := findAllocation(store.Val, allocations, storesTo); alloc != nil {
						pass.Reportf(store.Pos(),
							"arena-allocated value escapes to global variable (allocated at %s)",
							alloc.allocPos)
					}
				}
			}
		}
	}
}

// findAllocation traces a value back to see if it comes from an arena allocation
func findAllocation(val ssa.Value, allocations map[ssa.Value]*allocInfo, storesTo map[ssa.Value]ssa.Value) *allocInfo {
	visited := make(map[ssa.Value]bool)
	return findAllocationRec(val, allocations, storesTo, visited)
}

func findAllocationRec(val ssa.Value, allocations map[ssa.Value]*allocInfo, storesTo map[ssa.Value]ssa.Value, visited map[ssa.Value]bool) *allocInfo {
	if visited[val] {
		return nil
	}
	visited[val] = true

	// Direct match
	if alloc, ok := allocations[val]; ok {
		return alloc
	}

	// Trace through different operations
	switch v := val.(type) {
	case *ssa.UnOp:
		// Could be *x (load) or &x (address-of)
		operand := v.X

		// If this is a load from a local variable, check what was stored there
		if stored, ok := storesTo[operand]; ok {
			if alloc := findAllocationRec(stored, allocations, storesTo, visited); alloc != nil {
				return alloc
			}
		}

		// Also check the operand itself
		return findAllocationRec(operand, allocations, storesTo, visited)

	case *ssa.FieldAddr:
		return findAllocationRec(v.X, allocations, storesTo, visited)

	case *ssa.IndexAddr:
		return findAllocationRec(v.X, allocations, storesTo, visited)

	case *ssa.Phi:
		for _, edge := range v.Edges {
			if alloc := findAllocationRec(edge, allocations, storesTo, visited); alloc != nil {
				return alloc
			}
		}

	case *ssa.MakeInterface:
		return findAllocationRec(v.X, allocations, storesTo, visited)
	}

	return nil
}

func isPointerType(t types.Type) bool {
	switch t := t.(type) {
	case *types.Pointer:
		return true
	case *types.Named:
		return isPointerType(t.Underlying())
	}
	return false
}

func isGlobalVar(val ssa.Value) bool {
	_, ok := val.(*ssa.Global)
	return ok
}

// checkUseAfterFree detects if an instruction uses an allocation after its arena was freed
func checkUseAfterFree(pass *analysis.Pass, instr ssa.Instruction, allocations map[ssa.Value]*allocInfo, freedArenas map[ssa.Value]bool, storesTo map[ssa.Value]ssa.Value) {
	// Get all operands of this instruction
	operandPtrs := instr.Operands(nil)

	for _, operandPtr := range operandPtrs {
		if operandPtr == nil || *operandPtr == nil {
			continue
		}
		operand := *operandPtr

		// Check if this operand traces to an allocation
		if alloc := findAllocation(operand, allocations, storesTo); alloc != nil {
			// Check if this allocation's arena was freed
			if freedArenas[alloc.arena.value] {
				pass.Reportf(instr.Pos(),
					"use of arena allocation after Free() (allocated at %s)",
					alloc.allocPos)
				return // Only report once per instruction
			}
		}
	}
}

func debugValue(val ssa.Value) string {
	return fmt.Sprintf("%T: %v", val, val.Name())
}
