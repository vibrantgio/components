package controlface

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
		fill := p.ToolbarControlFill
		for _, tc := range []struct {
			state tokens.State
			want  color.NRGBA
		}{
			{tokens.StateNormal, fill},
			{tokens.StateFocus, fill},
			{tokens.StateHover, vgcolor.Flatten(p.HoverOverlay, fill)},
			{tokens.StatePressed, vgcolor.Flatten(p.PressOverlay, fill)},
		} {
			if got := Fill(p, tc.state); got != tc.want {
				t.Errorf("%s: Fill(%v) = %v, want %v", sc.name, tc.state, got, tc.want)
			}
		}
		// The rim is the platform's measured value where the platform draws
		// one, and nothing where it does not: MEASURED, the dark toolbar
		// control wears a 1 px #404040 highlight over its #262626 fill and
		// the light one wears no edge at all, its band stepping straight up
		// to the control's white.
		rim := Rim(p)
		if sc.name == "dark" {
			if want := (color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}); rim != want {
				t.Errorf("dark: Rim = %v, want the measured %v", rim, want)
			}
			if rim.R <= fill.R || rim.G <= fill.G || rim.B <= fill.B {
				t.Errorf("dark: Rim %v does not lift the fill %v; the platform's rim there is a highlight", rim, fill)
			}
			// The seam over the fill is the answer this name replaced: it
			// falls five of 255 short of the measured pixel, which is why
			// the rim is a value of its own.
			if seam := vgcolor.Flatten(p.Separator, fill); seam == rim {
				t.Errorf("dark: the seam over the fill %v lands the measured rim; the name is carried because it does not", seam)
			}
		} else if rim.A != 0 {
			t.Errorf("light: Rim = %v, want no edge at all — the platform draws none over this fill", rim)
		}
		// The toolbar's own label colour, and not the control text a FORM
		// control draws: MEASURED off the band's bare title and its glyphs,
		// which hold one plateau the control text does not land.
		if got, want := Label(p, fill), p.ToolbarLabel; got != want {
			t.Errorf("%s: Label = %v, want the toolbar's measured label %v", sc.name, got, want)
		}
		if got, want := Mark(p, fill), p.ToolbarLabel; got != want {
			t.Errorf("%s: Mark = %v, want the toolbar's measured label %v", sc.name, got, want)
		}
		if ct := vgcolor.Flatten(p.ControlText, fill); Mark(p, fill) == ct {
			t.Errorf("%s: the control text over the fill %v lands the measured label; the value is carried because it does not", sc.name, ct)
		}
		if Mark(p, fill) != Label(p, fill) {
			t.Errorf("%s: the mark and the wording are two colours; one control reads in one foreground", sc.name)
		}
	}
}

// At rest the trigger draws the platform's measured toolbar control fill,
// which stands lighter than the chrome band it is on: the control is a figure
// on its band and not part of it.
func TestRestingFillStandsOffTheChrome(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		rest := Fill(p, tokens.StateNormal)
		if rest != p.ToolbarControlFill {
			t.Errorf("resting fill = %v, want the measured toolbar control fill %v", rest, p.ToolbarControlFill)
		}
		if rest.A != 0xff {
			t.Errorf("resting fill alpha = %d, want an opaque answer", rest.A)
		}
		band := p.SidebarMaterial
		if rest.R <= band.R || rest.G <= band.G || rest.B <= band.B {
			t.Errorf("resting fill %v is not lighter than the chrome %v on every channel", rest, band)
		}
	}
}

// The light hover lands on the capture to the byte: control-hover-light.png
// holds a Finder toolbar control moving from #ffffff to #f2f2f2 under the
// pointer, which is the overlay over the fill this control draws at rest.
func TestLightHoverIsTheCapturedPixel(t *testing.T) {
	want := color.NRGBA{R: 0xf2, G: 0xf2, B: 0xf2, A: 0xff}
	if got := Fill(tokens.PlatformLight, tokens.StateHover); got != want {
		t.Errorf("light hover = %v, want the captured %v", got, want)
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
		if p.PressOverlay.A <= p.HoverOverlay.A {
			t.Errorf("press coverage %d does not exceed hover's %d", p.PressOverlay.A, p.HoverOverlay.A)
		}
	}
}

// Both overlays are the platform's own black or white at a coverage, and what
// the control paints is that coverage resolved against its own fill: an opaque
// answer, because the platform composites in encoded sRGB and Gio's rasterizer
// would not.
func TestBothOverlaysResolveAgainstTheFill(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		for _, st := range []tokens.State{tokens.StateHover, tokens.StatePressed} {
			if a := p.HoverOverlay.A; a == 0 || a == 0xff {
				t.Errorf("hover overlay alpha = %d, want the platform's partial coverage", a)
			}
			if a := Fill(p, st).A; a != 0xff {
				t.Errorf("%v fill alpha = %d, want an opaque answer", st, a)
			}
			if got := Fill(p, st); got == p.ToolbarControlFill {
				t.Errorf("%v reads as the resting fill %v", st, got)
			}
		}
	}
}
