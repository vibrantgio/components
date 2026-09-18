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
			accent     = p.ControlAccent
			accentHit  = on(p.PressOverlay, accent)
			accentOver = on(p.HoverOverlay, accent)
			push       = p.PushButtonFill
			pushHit    = on(p.PressOverlay, push)
			pushOver   = on(p.HoverOverlay, push)
			ghostHit   = on(p.PressOverlay, plane)
			ghostOver  = on(p.HoverOverlay, plane)
			pinHit     = on(p.PressOverlay, pinFill)
			// Switched off, a fill falls back to the push button's and
			// fades toward the surface at the platform's measured
			// coverage; the hairline over it fades by the same amount.
			pushOff = vgcolor.Flatten(vgcolor.Fade(push, tokens.DisabledCoverage), plane)
			seamOff = on(vgcolor.Fade(p.Separator, tokens.DisabledCoverage), pushOff)
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
			{"filled hovered", RenderState{Hovered: true}, accentOver, transparent, on(p.AlternateSelectedControlText, accentOver), on(p.KeyboardFocusIndicator, accentOver)},
			{"filled focused", RenderState{Focused: true}, accent, transparent, on(p.AlternateSelectedControlText, accent), on(p.KeyboardFocusIndicator, accent)},
			{"filled pressed", RenderState{Pressed: true}, accentHit, transparent, on(p.AlternateSelectedControlText, accentHit), on(p.KeyboardFocusIndicator, accentHit)},
			{"filled disabled", RenderState{Disabled: true}, pushOff, seamOff, on(p.DisabledControlText, pushOff), on(p.KeyboardFocusIndicator, pushOff)},

			{"tonal rest", RenderState{Emphasis: Tonal}, push, on(p.Separator, push), on(p.ControlText, push), on(p.KeyboardFocusIndicator, push)},
			{"tonal hovered", RenderState{Emphasis: Tonal, Hovered: true}, pushOver, on(p.Separator, pushOver), on(p.ControlText, pushOver), on(p.KeyboardFocusIndicator, pushOver)},
			{"tonal pressed", RenderState{Emphasis: Tonal, Pressed: true}, pushHit, on(p.Separator, pushHit), on(p.ControlText, pushHit), on(p.KeyboardFocusIndicator, pushHit)},
			{"tonal disabled", RenderState{Emphasis: Tonal, Disabled: true}, pushOff, seamOff, on(p.DisabledControlText, pushOff), on(p.KeyboardFocusIndicator, pushOff)},

			{"ghost rest", RenderState{Emphasis: Ghost}, transparent, transparent, on(p.ControlText, plane), on(p.KeyboardFocusIndicator, plane)},
			{"ghost hovered", RenderState{Emphasis: Ghost, Hovered: true}, ghostOver, transparent, on(p.ControlText, ghostOver), on(p.KeyboardFocusIndicator, ghostOver)},
			{"ghost pressed", RenderState{Emphasis: Ghost, Pressed: true}, ghostHit, transparent, on(p.ControlText, ghostHit), on(p.KeyboardFocusIndicator, ghostHit)},
			{"ghost disabled", RenderState{Emphasis: Ghost, Disabled: true}, transparent, transparent, on(p.DisabledControlText, plane), on(p.KeyboardFocusIndicator, plane)},

			{"pinned rest", RenderState{Fill: pinFill, Foreground: pinForeground}, pinFill, transparent, pinForeground, on(p.KeyboardFocusIndicator, pinFill)},
			{"pinned pressed", RenderState{Fill: pinFill, Foreground: pinForeground, Pressed: true}, pinHit, transparent, pinForeground, on(p.KeyboardFocusIndicator, pinHit)},

			// A press wins over a hover: the two overlays are one
			// answer and are never laid on each other.
			{"tonal held under the pointer", RenderState{Emphasis: Tonal, Hovered: true, Pressed: true}, pushHit, on(p.Separator, pushHit), on(p.ControlText, pushHit), on(p.KeyboardFocusIndicator, pushHit)},
			{"tonal switched off under the pointer", RenderState{Emphasis: Tonal, Hovered: true, Disabled: true}, pushOff, seamOff, on(p.DisabledControlText, pushOff), on(p.KeyboardFocusIndicator, pushOff)},
		} {
			bg, edge, fg, _, ring := buttonColors(p, tc.state)
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
				bg, edge, fg, _, ring := buttonColors(p, s)
				for _, c := range []color.NRGBA{bg, edge, fg, ring} {
					if c.A != 0 && c.A != 0xff {
						t.Errorf("%v %+v: %v carries a coverage; the button paints opaque", e, s, c)
					}
				}
			}
		}
	}
}

