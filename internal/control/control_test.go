package control_test

import (
	"image/color"
	"testing"

	"github.com/vibrantgio/components/internal/control"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// One control family, one set of names: the text field, the checkbox, the
// radio and the picker's field trigger all draw their box from these, and
// each is the platform's own name for that part of a field.
func TestTheBoxTakesThePlatformsNames(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := sc.p
		if got, want := control.Border(p), p.FieldEdge; got != want {
			t.Errorf("%s: Border = %v, want the platform's field edge %v", sc.name, got, want)
		}
		if got, want := control.Fill(p), p.TextBackground; got != want {
			t.Errorf("%s: Fill = %v, want the platform's text background %v", sc.name, got, want)
		}
		if got, want := control.Recess(p), p.SidebarSearchFill; got != want {
			t.Errorf("%s: Recess = %v, want the platform's sidebar search fill %v", sc.name, got, want)
		}
		if got := control.Recess(p); got == control.FieldFill(p, color.NRGBA{}) {
			t.Errorf("%s: Recess = %v, the same as a form field's interior: a recess on chrome is a fill of its own", sc.name, got)
		}
		if got, want := control.Placeholder(p, control.Fill(p)), vgcolor.Flatten(p.PlaceholderText, control.Fill(p)); got != want {
			t.Errorf("%s: Placeholder = %v, want the platform's placeholder text over the fill %v", sc.name, got, want)
		}
	}
}

// The prompt has to be visibly not a value, and the platform says how far:
// its placeholder carries a coverage over the field's own fill rather than
// being a second opaque colour. What this package hands back is that
// coverage resolved against the fill, because the platform composites it in
// encoded sRGB and Gio's rasterizer would not.
func TestThePlaceholderResolvesACoverageAgainstTheFill(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		if a := p.PlaceholderText.A; a == 0 || a == 0xff {
			t.Errorf("placeholderTextColor alpha = %d, want the platform's partial coverage", a)
		}
		fill := control.Fill(p)
		got := control.Placeholder(p, fill)
		if got.A != 0xff {
			t.Errorf("Placeholder alpha = %d, want an opaque answer", got.A)
		}
		if got == fill || got == p.Text {
			t.Errorf("Placeholder = %v, want a prompt distinct from both the fill and a value", got)
		}
	}
}

// TestTheDisabledBoxDrainsTheAccent: a switched-off control that still
// carries a value wears the platform's disabled coverage over what it stands
// on, not the accent at a fraction of its own. The Save dialog's two
// switched-off checkboxes carry no accent at all, and their wording reads at
// DisabledControlText's coverage — #bdbdbd on the light sheet's white,
// #595f62 on the dark sheet's #232a2f.
func TestTheDisabledBoxDrainsTheAccent(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := sc.p
		standsOn := p.WindowBackground
		fill := control.DisabledFill(p, standsOn)
		if want := vgcolor.Flatten(p.DisabledControlText, standsOn); fill != want {
			t.Errorf("%s: DisabledFill = %v, want the platform's disabled text over the surface %v", sc.name, fill, want)
		}
		if fill.A != 0xff {
			t.Errorf("%s: DisabledFill = %v; a fill handed to the rasterizer is opaque", sc.name, fill)
		}
		if fill == p.ControlAccent {
			t.Errorf("%s: DisabledFill is the accent; a switched-off control carries none", sc.name)
		}
		mark := control.DisabledMark(p, fill)
		if want := vgcolor.Flatten(p.ControlText, fill); mark != want {
			t.Errorf("%s: DisabledMark = %v, want the platform's control text over the fill %v", sc.name, mark, want)
		}
		// The mark still has to be resolvable against the fill it is drawn
		// on: a disabled control is dimmed, not erased.
		if lc := vgcolor.Magnitude(mark, fill); lc < tokens.GraphicFloor {
			t.Errorf("%s: the disabled mark reads |Lc| %.1f on its fill, under the graphic floor %.1f", sc.name, lc, tokens.GraphicFloor)
		}
	}
}
