# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Improved arenacheck detection rates
- Better error messages with suggestions
- Performance optimizations
- Production readiness improvements

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

[Unreleased]: https://github.com/scttfrdmn/safearena/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/scttfrdmn/safearena/releases/tag/v0.1.0
