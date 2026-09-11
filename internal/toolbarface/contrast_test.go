package toolbarface

import (
	"image/color"
	"testing"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// The chrome-variant trigger is the one control in this library that tints
// under the pointer, and every colour it draws is a platform name. This is
// the table that says which.
func TestTheTriggerTakesThePlatformsNames(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := sc.p
		chrome := p.SidebarMaterial
		for _, tc := range []struct {
			state tokens.State
			want  color.NRGBA
		}{
			{tokens.StateNormal, color.NRGBA{}},
			{tokens.StateFocus, color.NRGBA{}},
			{tokens.StateHover, vgcolor.Flatten(p.HoverOverlay, chrome)},
			{tokens.StatePressed, vgcolor.Flatten(p.PressOverlay, chrome)},
		} {
			if got := Fill(p, tc.state, chrome); got != tc.want {
				t.Errorf("%s: Fill(%v) = %v, want %v", sc.name, tc.state, got, tc.want)
			}
		}
		if got, want := Rim(p, chrome), vgcolor.Flatten(p.Separator, chrome); got != want {
			t.Errorf("%s: Rim = %v, want the platform's separator over the chrome %v", sc.name, got, want)
		}
		if got, want := Label(p, chrome), vgcolor.Flatten(p.ControlText, chrome); got != want {
			t.Errorf("%s: Label = %v, want the platform's control text over the chrome %v", sc.name, got, want)
		}
		if got, want := Mark(p, chrome), vgcolor.Flatten(p.SecondaryLabel, chrome); got != want {
			t.Errorf("%s: Mark = %v, want the platform's secondary label over the chrome %v", sc.name, got, want)
		}
	}
}

// At rest the trigger lays nothing over the chrome it stands on: alpha zero
// composites as a no-op, so the chrome survives it untouched.
func TestRestingFillIsFullyTransparent(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		if a := Fill(p, tokens.StateNormal, p.SidebarMaterial).A; a != 0 {
			t.Errorf("resting fill alpha = %d, want 0", a)
		}
	}
}

// Press lies beyond hover, so the two states stay two: a control held down
// must not look like one merely under the pointer.
func TestPressLiesBeyondHover(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		hover, press := Fill(p, tokens.StateHover, p.SidebarMaterial), Fill(p, tokens.StatePressed, p.SidebarMaterial)
		if hover == press {
			t.Errorf("hover and press are one colour %v", hover)
		}
		if p.PressOverlay.A <= p.HoverOverlay.A {
			t.Errorf("press coverage %d does not exceed hover's %d", p.PressOverlay.A, p.HoverOverlay.A)
		}
	}
}

// Both overlays are the platform's own black or white at a coverage, and
// what the control paints is that coverage resolved against the chrome: an
// opaque fill, because the platform composites in encoded sRGB and Gio's
// rasterizer would not.
func TestBothOverlaysResolveAgainstTheChrome(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		for _, st := range []tokens.State{tokens.StateHover, tokens.StatePressed} {
			if a := p.HoverOverlay.A; a == 0 || a == 0xff {
				t.Errorf("hover overlay alpha = %d, want the platform's partial coverage", a)
			}
			if a := Fill(p, st, p.SidebarMaterial).A; a != 0xff {
				t.Errorf("%v fill alpha = %d, want an opaque answer", st, a)
			}
			if onChrome, onButton := Fill(p, st, p.SidebarMaterial), Fill(p, st, p.PushButtonFill); onChrome == onButton {
				t.Errorf("%v reads the same on the chrome and on a push button's fill (%v)", st, onChrome)
			}
		}
	}
}
