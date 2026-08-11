package initial_test

import (
	"testing"

	"github.com/vibrantgio/components/initial"
)

func TestValue_FirstFrame(t *testing.T) {
	t.Run("zero value is unset", func(t *testing.T) {
		var v initial.Value[int]
		if v.IsSet() {
			t.Fatal("zero Value should be unset")
		}
	})

	t.Run("GetOrSet initialises on first call", func(t *testing.T) {
		var v initial.Value[int]
		calls := 0
		got := v.GetOrSet(func() int { calls++; return 42 })
		if got != 42 {
			t.Fatalf("want 42, got %d", got)
		}
		if calls != 1 {
			t.Fatalf("fn should be called once, got %d", calls)
		}
		if !v.IsSet() {
			t.Fatal("value should be marked set after GetOrSet")
		}
	})

	t.Run("Set stores value and marks as set", func(t *testing.T) {
		var v initial.Value[string]
		v.Set("hello")
		if !v.IsSet() {
			t.Fatal("IsSet should be true after Set")
		}
		if got := v.Get(); got != "hello" {
			t.Fatalf("want %q, got %q", "hello", got)
		}
	})

	t.Run("Get panics when unset", func(t *testing.T) {
		var v initial.Value[float64]
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Get on unset Value should panic")
			}
		}()
		v.Get()
	})
}

func TestValue_SubsequentFrame(t *testing.T) {
	t.Run("GetOrSet returns existing value without calling fn again", func(t *testing.T) {
		var v initial.Value[int]
		calls := 0
		v.GetOrSet(func() int { calls++; return 99 })

		// Simulate many subsequent frames.
		for range 100 {
			got := v.GetOrSet(func() int { calls++; return 0 })
			if got != 99 {
				t.Fatalf("subsequent call: want 99, got %d", got)
			}
		}
		if calls != 1 {
			t.Fatalf("fn called %d times; want exactly 1", calls)
		}
	})

	t.Run("Set overwrites after initial GetOrSet", func(t *testing.T) {
		var v initial.Value[int]
		v.GetOrSet(func() int { return 1 })
		v.Set(2)
		if got := v.Get(); got != 2 {
			t.Fatalf("want 2 after Set, got %d", got)
		}
	})

	t.Run("IsSet remains true after Set", func(t *testing.T) {
		var v initial.Value[int]
		v.Set(0) // set to the zero value of int — should still be "set"
		if !v.IsSet() {
			t.Fatal("zero-value T after Set should still report IsSet=true")
		}
	})
}
