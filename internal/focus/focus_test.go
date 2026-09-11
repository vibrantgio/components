package focus_test

import (
	"testing"

	"gioui.org/unit"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/theme/tokens"
)

// Every control in this library draws one ring, and the ring is the
// platform's own keyboard focus indicator. A control that reached for
// anything else would teach a keyboard user a second appearance for one
// state.
func TestRingIsThePlatformsFocusIndicator(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		if got, want := focus.Ring(sc.p), sc.p.KeyboardFocusIndicator; got != want {
			t.Errorf("%s: Ring = %v, want the platform's keyboard focus indicator %v", sc.name, got, want)
		}
	}
}

// The indicator carries a coverage of its own, which is what lets one value
// read on every fill a control can put under it. A ring the platform made
// opaque would have to be measured against each of them instead.
func TestTheRingCarriesTheCoverageThePlatformGaveIt(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		if a := focus.Ring(p).A; a == 0 || a == 0xff {
			t.Errorf("Ring alpha = %d, want the platform's partial coverage", a)
		}
	}
}

// The ring is a keyboard affordance rather than an ornament, so it does not
// thin out when the controls around it tighten: one width at every density.
func TestWidthIsOneValue(t *testing.T) {
	if focus.Width != unit.Dp(2) {
		t.Errorf("Width = %v, want 2 dp", focus.Width)
	}
}
