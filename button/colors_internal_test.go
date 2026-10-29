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
		// Under the pointer the push button keeps its resting fill; only the
		// ghost button lays the hover overlay.
		var (
			// The default push button's measured fill, which is what the
			// Filled emphasis paints — not the accent the platform
			// publishes under it.
			deflt     = p.DefaultButtonFill
			defltHit  = on(p.PressOverlay, deflt)
			push      = p.PushButtonFill
			pushHit   = on(p.PressOverlay, push)
			ghostHit  = on(p.PressOverlay, plane)
			ghostOver = on(p.HoverOverlay, plane)
			pinHit    = on(p.PressOverlay, pinFill)
			// Switched off, a fill falls back to the push button's and
			// fades toward the surface at the platform's measured
			// coverage.
			pushOff = vgcolor.Flatten(vgcolor.Fade(push, tokens.DisabledCoverage), plane)
		)
		for _, tc := range []struct {
			name  string
			state RenderState
			bg    color.NRGBA
			fg    color.NRGBA
			ring  color.NRGBA
		}{
			{"filled rest", RenderState{}, deflt, on(p.AlternateSelectedControlText, deflt), on(p.KeyboardFocusIndicator, deflt)},
			{"filled hovered", RenderState{Hovered: true}, deflt, on(p.AlternateSelectedControlText, deflt), on(p.KeyboardFocusIndicator, deflt)},
			{"filled focused", RenderState{Focused: true}, deflt, on(p.AlternateSelectedControlText, deflt), on(p.KeyboardFocusIndicator, deflt)},
			{"filled pressed", RenderState{Pressed: true}, defltHit, on(p.AlternateSelectedControlText, defltHit), on(p.KeyboardFocusIndicator, defltHit)},
			{"filled disabled", RenderState{Disabled: true}, pushOff, on(p.DisabledControlText, pushOff), on(p.KeyboardFocusIndicator, pushOff)},

			{"tonal rest", RenderState{Emphasis: Tonal}, push, on(p.ControlText, push), on(p.KeyboardFocusIndicator, push)},
			{"tonal hovered", RenderState{Emphasis: Tonal, Hovered: true}, push, on(p.ControlText, push), on(p.KeyboardFocusIndicator, push)},
			{"tonal pressed", RenderState{Emphasis: Tonal, Pressed: true}, pushHit, on(p.ControlText, pushHit), on(p.KeyboardFocusIndicator, pushHit)},
			{"tonal disabled", RenderState{Emphasis: Tonal, Disabled: true}, pushOff, on(p.DisabledControlText, pushOff), on(p.KeyboardFocusIndicator, pushOff)},

			{"ghost rest", RenderState{Emphasis: Ghost}, transparent, on(p.ControlText, plane), on(p.KeyboardFocusIndicator, plane)},
			{"ghost hovered", RenderState{Emphasis: Ghost, Hovered: true}, ghostOver, on(p.ControlText, ghostOver), on(p.KeyboardFocusIndicator, ghostOver)},
			{"ghost pressed", RenderState{Emphasis: Ghost, Pressed: true}, ghostHit, on(p.ControlText, ghostHit), on(p.KeyboardFocusIndicator, ghostHit)},
			{"ghost disabled", RenderState{Emphasis: Ghost, Disabled: true}, transparent, on(p.DisabledControlText, plane), on(p.KeyboardFocusIndicator, plane)},

			{"pinned rest", RenderState{Fill: pinFill, Foreground: pinForeground}, pinFill, pinForeground, on(p.KeyboardFocusIndicator, pinFill)},
			{"pinned pressed", RenderState{Fill: pinFill, Foreground: pinForeground, Pressed: true}, pinHit, pinForeground, on(p.KeyboardFocusIndicator, pinHit)},

			// A press wins over a hover: the two overlays are one
			// answer and are never laid on each other.
			{"tonal held under the pointer", RenderState{Emphasis: Tonal, Hovered: true, Pressed: true}, pushHit, on(p.ControlText, pushHit), on(p.KeyboardFocusIndicator, pushHit)},
			{"tonal switched off under the pointer", RenderState{Emphasis: Tonal, Hovered: true, Disabled: true}, pushOff, on(p.DisabledControlText, pushOff), on(p.KeyboardFocusIndicator, pushOff)},
		} {
			bg, fg, _, ring := buttonColors(p, tc.state)
			if bg != tc.bg || fg != tc.fg || ring != tc.ring {
				t.Errorf("%s/%s: got fill %v foreground %v ring %v, want %v %v %v",
					sc.name, tc.name, bg, fg, ring, tc.bg, tc.fg, tc.ring)
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
				bg, fg, _, ring := buttonColors(p, s)
				for _, c := range []color.NRGBA{bg, fg, ring} {
					if c.A != 0 && c.A != 0xff {
						t.Errorf("%v %+v: %v carries a coverage; the button paints opaque", e, s, c)
					}
				}
			}
		}
	}
}

