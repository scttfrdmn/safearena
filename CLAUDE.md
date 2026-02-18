# safearena — Claude Code Instructions

## Project Overview
SafeArena is a safe, ergonomic wrapper around Go's experimental `arena` package.
It provides runtime use-after-free and double-free detection via typed wrappers (`Ptr[T]`, `Slice[T]`)
and a static analyzer (`cmd/arenacheck`).

All code requires `GOEXPERIMENT=arenas` to build and test.

## Build & Test

```bash
GOEXPERIMENT=arenas go build ./...
GOEXPERIMENT=arenas go test .               # run all tests (root package only)
GOEXPERIMENT=arenas go test -race .         # with race detector
GOEXPERIMENT=arenas go vet ./...
staticcheck ./...                           # lint (requires: go install honnef.co/go/tools/cmd/staticcheck@latest)
gofmt -l .                                  # must return empty
```

## Project Tracking — GitHub is the source of truth

All work is tracked via GitHub. **Do not create status documents, session summaries, or
planning files in the repo.** Use GitHub exclusively:

- **Issues** → bugs, features, tasks (`gh issue list`)
- **Milestones** → release targets (`gh milestone list`)
- **Labels** → categorize issues (type:, priority:, area:, status:)
- **Projects** → "SafeArena Roadmap" at https://github.com/users/scttfrdmn/projects/23

When starting work on a tracked item:
1. Check open issues: `gh issue list --state open`
2. Assign/update the relevant issue
3. Close the issue when the work is committed and CI passes

## Label Conventions

| Prefix     | Examples                                              |
|------------|-------------------------------------------------------|
| `type:`    | bug, feature, enhancement, documentation, performance |
| `priority:`| high, medium, low                                     |
| `area:`    | runtime, analyzer, tests, docs                        |
| `status:`  | blocked, help wanted, good first issue                |

## Architecture

```
safearena.go          — primary API: Arena, Ptr[T], Slice[T], Scoped, Clone
safearena_optimized.go — ArenaOpt / PtrOpt variants with minimal error detail
errors.go             — panic message formatting with stack traces
cmd/arenacheck/       — separate Go module; SSA-based static analyzer
examples/             — standalone main packages; no tests
docs/                 — guides (MIGRATION.md, FAQ.md, PERFORMANCE.md)
```

## Key API Rules
- `Scoped[R]` is the preferred pattern — arena is freed automatically via defer
- `ScopedVoid` (formerly `ScopedPtr`) for fire-and-forget scopes with no return value
- `ScopedPtr` exists as a deprecated alias for `ScopedVoid` — do not add new uses
- Never return `Ptr[T]` or `Slice[T]` from a `Scoped` callback — they become invalid
- `AllocSlice` backing arrays are heap-allocated (Go arena API limitation) — document this at call sites
- All errors panic (invariant violations); use `recover()` in tests

## CI Matrix
- **OS**: ubuntu-latest, macos-latest (Windows excluded — GOEXPERIMENT=arenas unsupported)
- **Go**: 1.23, 1.24, 1.25
- Tests run with `go test . ` (not `./...`) to avoid coverage tool errors on example main packages

## Versioning
The project is pre-1.0. Use `v0.x.y` tags. Do not tag `v1.0.0` without explicit user approval.
