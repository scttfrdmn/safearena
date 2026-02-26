package main

// safearena_check.go — detects unsafe usage of safearena.Ptr[T] and Slice[T].
//
// Detected patterns:
//   - safearena.Ptr[T] or Slice[T] escaping via function return
//   - safearena.Ptr[T] or Slice[T] stored to a global variable
//   - Raw *T / []T from a .Get() call escaping via return
//   - Raw *T / []T from a .Get() call stored to a global variable
//
// Safe patterns (not flagged):
//   - Calling .Deref() to obtain a copy of the value
//   - Calling safearena.Clone() to copy to the heap
//   - Calling .Get() and using the result only within the function scope

import (
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

const safeArenaPath = "github.com/scttfrdmn/safearena"

// checkFunctionForSafeArena checks a single SSA function for unsafe SafeArena usage.
func checkFunctionForSafeArena(pass *analysis.Pass, fn *ssa.Function) {
	// Skip the safearena package itself — Alloc/AllocSlice legitimately return Ptr[T]/Slice[T].
	if fn.Package() != nil && fn.Package().Pkg.Path() == safeArenaPath {
		return
	}

	if fn.Blocks == nil {
		return
	}

	// First pass: collect .Get() call results and store targets.
	getResults := make(map[ssa.Value]bool)
	storesTo := make(map[ssa.Value]ssa.Value) // addr → stored value

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch v := instr.(type) {
			case *ssa.Store:
				storesTo[v.Addr] = v.Val
			case *ssa.Call:
				if isGetCallOnSafeArena(v) {
					getResults[v] = true
				}
			}
		}
	}

	// Second pass: check for escaping values.
	// reportedSources deduplicates across the normal-return and panic-return blocks
	// that Go SSA creates for functions with defer. Both blocks load from the same
	// return-variable slot; without dedup, we'd report the same escape twice.
	reportedSources := make(map[ssa.Value]bool)

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch v := instr.(type) {
			case *ssa.Return:
				for _, result := range v.Results {
					src := canonicalSource(result, storesTo)
					if reportedSources[src] {
						continue
					}
					pos := effectivePos(result, storesTo, v.Pos())
					if pos.IsValid() {
						reportedSources[src] = true
						checkSafeArenaEscape(pass, result, pos, getResults, storesTo, "return")
					}
				}
			case *ssa.Store:
				if isGlobalVar(v.Addr) {
					checkSafeArenaEscape(pass, v.Val, effectivePos(v.Val, storesTo, v.Pos()), getResults, storesTo, "global variable")
				}
			}
		}
	}
}

// checkSafeArenaEscape reports a diagnostic if val is an unsafe SafeArena value.
// pos is the source position at which to report; it should already be the "effective"
// position (accounting for defer-induced return-variable indirection).
func checkSafeArenaEscape(pass *analysis.Pass, val ssa.Value, pos token.Pos, getResults map[ssa.Value]bool, storesTo map[ssa.Value]ssa.Value, kind string) {
	if !pos.IsValid() {
		return
	}

	// Check if the value itself is Ptr[T] or Slice[T].
	if name := safeArenaWrapperTypeName(val.Type()); name != "" {
		pass.Reportf(pos, "safearena.%s escapes via %s; use Deref() or Clone() to copy value to heap",
			name, kind)
		return
	}

	// Check if the value traces back to a .Get() call result.
	visited := make(map[ssa.Value]bool)
	if tracesBackToGet(val, getResults, storesTo, visited) {
		pass.Reportf(pos, "raw pointer from safearena .Get() escapes via %s; arena may be freed by caller",
			kind)
	}
}

// canonicalSource returns the "root" value for an SSA value.
// For loads from local alloc slots, it returns the stored value so that loads
// from the same slot across multiple SSA blocks (e.g., normal-path and panic-path
// return blocks in defer-using functions) are identified as the same logical value.
func canonicalSource(val ssa.Value, storesTo map[ssa.Value]ssa.Value) ssa.Value {
	if unop, ok := val.(*ssa.UnOp); ok {
		if stored, ok := storesTo[unop.X]; ok {
			return stored
		}
	}
	return val
}

// effectivePos returns the best source position for a value.
// When a function has defer, the return value may go through a synthetic store/load
// that loses its position. In that case we trace back one level through the load chain
// to find the original computation's position. Falls back to instrPos if all else fails.
func effectivePos(val ssa.Value, storesTo map[ssa.Value]ssa.Value, instrPos token.Pos) token.Pos {
	if pos := val.Pos(); pos.IsValid() {
		return pos
	}
	// val may be a synthetic load from the return variable slot created for defer.
	// Trace through one load to find the stored value (the original computation).
	if unop, ok := val.(*ssa.UnOp); ok {
		if stored, ok := storesTo[unop.X]; ok {
			if pos := stored.Pos(); pos.IsValid() {
				return pos
			}
		}
	}
	return instrPos
}

// isGetCallOnSafeArena returns true if the call is .Get() on a safearena.Ptr[T] or Slice[T].
// Handles generic instantiation names like "Get[int]" in addition to plain "Get".
func isGetCallOnSafeArena(call *ssa.Call) bool {
	callee := call.Call.StaticCallee()
	if callee == nil {
		return false
	}
	// Base method name, stripping generic type arguments: "Get[int]" → "Get".
	name := callee.Name()
	if idx := strings.IndexByte(name, '['); idx >= 0 {
		name = name[:idx]
	}
	if name != "Get" {
		return false
	}
	// callee.Package() returns nil for instantiated generic methods (e.g., Get[int]).
	// Use callee.String() which includes the full receiver type path, e.g.:
	//   "(github.com/scttfrdmn/safearena.Ptr[int]).Get[int]"
	return strings.Contains(callee.String(), "("+safeArenaPath+".")
}

// safeArenaWrapperTypeName returns "Ptr" or "Slice" if t is safearena.Ptr[T] or Slice[T], else "".
func safeArenaWrapperTypeName(t types.Type) string {
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	// Handle instantiated generics: named.Origin() gives the generic Ptr[T any] type.
	origin := named.Origin()
	obj := origin.Obj()
	if obj.Pkg() == nil || obj.Pkg().Path() != safeArenaPath {
		return ""
	}
	switch obj.Name() {
	case "Ptr", "Slice":
		return obj.Name()
	}
	return ""
}

// tracesBackToGet follows SSA load-store chains to determine whether val
// originated from a .Get() call on a SafeArena wrapper.
//
// It traces through local variable alloc slots (store/load pairs) and phi nodes,
// but does NOT trace through arbitrary pointer dereferences — dereferencing a Get()
// result reads the stored value and is safe to return.
func tracesBackToGet(val ssa.Value, getResults map[ssa.Value]bool, storesTo map[ssa.Value]ssa.Value, visited map[ssa.Value]bool) bool {
	if visited[val] {
		return false
	}
	visited[val] = true

	// Direct match: this value IS a .Get() call result.
	if getResults[val] {
		return true
	}

	switch v := val.(type) {
	case *ssa.UnOp:
		// Trace only through local variable loads (alloc slots tracked in storesTo).
		// Do NOT trace through arbitrary pointer dereferences — that would cause false
		// positives when code legitimately reads *ptr and returns the dereferenced value.
		if stored, ok := storesTo[v.X]; ok {
			return tracesBackToGet(stored, getResults, storesTo, visited)
		}

	case *ssa.Phi:
		// A value may flow through a phi node in control flow.
		for _, edge := range v.Edges {
			if tracesBackToGet(edge, getResults, storesTo, visited) {
				return true
			}
		}
	}

	return false
}
