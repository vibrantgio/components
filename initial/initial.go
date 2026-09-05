// Package initial provides Value[T], a zero-value sentinel for first-frame
// initialisation inside rx.Defer closures.
//
// # Problem
//
// Inside an rx.Defer closure, state is allocated once per subscription and
// then captured by all downstream map functions and layout.Widget closures. State
// that depends on layout dimensions (scroll position, viewport offsets) cannot
// be computed until the first Gio frame, because the layout context is not
// available at subscription time.
//
// The ad-hoc solution is a magic sentinel:
//
//	offset := struct{ X, Y unit.Dp }{X: -1}
//	// ...
//	if offset.X < 0 {
//	    offset.X = computeFromLayout(gtx)
//	}
//
// The sentinel is invisible to the type system, chosen arbitrarily, and
// repeated across every pane that needs first-frame initialisation.
//
// # Solution
//
// Value[T] wraps a T together with a boolean "has been set" flag.
// The zero value of Value[T] is explicitly unset — no sentinel value required.
//
//	var offsetX initial.Value[unit.Dp]
//	// ...
//	offsetX.GetOrSet(func() unit.Dp { return computeFromLayout(gtx) })
//
// # Threading
//
// Value is not safe for concurrent use. All calls must occur on the same
// goroutine — typically the Gio frame goroutine inside a layout.Widget closure.
package initial

// Value[T] is an optional T that distinguishes "never set" from "set to
// the zero value of T". The zero value of Value[T] is unset.
type Value[T any] struct {
	v  T
	ok bool
}

// IsSet reports whether the value has been set via Set or GetOrSet.
func (v *Value[T]) IsSet() bool { return v.ok }

// Set stores val and marks the value as set.
func (v *Value[T]) Set(val T) { v.v = val; v.ok = true }

// Get returns the stored value. It panics if the value has not been set.
// Pair with IsSet when the caller cannot guarantee prior initialisation.
func (v *Value[T]) Get() T {
	if !v.ok {
		panic("initial.Value: Get called before Set")
	}
	return v.v
}

// GetOrSet returns the stored value if already set; otherwise it calls fn,
// stores the result, marks the value as set, and returns the result.
// fn is called at most once — on the first frame.
func (v *Value[T]) GetOrSet(fn func() T) T {
	if !v.ok {
		v.v = fn()
		v.ok = true
	}
	return v.v
}
