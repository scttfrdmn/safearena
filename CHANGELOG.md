# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- More comprehensive examples (parser, game engine)
- Interprocedural analysis for arenacheck
- Production readiness improvements

## [0.3.0] - 2026-02-03

### Added
- **CI/CD Pipeline**: GitHub Actions for automated testing and releases
  - Tests on Linux, macOS, Windows
  - Multiple Go versions (1.23, 1.24, 1.25)
  - Automated benchmarks
  - Coverage reporting
  - Multi-platform binary builds
- **HTTP Server Example**: Request-scoped arena allocation pattern
- **Improved Error Messages**: Stack traces and helpful hints
  - Shows file:line location of errors
  - Actionable suggestions (e.g., "Use Clone() to copy to heap")
  - Emoji indicators for better visibility

### Changed - Performance Optimizations
- **9.6x faster** allocations (1,167 ns vs 11,167 ns)
- **5.6x less memory** per pointer (64 B vs 359 B)
- **3x fewer allocations** (2 vs 6 per operation)
- Removed unused `sync.Map` tracking
- Removed redundant `arenaID` field from `Ptr[T]`
- Streamlined struct layouts for better cache locality

### Technical
- `errors.go`: Stack capture and hint system
- Optimized hot paths in `Alloc()` and `Get()`
- CI workflows for continuous integration and release automation

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

[Unreleased]: https://github.com/scttfrdmn/safearena/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/scttfrdmn/safearena/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/scttfrdmn/safearena/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/safearena/releases/tag/v0.1.0
