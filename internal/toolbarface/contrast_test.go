package toolbarface

import (
	"image/color"
	"testing"

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
		for _, tc := range []struct {
			state tokens.State
			want  color.NRGBA
		}{
			{tokens.StateNormal, color.NRGBA{}},
			{tokens.StateFocus, color.NRGBA{}},
			{tokens.StateHover, p.HoverOverlay},
			{tokens.StatePressed, p.PressOverlay},
		} {
			if got := Fill(p, tc.state); got != tc.want {
				t.Errorf("%s: Fill(%v) = %v, want %v", sc.name, tc.state, got, tc.want)
			}
		}
		if got, want := Rim(p), p.Separator; got != want {
			t.Errorf("%s: Rim = %v, want the platform's separator %v", sc.name, got, want)
		}
		if got, want := Label(p), p.ControlText; got != want {
			t.Errorf("%s: Label = %v, want the platform's control text %v", sc.name, got, want)
		}
		if got, want := Mark(p), p.SecondaryLabel; got != want {
			t.Errorf("%s: Mark = %v, want the platform's secondary label %v", sc.name, got, want)
		}
	}
}

// At rest the trigger lays nothing over the chrome it stands on: alpha zero
// composites as a no-op, so the chrome survives it untouched.
func TestRestingFillIsFullyTransparent(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		if a := Fill(p, tokens.StateNormal).A; a != 0 {
			t.Errorf("resting fill alpha = %d, want 0", a)
		}
	}
}

// Press lies beyond hover, so the two states stay two: a control held down
// must not look like one merely under the pointer.
func TestPressLiesBeyondHover(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		hover, press := Fill(p, tokens.StateHover), Fill(p, tokens.StatePressed)
		if hover == press {
			t.Errorf("hover and press are one colour %v", hover)
		}
		if press.A <= hover.A {
			t.Errorf("press coverage %d does not exceed hover's %d", press.A, hover.A)
		}
	}
}

// Both overlays are the platform's own black or white at a coverage, so the
// control needs to know nothing about the chrome beneath it.
func TestBothOverlaysCarryACoverage(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		for _, st := range []tokens.State{tokens.StateHover, tokens.StatePressed} {
			if a := Fill(p, st).A; a == 0 || a == 0xff {
				t.Errorf("%v overlay alpha = %d, want the platform's partial coverage", st, a)
			}
		}
	}
}
