# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- awesome-go listing (PR #6026 submitted, pending maintainer review)

## [0.5.3] - 2026-02-26

### Added
- **arenacheck: struct field escape detection** (closes #36)
  - Detects `Ptr[T]`/`Slice[T]` stored into fields of escaping structs
  - Flags struct pointers that are function parameters (method receivers, pointer args)
    and heap-allocated structs (`new(T)` / `&T{}`)
  - Also detects raw `*T` / `[]T` from `.Get()` stored into escaping struct fields
  - Stack-local structs (`var s T`) are intentionally not flagged
  - 4 new bad-pattern test cases, 1 new good-pattern test case

## [0.5.2] - 2026-02-26

### Fixed
- Release workflow: removed Windows binary — `GOEXPERIMENT=arenas` is not supported on
  Windows; Linux and macOS binaries are unaffected (closes #34)

### Documentation
- `PERFORMANCE.md`: new "Go Escape Analysis and Arena Allocations" section covering
  `AllocSlice` heap-backing limitation, interface boxing escapes, closure captures, and
  how to use `-gcflags=-m` to verify arena allocations (closes #35)

## [0.5.1] - 2026-02-26

### Fixed
- `NewWithFinalizer`: leak warning now written to `stderr` instead of `stdout`
- `AllocSlice`: panics with descriptive `"negative size N"` message on negative input;
  previously delegated to the Go runtime with a cryptic message
- Added `TestAllocSliceNegativeSize` test

### Documentation
- `Ptr[T]` godoc: documents memory retention behaviour — the Arena control struct
  (~64 bytes) stays alive while any `Ptr[T]` is reachable, but the allocation pool is
  reclaimed on `Free()`/`Reset()`
- `README`: added Thread Safety section with per-goroutine arena pattern examples
- `README`: added arenacheck known limitations to Static Analysis section

## [0.5.0] - 2026-02-26

### Added
- **arenacheck: closure and goroutine escape detection** (closes #31)
  - Detects `Ptr[T]`/`Slice[T]` values captured by returned closures
  - Detects `Ptr[T]`/`Slice[T]` values captured by goroutine launches
  - Detects `.Get()` results captured by either pattern
  - New test cases in `testdata/src/satest/patterns.go`
- **Arena allocation statistics** — opt-in via `NewWithStats()` (closes #32)
  - `ArenaStats` struct with `AllocCount` field
  - `(a *Arena) Stats() ArenaStats` method
  - Zero overhead on normal arenas (nil pointer check)
- **Pool statistics** — always available via atomic counters (closes #32)
  - `PoolStats` struct: `Gets`, `Puts`, `Created`, `Reused`
  - `(p *Pool) Stats() PoolStats` method
  - `ExampleNewWithStats` and `ExamplePool_Stats` godoc examples

### Fixed
- `TestPoolStats` and `ExamplePool_Stats`: assert `Created+Reused == Gets` invariant
  rather than specific split — `sync.Pool` is non-deterministic under GC pressure

## [0.4.7] - 2026-02-25

### Documentation
- Added `docs/CI_INTEGRATION.md`: comprehensive guide for running arenacheck in CI
  pipelines with `go vet -vettool`, GitHub Actions workflow example, golangci-lint
  compatibility note, and detection table (closes #33)
- Added `arenacheck` target to `Makefile`
- Linked CI guide from `README.md` and `cmd/arenacheck/README.md`

## [0.4.6] - 2026-02-25

### Added
- `Makefile` with targets: `build`, `test`, `test-race`, `vet`, `fmt`, `lint`,
  `arenacheck`, `check`, `install-tools`, `clean`, `help`
- 14 additional godoc examples in `example_test.go`: `ExampleArena_Reset`,
  `ExamplePool`, `ExampleNewStringBuilder`, `ExampleNewWithFinalizer`, and more

## [0.4.5] - 2026-02-25

### Documentation
- Clarified arenacheck checker scope and detection rate in docs and FAQ
- Updated known limitations section

## [0.4.4] - 2026-02-25

### Fixed
- Audit round 2: Pool safety documentation, arenacheck `interface{}` escape detection,
  `captureStack` fallback handling, `Deref` docstring corrections

## [0.4.3] - 2026-02-25

### Fixed
- Release workflow: use `go test .` (not `./...`) to avoid coverage tool errors on
  example `main` packages; `cd` into `cmd/arenacheck` before building

## [0.4.2] - 2026-02-25

### Fixed
- Wrong `AllocSlice` doc comment (said arena-allocated, backing array is heap-allocated)
- `StringBuilder` overflow error message format
- Added missing `ExampleSlice_Get` godoc example
- `gofmt -s` alignment in `errors.go` const block

## [0.4.1] - 2026-02-25

### Added
- `Arena.Reset()`: frees all allocations and prepares arena for reuse; existing
  `Ptr[T]`/`Slice[T]` values panic with `"use after reset"` on access
- `Pool`: thread-safe arena pool via `sync.Pool` for high-throughput workloads;
  `Pool.Put` resets and returns the arena, `Pool.Get` retrieves or creates one
- arenacheck: detects unsafe usage of `safearena.Ptr[T]` and `Slice[T]` — direct
  returns, global stores, `.Get()` result escapes, `interface{}` wrapping (closes #28)
- `ScopedVoid` (replaces deprecated `ScopedPtr`) for fire-and-forget arena scopes

### Changed
- Removed `ArenaOpt`/`PtrOpt` tier — consolidated into single API (closes #27)
- License changed from MIT to Apache 2.0
- CI: added `gofmt -s` check; Go Report Card A+ required

### Fixed
- Security fixes from senior engineer review (input validation, error message
  consistency, `captureStack` correctness)
- CI: excluded example `main` packages from coverage instrumentation
- CI: resolved `staticcheck` SA4010 false positive; removed unsupported Windows matrix

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

[Unreleased]: https://github.com/scttfrdmn/safearena/compare/v0.5.3...HEAD
[0.5.3]: https://github.com/scttfrdmn/safearena/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/scttfrdmn/safearena/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/scttfrdmn/safearena/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/scttfrdmn/safearena/compare/v0.4.7...v0.5.0
[0.4.7]: https://github.com/scttfrdmn/safearena/compare/v0.4.6...v0.4.7
[0.4.6]: https://github.com/scttfrdmn/safearena/compare/v0.4.5...v0.4.6
[0.4.5]: https://github.com/scttfrdmn/safearena/compare/v0.4.4...v0.4.5
[0.4.4]: https://github.com/scttfrdmn/safearena/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/scttfrdmn/safearena/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/scttfrdmn/safearena/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/scttfrdmn/safearena/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/scttfrdmn/safearena/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/scttfrdmn/safearena/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/scttfrdmn/safearena/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/safearena/releases/tag/v0.1.0