// Every variant tints under the pointer, and by the platform's own overlay.
// A ghost, which carries no fill at rest, gains one: that is the very case
// control-hover-{light,dark}.png measures, a Finder toolbar pop-up reading
// #f2f2f2 on the white band light and #384146 on the #242d32 band dark.
func TestEveryVariantTintsUnderThePointer(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		for _, e := range []Emphasis{Filled, Tonal, Ghost} {
			restBG, _, _, _, _ := buttonColors(p, RenderState{Emphasis: e})
			bg, _, _, _, _ := buttonColors(p, RenderState{Emphasis: e, Hovered: true})
			if bg == restBG {
				t.Errorf("%v hovered is still the resting fill %v", e, bg)
			}
			want := vgcolor.Flatten(p.HoverOverlay, fillOr(restBG, p.WindowBackground))
			if bg != want {
				t.Errorf("%v hovered fill = %v, want the platform's hover overlay over the resting fill, %v", e, bg, want)
			}
		}
	}
}

// A switched-off button is no longer the pixel an enabled Tonal one is. Both
// carry the push button's fill, and the switched-off one fades it toward the
// surface it stands on at the platform's measured coverage — the whole point
// of the reading: the Save dialog's switched-off checkbox reads #f2f2f2 where
// the enabled pop-up seventeen rows above it reads #ececec.
func TestSwitchedOffPartsFromTheEnabledTonal(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		tonal, tonalEdge, _, _, _ := buttonColors(p, RenderState{Emphasis: Tonal})
		for _, e := range []Emphasis{Filled, Tonal} {
			off, offEdge, _, _, _ := buttonColors(p, RenderState{Emphasis: e, Disabled: true})
			if off == tonal {
				t.Errorf("%v switched off fills %v, the same pixel an enabled tonal does", e, off)
			}
			if offEdge == tonalEdge {
				t.Errorf("%v switched off draws the hairline %v, the same pixel an enabled tonal does", e, offEdge)
			}
			want := vgcolor.Flatten(vgcolor.Fade(p.PushButtonFill, tokens.DisabledCoverage), p.WindowBackground)
			if off != want {
				t.Errorf("%v switched off fills %v, want the push button's fill at the platform's disabled coverage, %v", e, off, want)
			}
		}
	}
}

// The light appearance's switched-off fill is the pixel the capture holds. On
// the white the Save sheet carries, the push button's #ececec at the
// platform's disabled coverage lands on the #f2f2f2 the sheet's switched-off
// checkbox reads.
func TestTheSwitchedOffFillIsTheCapturedPixel(t *testing.T) {
	p := tokens.PlatformLight
	got := vgcolor.Flatten(vgcolor.Fade(p.PushButtonFill, tokens.DisabledCoverage), color.NRGBA{0xff, 0xff, 0xff, 0xff})
	want := color.NRGBA{0xf2, 0xf2, 0xf2, 0xff}
	if got != want {
		t.Errorf("the push button's fill switched off on white = %v, want the capture's %v", got, want)
	}
}

// A transparent fill is the ghost emphasis' whole mechanism: alpha zero
// composites as a no-op, so the surface behind the button survives it. Any
// non-zero alpha here would mean a ghost fills whatever it sits on.
func TestGhostRestingFillIsFullyTransparent(t *testing.T) {
	for _, s := range []RenderState{
		{Emphasis: Ghost},
		{Emphasis: Ghost, Focused: true},
		{Emphasis: Ghost, Disabled: true},
	} {
		bg, edge, _, _, _ := buttonColors(tokens.PlatformLight, s)
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
	pinned := RenderState{Fill: pinFill, Foreground: pinForeground}
	lightBG, _, lightFG, _, _ := buttonColors(tokens.PlatformLight, pinned)
	darkBG, _, darkFG, _, _ := buttonColors(tokens.PlatformDark, pinned)
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
		{Foreground: pinForeground},
	} {
		bg, _, fg, _, _ := buttonColors(p, s)
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
