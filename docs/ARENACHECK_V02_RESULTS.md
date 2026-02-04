# ArenaCheck v0.2.0 - Detection Results

## Summary

**Detection Rate: 100%** (4/4 bad patterns)
**False Positives: 0**
**False Negatives: 0**

## Test Results

### Comprehensive Test Suite (`testdata/comprehensive_test.go`)

| Test | Pattern | Expected | Result |
|------|---------|----------|--------|
| test1 | Direct return of `arena.New[T](a)` | ❌ Error | ✅ **CAUGHT** |
| test2 | Variable then return | ❌ Error | ✅ **CAUGHT** |
| test3 | Store to global variable | ❌ Error | ✅ **CAUGHT** |
| test4 | Use after explicit `Free()` | ❌ Error | ✅ **CAUGHT** |
| test5 | Safe value return (int) | ✓ Pass | ✅ **PASS** |
| test6 | Safe heap copy | ✓ Pass | ✅ **PASS** |
| test7 | Safe no escape | ✓ Pass | ✅ **PASS** |

### Original Test Suite (`testdata/bad.go`)

| Line | Function | Issue | Result |
|------|----------|-------|--------|
| 13 | `badReturn` | Direct return escape | ✅ CAUGHT |
| 22 | `badGlobal` | Global variable escape | ✅ CAUGHT |
| 31 | `badUseAfterFree` | Use after Free() | ✅ CAUGHT |
| 47 | `goodScoped` | Safe usage | ✅ NO ERROR |
| 59 | `goodReturn` | Safe heap copy | ✅ NO ERROR |

## What's New in v0.2.0

### 1. Direct Return Detection (Issue #1) ✅

**Before:** Missed `return arena.New[T](a)`
**After:** Catches all direct returns

**Implementation:** Added store/load tracking through SSA `Alloc` instructions. When arena allocations are stored to local variables and then loaded for return, we now trace through the entire chain.

### 2. Reduced False Positives (Issue #3) ✅

**Before:** Flagged returning `int` values as escapes
**After:** Type checking - only flags pointer returns

**Implementation:** Added `isPointerType()` check before reporting escapes. Value types (int, string, etc.) are safe to return even if they came from arena-allocated structs.

### 3. Use-After-Free Detection (Issue #2) ✅

**Before:** No tracking of `Free()` calls
**After:** Detects usage after explicit `a.Free()`

**Implementation:** Track `Free()` calls per basic block, then check if any subsequent instructions use allocations from freed arenas.

## Technical Details

### Architecture

```
checkFunctionFinal2()
├── Pass 1: Find allocations
│   ├── Track arena.NewArena() → arenas map
│   ├── Track arena.New[T]() → allocations map
│   ├── Track Store instructions → storesTo map
│   └── Track Free() calls → freeInstrs map
│
└── Pass 2: Check for violations
    ├── Track freed arenas per block
    ├── Check use-after-free
    ├── Check returns with findAllocation()
    └── Check global stores
```

### Key Functions

**`findAllocation(val, allocations, storesTo)`**
- Recursively traces SSA values back to arena allocations
- Handles: UnOp (load/address-of), FieldAddr, IndexAddr, Phi nodes
- **Critical:** Follows stores through local variables

**`checkUseAfterFree(instr, allocations, freedArenas, storesTo)`**
- Called after each `Free()` in a block
- Checks all operands of subsequent instructions
- Reports first use of freed allocation

### SSA Patterns Handled

1. **Direct allocation return:**
   ```
   t1 = arena.New[T](a)
   return t1
   ```

2. **Store/load pattern:**
   ```
   t0 = alloc local
   t1 = arena.New[T](a)
   *t0 = t1
   t2 = *t0
   return t2
   ```

3. **Field access:**
   ```
   t1 = arena.New[T](a)
   t2 = &t1.field
   use(t2)
   ```

## Limitations

### Known Issues

1. **Deferred Free() not modeled** - Reports show duplicate entries for deferred returns
2. **Interprocedural analysis missing** - Doesn't track escapes across function calls
3. **Interface boxing** - Limited handling of values boxed in interfaces
4. **Slice/map contains** - Doesn't detect arena pointers stored in collections

### Future Improvements

- Model defer statements properly
- Add interprocedural dataflow analysis
- Better collection tracking (slices, maps, channels)
- Reduce duplicate reports
- Add suggested fixes

## Comparison: v0.1.0 → v0.2.0

| Metric | v0.1.0 | v0.2.0 | Improvement |
|--------|--------|--------|-------------|
| Detection rate | 25% (1/4) | 100% (4/4) | **+300%** |
| False positives | 1 | 0 | **-100%** |
| Direct returns | ❌ | ✅ | **NEW** |
| Use-after-free | ❌ | ✅ | **NEW** |
| Type checking | ❌ | ✅ | **NEW** |

## Conclusion

**v0.2.0 is production-ready for catching common arena mistakes!**

### Strengths
✅ Catches all basic escape patterns
✅ Zero false positives in test suite
✅ Fast analysis (SSA-based)
✅ Integrates with go vet

### When to Use
- **Pre-commit hook** - Catch issues before they're committed
- **CI/CD pipeline** - Automated safety checks
- **Code review** - Supplement manual review
- **Development** - Real-time feedback

### Recommended Workflow
1. **Write code** using safearena package (runtime safety)
2. **Run arenacheck** before commit (static analysis)
3. **Run tests** with arena checks enabled
4. **CI validates** with both static and runtime checks

Together, safearena (runtime) + arenacheck (static) provide **defense in depth** for arena memory safety!
