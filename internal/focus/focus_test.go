package focus_test

import (
	"testing"

	"gioui.org/unit"

	"github.com/vibrantgio/components/internal/focus"
	vgcolor "github.com/vibrantgio/theme/color"
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
		beneath := sc.p.WindowBackground
		want := vgcolor.Flatten(sc.p.KeyboardFocusIndicator, beneath)
		if got := focus.Ring(sc.p, beneath); got != want {
			t.Errorf("%s: Ring = %v, want the platform's keyboard focus indicator over the plane, %v", sc.name, got, want)
		}
	}
}

// The indicator carries a coverage of its own, which is what lets one value
// read on every fill a control can put under it — and what the ring hands
// back is that coverage resolved against one of them, opaque, because the
// platform composites it in encoded sRGB and Gio's rasterizer would not.
func TestTheRingResolvesTheCoverageAgainstWhatIsBeneathIt(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		if p.KeyboardFocusIndicator.A == 0 || p.KeyboardFocusIndicator.A == 0xff {
			t.Errorf("keyboardFocusIndicatorColor alpha = %d, want the platform's partial coverage", p.KeyboardFocusIndicator.A)
		}
		if a := focus.Ring(p, p.WindowBackground).A; a != 0xff {
			t.Errorf("Ring alpha = %d, want an opaque answer", a)
		}
		if onPlane, onButton := focus.Ring(p, p.WindowBackground), focus.Ring(p, p.PushButtonFill); onPlane == onButton {
			t.Errorf("the halo reads the same on the plane and on a push button's fill (%v); the coverage is not being resolved", onPlane)
		}
	}
}

// The halo is a keyboard affordance rather than an ornament, so it does not
// thin out when the controls around it tighten: one width at every density.
//
// MEASURED, save-dialog-{light,dark}.png, the focused "Save As:" field: four
// px on every side of a box running x 264-495, covering x 262-265 and x
// 494-497 across and y 205-208 and y 231-234 down, with the sheet unblended
// one px beyond each. Half of that lies past the box, which is what Outside
// carries.
func TestWidthIsTheMeasuredBand(t *testing.T) {
	if focus.Width != unit.Dp(4) {
		t.Errorf("Width = %v, want 4 dp", focus.Width)
	}
	if focus.Outside != focus.Width/2 {
		t.Errorf("Outside = %v, want half of Width (%v)", focus.Outside, focus.Width/2)
	}
}
