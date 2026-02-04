package safearena

import (
	"testing"
)

// FuzzAlloc tests Alloc with various input types and sizes
func FuzzAlloc(f *testing.F) {
	// Seed corpus
	f.Add(int(0))
	f.Add(int(42))
	f.Add(int(-1))
	f.Add(int(1000000))

	f.Fuzz(func(t *testing.T, val int) {
		result := Scoped(func(a *Arena) int {
			p := Alloc(a, val)
			return p.Deref()
		})

		if result != val {
			t.Errorf("expected %d, got %d", val, result)
		}
	})
}

// FuzzAllocSlice tests AllocSlice with various sizes
func FuzzAllocSlice(f *testing.F) {
	// Seed corpus
	f.Add(0)
	f.Add(1)
	f.Add(100)
	f.Add(1024)

	f.Fuzz(func(t *testing.T, size int) {
		// Skip negative sizes and unreasonably large sizes
		if size < 0 || size > 10000000 {
			return
		}

		result := Scoped(func(a *Arena) int {
			s := AllocSlice[byte](a, size)
			slice := s.Get()
			return len(slice)
		})

		if result != size {
			t.Errorf("expected size %d, got %d", size, result)
		}
	})
}

// FuzzStringBuilder tests StringBuilder with various inputs
func FuzzStringBuilder(f *testing.F) {
	// Seed corpus
	f.Add("hello", "world")
	f.Add("", "test")
	f.Add("a", "b")
	f.Add("longer string with spaces", "and another one")

	f.Fuzz(func(t *testing.T, s1, s2 string) {
		// Skip if total length would be too large
		if len(s1)+len(s2) > 10000 {
			return
		}

		result := Scoped(func(a *Arena) string {
			capacity := len(s1) + len(s2) + 100
			sb := NewStringBuilder(a, capacity)

			builder := sb.Get()
			builder.Append(s1)
			builder.Append(s2)

			return builder.String()
		})

		expected := s1 + s2
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// FuzzClone tests Clone with various data types
func FuzzClone(f *testing.F) {
	// Seed corpus
	f.Add("test string")
	f.Add("")
	f.Add("a very long string that might trigger different code paths")

	f.Fuzz(func(t *testing.T, val string) {
		a := New()

		p := Alloc(a, val)
		cloned := Clone(p)

		a.Free()

		if *cloned != val {
			t.Errorf("expected %q, got %q", val, *cloned)
		}
	})
}

// FuzzOptimized tests the optimized version with various inputs
func FuzzOptimized(f *testing.F) {
	// Seed corpus
	f.Add(int64(0))
	f.Add(int64(42))
	f.Add(int64(-100))
	f.Add(int64(999999))

	f.Fuzz(func(t *testing.T, val int64) {
		result := ScopedOpt(func(a *ArenaOpt) int64 {
			p := AllocOpt(a, val)
			return p.Deref()
		})

		if result != val {
			t.Errorf("expected %d, got %d", val, result)
		}
	})
}
