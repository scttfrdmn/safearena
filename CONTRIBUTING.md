# Contributing to SafeArena

Thank you for your interest in contributing to SafeArena! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Go version** and platform (OS, architecture)
- **Code sample** demonstrating the issue
- **GOEXPERIMENT=arenas** - confirm you're using experimental arenas

**Example:**
```markdown
## Bug: Use-after-free not detected in X scenario

**Go Version:** 1.23.1
**Platform:** Linux amd64
**GOEXPERIMENT:** arenas

**Steps to reproduce:**
1. Create arena with `New()`
2. Allocate data
3. ... (detailed steps)

**Expected:** Panic with error message
**Actual:** Silent corruption
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. Include:

- **Clear use case** - What problem does this solve?
- **Proposed solution** - How would it work?
- **Alternatives considered** - What other approaches did you consider?
- **Examples** - Show how the API would be used

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following our coding standards
3. **Add tests** for new functionality
4. **Ensure tests pass** with `GOEXPERIMENT=arenas go test ./...`
5. **Update documentation** if you changed APIs
6. **Write clear commit messages** (see below)
7. **Submit the pull request**

## Development Setup

### Prerequisites

- **Go 1.23+** with arena support
- **Git** for version control
- **GitHub CLI** (optional, for `gh` commands)

### Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/safearena.git
cd safearena

# Add upstream remote
git remote add upstream https://github.com/scttfrdmn/safearena.git

# Install dependencies
GOEXPERIMENT=arenas go mod download
```

### Running Tests

```bash
# Run all tests
GOEXPERIMENT=arenas go test -v ./...

# Run with coverage
GOEXPERIMENT=arenas go test -cover ./...

# Run with race detector
GOEXPERIMENT=arenas go test -race ./...

# Run specific test
GOEXPERIMENT=arenas go test -run TestName

# Run benchmarks
GOEXPERIMENT=arenas go test -bench=. -benchmem

# Run fuzz tests (short duration)
GOEXPERIMENT=arenas go test -fuzz=FuzzAlloc -fuzztime=10s
```

### Code Quality

```bash
# Format code
GOEXPERIMENT=arenas gofmt -w .

# Run linters
GOEXPERIMENT=arenas go vet ./...

# Run static analyzer
GOEXPERIMENT=arenas arenacheck ./...
```

## Coding Standards

### Go Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Keep functions focused and small
- Write clear, descriptive variable names

### Documentation

- **All exported functions** must have godoc comments
- **Include examples** for major functions
- **Explain panics** and error conditions
- **Show usage patterns** in comments

**Good documentation:**
```go
// Alloc allocates a value in the arena and returns a safe pointer.
// The returned Ptr[T] tracks the arena lifetime and will panic on use-after-free.
//
// Panics if the arena has already been freed.
//
// Example:
//
//	data := safearena.Alloc(a, MyStruct{Field: "value"})
//	ptr := data.Get() // Safe while arena is alive
func Alloc[T any](a *Arena, value T) Ptr[T]
```

### Testing

- **Test coverage** should be 90%+ for new code
- **Test edge cases** and error paths
- **Include benchmarks** for performance-critical code
- **Use table-driven tests** where appropriate
- **Test concurrent usage** with race detector

**Example test structure:**
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   int
        want    int
        wantErr bool
    }{
        {"positive", 5, 25, false},
        {"zero", 0, 0, false},
        {"negative", -5, 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Commit Messages

Use conventional commits format:

```
type(scope): brief description

Longer explanation if needed.

- Bullet points for details
- Reference issues with #123
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions or changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

**Examples:**
```
feat(arena): add SetFinalizer method for leak detection

Add optional finalizer support to detect arenas that are
garbage collected without being freed.

- NewWithFinalizer() creates arena with finalizer
- Warns on GC if not freed
- Useful for debugging, not production

Closes #42
```

## Project Structure

```
safearena/
├── safearena.go          # Core arena API
├── safearena_optimized.go # Optimized version
├── errors.go             # Error handling
├── doc.go                # Package documentation
├── *_test.go             # Tests
├── cmd/
│   └── arenacheck/       # Static analyzer
├── examples/             # Usage examples
└── docs/                 # Additional documentation
```

## Areas for Contribution

### High Priority
- Improving arenacheck detection rates
- More real-world examples
- Performance benchmarks
- Cross-platform testing

### Good First Issues
Look for issues labeled `good first issue` or `help wanted`.

### Documentation
- Improve godoc examples
- Add more use case documentation
- Tutorial blog posts
- Video tutorials

### Testing
- Edge case tests
- Fuzz test improvements
- Benchmark suite expansion
- Platform-specific tests

## Pull Request Process

1. **Update CHANGELOG.md** under "Unreleased" section
2. **Update documentation** for API changes
3. **Add tests** for new features
4. **Ensure CI passes** (tests, linting, formatting)
5. **Request review** from maintainers
6. **Address feedback** promptly
7. **Squash commits** if requested

### Review Process

- Maintainers will review within **1 week**
- Address feedback in comments
- Once approved, maintainer will merge
- Changes will be included in next release

## Release Process

SafeArena follows [Semantic Versioning](https://semver.org/):

- **Major (1.0.0)**: Breaking API changes
- **Minor (0.1.0)**: New features, backward compatible
- **Patch (0.0.1)**: Bug fixes, backward compatible

Releases include:
- Updated CHANGELOG.md
- Git tag (vX.Y.Z)
- GitHub release with notes
- Announcement (if major/minor)

## Questions?

- **GitHub Discussions** for questions and ideas
- **GitHub Issues** for bugs and features
- **Email** maintainer for private concerns

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to SafeArena! 🎉
