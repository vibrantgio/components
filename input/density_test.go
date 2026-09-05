package input_test

import (
	"context"
	"testing"

	"gioui.org/layout"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// densityTheme returns a theme whose density is d, with sharp corners for
// golden determinism (anti-aliased rounded corners vary between GPU context
// initialisations).
func densityTheme(d tokens.Density) theme.Theme {
	th := theme.Default()
	th.Density = rx.Of(d)
	th.Radius = rx.Of(tokens.RadiusScale{})
	return th
}

// materialize subscribes to a component observable and returns the last
// layout.Widget it emitted.
func materialize(t *testing.T, obs rx.Observable[layout.Widget]) layout.Widget {
	t.Helper()
	var w layout.Widget
	if err := obs.Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("component did not emit a layout.Widget")
	}
	return w
}