// The push button lays no hover overlay. MEASURED,
// save-panel-cancel-{rest,hover}-{light,dark}.png: "Cancel" at rest and under
// the pointer differ in 0 pixels. MEASURED, save-panel-untinted-{light,dark}.png:
// its fill reads #ececec light and #2f2f2f dark. Held, it reads #d5d5d5 light
// (save-panel-cancel-pressed-light.png) and #434343 dark
// (save-panel-cancel-pressed-untinted-dark.png).
//
// The filled and tonal buttons are the push button; the ghost button lays
// the hover overlay over the surface it stands on.
func TestPushButtonLaysNoHoverOverlay(t *testing.T) {
	for _, sc := range []struct {
		name          string
		p             tokens.PlatformColors
		tonal, tonalP color.NRGBA
	}{
		{"light", tokens.PlatformLight, color.NRGBA{R: 0xec, G: 0xec, B: 0xec, A: 0xff}, color.NRGBA{R: 0xd5, G: 0xd5, B: 0xd5, A: 0xff}},
		{"dark", tokens.PlatformDark, color.NRGBA{R: 0x2f, G: 0x2f, B: 0x2f, A: 0xff}, color.NRGBA{R: 0x43, G: 0x43, B: 0x43, A: 0xff}},
	} {
		p := sc.p
		for _, e := range []Emphasis{Filled, Tonal} {
			restBG, restFG, _, _ := buttonColors(p, RenderState{Emphasis: e})
			bg, fg, _, _ := buttonColors(p, RenderState{Emphasis: e, Hovered: true})
			if bg != restBG || fg != restFG {
				t.Errorf("%s: %v under the pointer = %v on %v, want the resting %v on %v", sc.name, e, fg, bg, restFG, restBG)
			}
		}
		if bg, _, _, _ := buttonColors(p, RenderState{Emphasis: Tonal, Hovered: true}); bg != sc.tonal {
			t.Errorf("%s: tonal under the pointer = %v, want the measured %v", sc.name, bg, sc.tonal)
		}
		if bg, _, _, _ := buttonColors(p, RenderState{Emphasis: Tonal, Hovered: true, Pressed: true}); bg != sc.tonalP {
			t.Errorf("%s: tonal held = %v, want the measured %v", sc.name, bg, sc.tonalP)
		}
		ghost, _, _, _ := buttonColors(p, RenderState{Emphasis: Ghost, Hovered: true})
		if want := vgcolor.Flatten(p.HoverOverlay, p.WindowBackground); ghost != want {
			t.Errorf("%s: ghost under the pointer = %v, want the hover overlay over the surface, %v", sc.name, ghost, want)
		}
		// The symbol push button and the form segmented control.
		if got := variantFill(p, Form, tokens.StateHover); got != sc.tonal {
			t.Errorf("%s: form symbol button under the pointer = %v, want the measured %v", sc.name, got, sc.tonal)
		}
		if got, want := segmentFill(p, Form, tokens.StateHover), vgcolor.Flatten(p.HoverOverlay, p.PushButtonFill); got != want {
			t.Errorf("%s: form segment under the pointer = %v, want the hover overlay over the fill, %v", sc.name, got, want)
		}
	}
}

// A switched-off button is no longer the pixel an enabled Tonal one is. Both
// carry the push button's fill, and the switched-off one fades it toward the
// surface it stands on at the platform's measured coverage — the whole point
// of the reading: the Save panel's switched-off checkbox reads #f2f2f2 where
// the enabled pop-up seventeen rows above it reads #ececec.
func TestSwitchedOffPartsFromTheEnabledTonal(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		tonal, _, _, _ := buttonColors(p, RenderState{Emphasis: Tonal})
		for _, e := range []Emphasis{Filled, Tonal} {
			off, _, _, _ := buttonColors(p, RenderState{Emphasis: e, Disabled: true})
			if off == tonal {
				t.Errorf("%v switched off fills %v, the same pixel an enabled tonal does", e, off)
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
		bg, _, _, _ := buttonColors(tokens.PlatformLight, s)
		if bg.A != 0 {
			t.Errorf("ghost %+v: fill alpha = %d, want 0", s, bg.A)
		}
	}
}

// A pinned pair is the caller's own colours in both appearances: a fill
// fixed from outside the platform's set answers to neither appearance, which
// is the whole reason the pair exists.
func TestPinnedFillIsTheCallersPairInBothAppearances(t *testing.T) {
	pinned := RenderState{Fill: pinFill, Foreground: pinForeground}
	lightBG, lightFG, _, _ := buttonColors(tokens.PlatformLight, pinned)
	darkBG, darkFG, _, _ := buttonColors(tokens.PlatformDark, pinned)
	if lightBG != darkBG || lightFG != darkFG {
		t.Errorf("pinned pair moved between appearances: light %v on %v, dark %v on %v",
			lightFG, lightBG, darkFG, darkBG)
	}
	if lightBG != pinFill || lightFG != pinForeground {
		t.Errorf("pinned pair resolved to %v on %v, want the caller's %v on %v",
			lightFG, lightBG, pinForeground, pinFill)
	}
}

// Half a pin is no pin: the variant falls back to the platform's
// default-button pair rather than drawing a fill nobody chose or a title
// nobody can read.
func TestHalfAPinIsNoPin(t *testing.T) {
	p := tokens.PlatformLight
	for _, s := range []RenderState{
		{Fill: pinFill},
		{Foreground: pinForeground},
	} {
		bg, fg, _, _ := buttonColors(p, s)
		if want := on(p.AlternateSelectedControlText, p.DefaultButtonFill); bg != p.DefaultButtonFill || fg != want {
			t.Errorf("%+v resolved to %v on %v, want the default-button pair %v on %v",
				s, fg, bg, want, p.DefaultButtonFill)
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
