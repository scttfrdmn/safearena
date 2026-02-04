# Security Policy

## Supported Versions

SafeArena is currently in experimental stage (v0.x). Security updates will be provided for:

| Version | Supported          |
| ------- | ------------------ |
| 0.4.x   | :white_check_mark: |
| < 0.4   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in SafeArena, please report it responsibly:

### How to Report

**Email:** Create a GitHub security advisory at https://github.com/scttfrdmn/safearena/security/advisories/new

**Please include:**
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### What to Expect

- **Acknowledgment** within 48 hours
- **Initial assessment** within 1 week
- **Status updates** as investigation progresses
- **Credit** in release notes (if desired)

### Security Considerations

SafeArena is built to **prevent memory safety issues**:

- Use-after-free detection
- Double-free prevention
- Allocation-after-free checks

However, please note:

⚠️ **Experimental Status**: The underlying Go arena package is experimental and not recommended for production use.

⚠️ **Runtime Checks**: SafeArena uses runtime checks, not compile-time guarantees like Rust.

⚠️ **Static Analyzer**: The arenacheck tool provides additional compile-time detection but is not exhaustive.

### Out of Scope

- Issues in Go's arena package itself (report to Go team)
- General Go bugs (report to Go team)
- Performance issues (create regular GitHub issue)
- Feature requests (create regular GitHub issue)

## Best Practices

When using SafeArena:

1. Always use `Scoped()` pattern when possible
2. Never return arena pointers from functions
3. Run arenacheck during development
4. Test with race detector enabled
5. Keep arenas short-lived (request/frame scoped)

## Disclosure Policy

- Confirmed vulnerabilities will be disclosed publicly after a fix is released
- Security advisories will be published on GitHub
- Critical issues will be announced prominently in release notes

---

Thank you for helping keep SafeArena safe and secure!
