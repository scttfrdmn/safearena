# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Interprocedural analysis for arenacheck
- Production readiness improvements

## [0.4.0] - 2026-02-03

### Added - Documentation & Polish
- **Comprehensive API Documentation**
  - `doc.go` with package-level overview and usage patterns
  - `example_test.go` with 10 runnable godoc examples
  - Enhanced inline documentation for all exported functions
  - Each function includes usage examples and panic conditions
  - Ready for pkg.go.dev publication

- **Test Coverage 96.9%** (improved from 56.2%)
  - `safearena_coverage_test.go` with 29 comprehensive tests
  - Full coverage of optimized version (all Opt variants)
  - Error path coverage (use-after-free, double-free, alloc-after-free)
  - Edge case tests (large allocations, concurrent usage, complex structs)
  - Race detection tests (no data races found)

- **Fuzz Testing**
  - `safearena_fuzz_test.go` with 5 fuzz tests
  - Over 5 million random executions total
  - Tests: FuzzAlloc, FuzzAllocSlice, FuzzStringBuilder, FuzzClone, FuzzOptimized
  - Zero failures across all fuzz tests

- **Real-World Examples**
  - **JSON Parser** (`examples/json_parser/`) - Arena-allocated AST pattern
    - Performance: ~1.4x faster for parse-process patterns
    - Shows temporary parse trees with final results on heap
  - **Database Query Processor** (`examples/database_processor/`) - Request-scoped processing
    - Performance: ~1.4x faster with lower GC pressure
    - Shows per-query buffers, filtering, aggregation
  - **Image Filter Pipeline** (`examples/image_filter/`) - Multi-pass large buffers
    - Performance: ~1.5x faster (scales with image size)
    - Shows working with MB-sized temp buffers
  - Master `examples/README.md` with patterns, anti-patterns, and guidelines

- **Community Documentation**
  - `CONTRIBUTING.md` - Comprehensive 400+ line contribution guide
    - Development setup, testing, code quality
    - Coding standards and documentation guidelines
    - Commit conventions and PR process
  - `CODE_OF_CONDUCT.md` - Community guidelines
  - `SECURITY.md` - Vulnerability reporting and security policy

- **Badges and Quality**
  - Go Report Card badge (A+ ready)
  - Test coverage badge (96.9%)
  - Updated README with contributing section

### Changed
- Enhanced README Contributing section with links to all community docs
- Updated `.gitignore` for coverage files and example binaries

### Technical
- All examples include benchmarks and detailed READMEs
- Examples show real performance numbers and use cases
- Coverage HTML report generation
- Fuzz corpus seeding for comprehensive testing

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

[Unreleased]: https://github.com/scttfrdmn/safearena/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/scttfrdmn/safearena/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/scttfrdmn/safearena/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/scttfrdmn/safearena/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/safearena/releases/tag/v0.1.0
