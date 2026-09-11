package button

import (
	"image/color"
	"testing"

	"github.com/vibrantgio/theme/tokens"
)

// pinFill and pinForeground are a pair no appearance carries: a fixed red of
// the kind a caller pins when the meaning of an action, rather than the
// platform, chooses its colour, and the foreground that reads over it.
var (
	pinFill       = color.NRGBA{0xb3, 0x26, 0x1e, 0xff}
	pinForeground = color.NRGBA{0xff, 0xff, 0xff, 0xff}
)

// transparent is what buttonColors returns for a part the variant does not
// draw: alpha zero, which is no colour a fill could use.
var transparent = color.NRGBA{}

// Every colour this component draws is one field of the platform's set, and
// this is the table that says which. It is asserted in both appearances
// because the platform answers each name per appearance and the mapping must
// be the same one in both.
func TestEmphasisTakesThePlatformsNames(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := sc.p
		for _, tc := range []struct {
			name    string
			state   RenderState
			bg      color.NRGBA
			overlay color.NRGBA
			edge    color.NRGBA
			fg      color.NRGBA
		}{
			{"filled rest", RenderState{}, p.ControlAccent, transparent, transparent, p.AlternateSelectedControlText},
			{"filled hovered", RenderState{Hovered: true}, p.ControlAccent, transparent, transparent, p.AlternateSelectedControlText},
			{"filled focused", RenderState{Focused: true}, p.ControlAccent, transparent, transparent, p.AlternateSelectedControlText},
			{"filled pressed", RenderState{Pressed: true}, p.ControlAccent, p.PressOverlay, transparent, p.AlternateSelectedControlText},
			{"filled disabled", RenderState{Disabled: true}, p.Control, transparent, p.Separator, p.DisabledControlText},

			{"tonal rest", RenderState{Emphasis: Tonal}, p.Control, transparent, p.Separator, p.ControlText},
			{"tonal hovered", RenderState{Emphasis: Tonal, Hovered: true}, p.Control, transparent, p.Separator, p.ControlText},
			{"tonal pressed", RenderState{Emphasis: Tonal, Pressed: true}, p.Control, p.PressOverlay, p.Separator, p.ControlText},
			{"tonal disabled", RenderState{Emphasis: Tonal, Disabled: true}, p.Control, transparent, p.Separator, p.DisabledControlText},

			{"ghost rest", RenderState{Emphasis: Ghost}, transparent, transparent, transparent, p.ControlText},
			{"ghost hovered", RenderState{Emphasis: Ghost, Hovered: true}, transparent, transparent, transparent, p.ControlText},
			{"ghost pressed", RenderState{Emphasis: Ghost, Pressed: true}, transparent, p.PressOverlay, transparent, p.ControlText},
			{"ghost disabled", RenderState{Emphasis: Ghost, Disabled: true}, transparent, transparent, transparent, p.DisabledControlText},

			{"pinned rest", RenderState{Fill: pinFill, OnFill: pinForeground}, pinFill, transparent, transparent, pinForeground},
			{"pinned pressed", RenderState{Fill: pinFill, OnFill: pinForeground, Pressed: true}, pinFill, p.PressOverlay, transparent, pinForeground},
		} {
			bg, overlay, edge, fg := buttonColors(p, tc.state)
			if bg != tc.bg || overlay != tc.overlay || edge != tc.edge || fg != tc.fg {
				t.Errorf("%s/%s: got fill %v overlay %v edge %v foreground %v, want %v %v %v %v",
					sc.name, tc.name, bg, overlay, edge, fg, tc.bg, tc.overlay, tc.edge, tc.fg)
			}
		}
	}
}

// A push button does not tint under the pointer on this platform. The
// reference records a Finder toolbar button that does and a Save dialog's
// push button that does not, so hover must leave every part of every variant
// exactly where rest left it.
func TestNoVariantTintsUnderThePointer(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		for _, e := range []Emphasis{Filled, Tonal, Ghost} {
			restBG, restOverlay, restEdge, restFG := buttonColors(p, RenderState{Emphasis: e})
			bg, overlay, edge, fg := buttonColors(p, RenderState{Emphasis: e, Hovered: true})
			if bg != restBG || overlay != restOverlay || edge != restEdge || fg != restFG {
				t.Errorf("%v hovered moved off rest: %v %v %v %v against %v %v %v %v",
					e, bg, overlay, edge, fg, restBG, restOverlay, restEdge, restFG)
			}
		}
	}
}

// A transparent fill is the ghost emphasis' whole mechanism: alpha zero
// composites as a no-op, so the surface behind the button survives it. Any
// non-zero alpha here would mean a ghost fills whatever it sits on.
func TestGhostRestingFillIsFullyTransparent(t *testing.T) {
	for _, s := range []RenderState{
		{Emphasis: Ghost},
		{Emphasis: Ghost, Hovered: true},
		{Emphasis: Ghost, Focused: true},
		{Emphasis: Ghost, Disabled: true},
	} {
		bg, _, edge, _ := buttonColors(tokens.PlatformLight, s)
		if bg.A != 0 {
			t.Errorf("ghost %+v: fill alpha = %d, want 0", s, bg.A)
		}
		if edge.A != 0 {
			t.Errorf("ghost %+v: edge alpha = %d, want 0", s, edge.A)
		}
	}
}

// A pinned pair is the same colour in both appearances while the accent pair
// it stands in for is not — which is the whole reason the pair exists. A
// caller whose colour is fixed from outside the platform's set needs it to
// stay put.
func TestPinnedFillIsAppearanceStableWhereTheAccentPairIsNot(t *testing.T) {
	pinned := RenderState{Fill: pinFill, OnFill: pinForeground}
	lightBG, _, _, lightFG := buttonColors(tokens.PlatformLight, pinned)
	darkBG, _, _, darkFG := buttonColors(tokens.PlatformDark, pinned)
	if lightBG != darkBG || lightFG != darkFG {
		t.Errorf("pinned pair moved between appearances: light %v on %v, dark %v on %v",
			lightFG, lightBG, darkFG, darkBG)
	}
	if lightBG != pinFill || lightFG != pinForeground {
		t.Errorf("pinned pair resolved to %v on %v, want the caller's %v on %v",
			lightFG, lightBG, pinForeground, pinFill)
	}
}

// Half a pin is no pin: the variant falls back to the platform's accent pair
// rather than drawing a fill nobody chose or a label nobody can read.
func TestHalfAPinIsNoPin(t *testing.T) {
	p := tokens.PlatformLight
	for _, s := range []RenderState{
		{Fill: pinFill},
		{OnFill: pinForeground},
	} {
		bg, _, _, fg := buttonColors(p, s)
		if bg != p.ControlAccent || fg != p.AlternateSelectedControlText {
			t.Errorf("%+v resolved to %v on %v, want the accent pair %v on %v",
				s, fg, bg, p.AlternateSelectedControlText, p.ControlAccent)
		}
	}
}

// The zero value must be Filled emphasis: that is what makes every Props
// and RenderState written without one draw the default action.
func TestZeroEmphasisIsFilled(t *testing.T) {
	var e Emphasis
	if e != Filled {
		t.Errorf("zero Emphasis = %v, want Filled", e)
	}
	if got, want := e.String(), "filled"; got != want {
		t.Errorf("Emphasis(0).String() = %q, want %q", got, want)
	}
	if got, want := Tonal.String()+" "+Ghost.String(), "tonal ghost"; got != want {
		t.Errorf("variant names = %q, want %q", got, want)
	}
}
