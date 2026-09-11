package button

import (
	"image/color"
	"testing"

	vgcolor "github.com/vibrantgio/theme/color"
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

// on is the table's own copy of the rule buttonColors applies, kept here so
// each expectation is written independently of the code that resolved it.
func on(name, beneath color.NRGBA) color.NRGBA { return vgcolor.Flatten(name, beneath) }

// Every colour this component draws is one field of the platform's set, and
// this is the table that says which. It is asserted in both appearances
// because the platform answers each name per appearance and the mapping must
// be the same one in both.
//
// Each answer is that field resolved against the fill it lands on: the
// platform composites a coverage in encoded sRGB, so what the button paints
// is opaque and what this table names is the pair — the field and what is
// under it.
func TestEmphasisTakesThePlatformsNames(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := sc.p
		plane := p.WindowBackground
		// The fill each variant carries, and the press overlay laid onto it.
		var (
			accent    = p.ControlAccent
			accentHit = on(p.PressOverlay, accent)
			push      = p.PushButtonFill
			pushHit   = on(p.PressOverlay, push)
			ghostHit  = on(p.PressOverlay, plane)
			pinHit    = on(p.PressOverlay, pinFill)
		)
		for _, tc := range []struct {
			name  string
			state RenderState
			bg    color.NRGBA
			edge  color.NRGBA
			fg    color.NRGBA
			ring  color.NRGBA
		}{
			{"filled rest", RenderState{}, accent, transparent, on(p.AlternateSelectedControlText, accent), on(p.KeyboardFocusIndicator, accent)},
			{"filled hovered", RenderState{Hovered: true}, accent, transparent, on(p.AlternateSelectedControlText, accent), on(p.KeyboardFocusIndicator, accent)},
			{"filled focused", RenderState{Focused: true}, accent, transparent, on(p.AlternateSelectedControlText, accent), on(p.KeyboardFocusIndicator, accent)},
			{"filled pressed", RenderState{Pressed: true}, accentHit, transparent, on(p.AlternateSelectedControlText, accentHit), on(p.KeyboardFocusIndicator, accentHit)},
			{"filled disabled", RenderState{Disabled: true}, push, on(p.Separator, push), on(p.DisabledControlText, push), on(p.KeyboardFocusIndicator, push)},

			{"tonal rest", RenderState{Emphasis: Tonal}, push, on(p.Separator, push), on(p.ControlText, push), on(p.KeyboardFocusIndicator, push)},
			{"tonal hovered", RenderState{Emphasis: Tonal, Hovered: true}, push, on(p.Separator, push), on(p.ControlText, push), on(p.KeyboardFocusIndicator, push)},
			{"tonal pressed", RenderState{Emphasis: Tonal, Pressed: true}, pushHit, on(p.Separator, pushHit), on(p.ControlText, pushHit), on(p.KeyboardFocusIndicator, pushHit)},
			{"tonal disabled", RenderState{Emphasis: Tonal, Disabled: true}, push, on(p.Separator, push), on(p.DisabledControlText, push), on(p.KeyboardFocusIndicator, push)},

			{"ghost rest", RenderState{Emphasis: Ghost}, transparent, transparent, on(p.ControlText, plane), on(p.KeyboardFocusIndicator, plane)},
			{"ghost hovered", RenderState{Emphasis: Ghost, Hovered: true}, transparent, transparent, on(p.ControlText, plane), on(p.KeyboardFocusIndicator, plane)},
			{"ghost pressed", RenderState{Emphasis: Ghost, Pressed: true}, ghostHit, transparent, on(p.ControlText, ghostHit), on(p.KeyboardFocusIndicator, ghostHit)},
			{"ghost disabled", RenderState{Emphasis: Ghost, Disabled: true}, transparent, transparent, on(p.DisabledControlText, plane), on(p.KeyboardFocusIndicator, plane)},

			{"pinned rest", RenderState{Fill: pinFill, OnFill: pinForeground}, pinFill, transparent, pinForeground, on(p.KeyboardFocusIndicator, pinFill)},
			{"pinned pressed", RenderState{Fill: pinFill, OnFill: pinForeground, Pressed: true}, pinHit, transparent, pinForeground, on(p.KeyboardFocusIndicator, pinHit)},
		} {
			bg, edge, fg, ring := buttonColors(p, tc.state)
			if bg != tc.bg || edge != tc.edge || fg != tc.fg || ring != tc.ring {
				t.Errorf("%s/%s: got fill %v edge %v foreground %v ring %v, want %v %v %v %v",
					sc.name, tc.name, bg, edge, fg, ring, tc.bg, tc.edge, tc.fg, tc.ring)
			}
		}
	}
}

// Nothing the button hands Gio carries a coverage: the platform's names are
// resolved here, in the space the platform composites in, and a part the
// variant does not draw comes back at alpha zero rather than translucent.
func TestNothingTheButtonPaintsCarriesACoverage(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		for _, e := range []Emphasis{Filled, Tonal, Ghost} {
			for _, s := range []RenderState{
				{Emphasis: e}, {Emphasis: e, Pressed: true},
				{Emphasis: e, Focused: true}, {Emphasis: e, Disabled: true},
			} {
				bg, edge, fg, ring := buttonColors(p, s)
				for _, c := range []color.NRGBA{bg, edge, fg, ring} {
					if c.A != 0 && c.A != 0xff {
						t.Errorf("%v %+v: %v carries a coverage; the button paints opaque", e, s, c)
					}
				}
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
			restBG, restEdge, restFG, restRing := buttonColors(p, RenderState{Emphasis: e})
			bg, edge, fg, ring := buttonColors(p, RenderState{Emphasis: e, Hovered: true})
			if bg != restBG || edge != restEdge || fg != restFG || ring != restRing {
				t.Errorf("%v hovered moved off rest: %v %v %v %v against %v %v %v %v",
					e, bg, edge, fg, ring, restBG, restEdge, restFG, restRing)
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
		bg, edge, _, _ := buttonColors(tokens.PlatformLight, s)
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
	lightBG, _, lightFG, _ := buttonColors(tokens.PlatformLight, pinned)
	darkBG, _, darkFG, _ := buttonColors(tokens.PlatformDark, pinned)
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
		bg, _, fg, _ := buttonColors(p, s)
		if want := on(p.AlternateSelectedControlText, p.ControlAccent); bg != p.ControlAccent || fg != want {
			t.Errorf("%+v resolved to %v on %v, want the accent pair %v on %v",
				s, fg, bg, want, p.ControlAccent)
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
