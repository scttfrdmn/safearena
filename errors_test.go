package safearena

import (
	"strings"
	"testing"
)

func TestImprovedErrorMessages(t *testing.T) {
	t.Run("use after free shows hint", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic")
			}

			msg := r.(string)
			if !strings.Contains(msg, "use after free") {
				t.Errorf("expected 'use after free', got: %s", msg)
			}
			if !strings.Contains(msg, "Hint:") {
				t.Errorf("expected hint, got: %s", msg)
			}
			if !strings.Contains(msg, "Clone()") {
				t.Errorf("expected mention of Clone(), got: %s", msg)
			}

			t.Logf("Good error message:\n%s", msg)
		}()

		a := New()
		p := Alloc(a, 42)
		a.Free()
		_ = p.Get() // Should panic with helpful message
	})

	t.Run("double free shows hint", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic")
			}

			msg := r.(string)
			if !strings.Contains(msg, "double free") {
				t.Errorf("expected 'double free', got: %s", msg)
			}
			if !strings.Contains(msg, "Hint:") {
				t.Errorf("expected hint, got: %s", msg)
			}

			t.Logf("Good error message:\n%s", msg)
		}()

		a := New()
		a.Free()
		a.Free() // Should panic with helpful message
	})

	t.Run("alloc after free shows hint", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic")
			}

			msg := r.(string)
			if !strings.Contains(msg, "allocation after free") {
				t.Errorf("expected 'allocation after free', got: %s", msg)
			}
			if !strings.Contains(msg, "Hint:") {
				t.Errorf("expected hint, got: %s", msg)
			}

			t.Logf("Good error message:\n%s", msg)
		}()

		a := New()
		a.Free()
		_ = Alloc(a, 42) // Should panic with helpful message
	})
}
