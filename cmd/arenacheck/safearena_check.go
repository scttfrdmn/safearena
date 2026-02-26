package main

// safearena_check.go — detects unsafe usage of safearena.Ptr[T] and Slice[T].
//
// Detected patterns:
//   - safearena.Ptr[T] or Slice[T] escaping via function return
//   - safearena.Ptr[T] or Slice[T] stored to a global variable
//   - safearena.Ptr[T] or Slice[T] stored to a field of an escaping struct
//     (i.e. struct is a function parameter or heap-allocated with &T{}/new(T))
//   - Raw *T / []T from a .Get() call escaping via return
//   - Raw *T / []T from a .Get() call stored to a global variable
//   - Raw *T / []T from a .Get() call stored to a field of an escaping struct
//
// Safe patterns (not flagged):
//   - Calling .Deref() to obtain a copy of the value
//   - Calling safearena.Clone() to copy to the heap
//   - Calling .Get() and using the result only within the function scope
//   - Storing Ptr[T]/Slice[T] into a field of a purely stack-local struct (var s T)

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
				// Detect Ptr[T]/Slice[T] stored into a field of an escaping struct.
				// An "escaping" struct is one whose pointer either came from the
				// caller (function parameter) or was explicitly heap-allocated
				// (new(T) or &T{} — Alloc.Heap=true). Stack-local structs
				// (var s T, Alloc.Heap=false) are not flagged.
				if fa, ok := v.Addr.(*ssa.FieldAddr); ok && structBaseEscapes(fa.X) {
					checkSafeArenaEscape(pass, v.Val, effectivePos(v.Val, storesTo, v.Pos()), getResults, storesTo, "struct field")
				}
			}
		}
	}

	// Third pass: check for SafeArena values captured by closures or goroutines.
	checkClosureAndGoroutineEscapes(pass, fn, getResults, storesTo)
}

// checkClosureAndGoroutineEscapes detects SafeArena values captured by closures
// that escape the enclosing function via return or goroutine launch. Such captures
// are unsafe because the arena may be freed before the closure runs.
//
// Detected patterns:
//   - return func() { ... uses p ... }  where p is Ptr[T] or Slice[T]
//   - go func() { ... uses p ... }()    where p is Ptr[T] or Slice[T]
//   - Same patterns for raw *T / []T results from .Get()
func checkClosureAndGoroutineEscapes(pass *analysis.Pass, fn *ssa.Function, getResults map[ssa.Value]bool, storesTo map[ssa.Value]ssa.Value) {
	type closureEscape struct {
		mc   *ssa.MakeClosure
		kind string
		pos  token.Pos
	}

	seenMC := make(map[*ssa.MakeClosure]bool)
	var escapingClosures []closureEscape

	addEscape := func(mc *ssa.MakeClosure, kind string, pos token.Pos) {
		if seenMC[mc] {
			return
		}
		seenMC[mc] = true
		escapingClosures = append(escapingClosures, closureEscape{mc, kind, pos})
	}

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch v := instr.(type) {
			case *ssa.Go:
				// go func() { captures p }()
				if mc, ok := v.Call.Value.(*ssa.MakeClosure); ok {
					addEscape(mc, "goroutine launch", v.Pos())
				}
			case *ssa.Return:
				for _, r := range v.Results {
					if mc, ok := canonicalSource(r, storesTo).(*ssa.MakeClosure); ok {
						// Use the Return instruction's position (mc.Pos() is 0 for closures in SSA).
						addEscape(mc, "closure return", v.Pos())
					}
				}
			case *ssa.Store:
				if isGlobalVar(v.Addr) {
					if mc, ok := canonicalSource(v.Val, storesTo).(*ssa.MakeClosure); ok {
						addEscape(mc, "global variable", v.Pos())
					}
				}
			}
		}
	}

	// Per the SSA spec: Bindings[i].Type() == *FreeVars[i].Type()
	// FreeVars gives us the actual captured type (e.g. Ptr[int]);
	// Bindings gives us the address of the capture cell (e.g. *Ptr[int]).
	// We use FreeVars for type-based checks and storesTo[Binding] for .Get() tracing.
	reportedBindings := make(map[ssa.Value]bool)
	for _, esc := range escapingClosures {
		innerFn, ok := esc.mc.Fn.(*ssa.Function)
		if !ok {
			continue
		}
		pos := esc.pos
		if !pos.IsValid() {
			pos = esc.mc.Pos()
		}
		if !pos.IsValid() {
			continue
		}
		for i, fv := range innerFn.FreeVars {
			if i >= len(esc.mc.Bindings) {
				break
			}
			binding := esc.mc.Bindings[i]
			if reportedBindings[binding] {
				continue
			}
			// Go SSA represents captured variables as pointers to their heap-allocated
			// capture cells: FreeVar.Type() == *ActualCapturedType. Unwrap one level.
			fvElemType := fv.Type()
			if ptr, ok := fvElemType.(*types.Pointer); ok {
				fvElemType = ptr.Elem()
			}
			// Check if the captured variable is a SafeArena wrapper type.
			if name := safeArenaWrapperTypeName(fvElemType); name != "" {
				pass.Reportf(pos, "safearena.%s captured by %s; use Deref() or Clone() before creating the closure",
					name, esc.kind)
				reportedBindings[binding] = true
				continue
			}
			// Check if the captured variable holds a .Get() result.
			// binding is the address of the capture cell; storesTo[binding] is what was stored.
			if content, found := storesTo[binding]; found {
				visited := make(map[ssa.Value]bool)
				if tracesBackToGet(content, getResults, storesTo, visited) {
					pass.Reportf(pos, "raw pointer from safearena .Get() captured by %s; arena may be freed before closure runs",
						esc.kind)
					reportedBindings[binding] = true
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

	// Check if the value is a Ptr[T] or Slice[T] wrapped in interface{}.
	// e.g. return interface{}(p) where p is Ptr[int] — the MakeInterface
	// instruction hides the underlying type from the first check above.
	// Use canonicalSource to handle defer-induced return-variable indirection
	// (where val is a load *rv and the MakeInterface is the stored value).
	if mi, ok := canonicalSource(val, storesTo).(*ssa.MakeInterface); ok {
		if name := safeArenaWrapperTypeName(mi.X.Type()); name != "" {
			pass.Reportf(pos, "safearena.%s escapes via %s; use Deref() or Clone() to copy value to heap",
				name, kind)
			return
		}
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
