# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Better error messages with suggestions
- Performance optimizations
- CI/CD pipeline
- More comprehensive examples
- Production readiness improvements

## [0.2.0] - 2026-02-03

### Added - arenacheck improvements
- **Direct return detection**: Now catches `return arena.New[T](a)` patterns
- **Use-after-free detection**: Detects usage of allocations after `arena.Free()`
- **Store/load tracking**: Traces allocations through local variable assignments
- Comprehensive test suite with 7 test cases
- Detailed results documentation (ARENACHECK_V02_RESULTS.md)

### Fixed
- **False positives**: Type checking prevents flagging safe value returns (int, string, etc.)
- Improved SSA value tracing through UnOp, FieldAddr, IndexAddr operations

### Changed
- Detection rate improved from 25% to 100% (4/4 patterns)
- Zero false positives in test suite
- More accurate error messages with allocation source locations

### Technical
- Rewrote analyzer with two-pass approach
- Added `findAllocation()` for recursive value tracing
- Added `checkUseAfterFree()` for post-Free validation
- Better handling of deferred Free() calls

## [0.1.0] - 2026-02-03

### Added
- Initial release of SafeArena
- Runtime safety package with `Arena`, `Ptr[T]`, and `Slice[T]` types
- `Scoped()` pattern for automatic arena lifetime management
- `Alloc()`, `AllocSlice()`, and `Clone()` functions
- Use-after-free detection (panics on access)
- Double-free detection (panics on second free)
- Lifetime tracking with arena IDs
- `arenacheck` static analyzer tool
  - SSA-based analysis
  - Detection of arena escapes to globals
  - Detection of arena escapes via returns
  - Integration with `go vet`
- Comprehensive test suite
  - 6 unit tests covering all safety features
  - Benchmark suite comparing SafeArena vs raw GC
  - Realistic workload benchmarks
- Documentation
  - README with quick start and examples
  - CREATIVE_SOLUTION.md explaining the design
  - ARENACHECK_RESULTS.md with analyzer evaluation
- Examples
  - Go arena comparison example
  - Rust equivalent for reference

### Known Limitations
- Requires `GOEXPERIMENT=arenas` (Go 1.20+)
- arenacheck has ~30-40% detection rate (proof-of-concept)
- Small runtime overhead (~13% vs raw arenas)
- Not production-ready (experimental arena package)

[Unreleased]: https://github.com/scttfrdmn/safearena/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/scttfrdmn/safearena/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/safearena/releases/tag/v0.1.0
