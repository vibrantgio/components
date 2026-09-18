package button_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// onChrome stands the control on the chrome material, which is the band a
// toolbar control stands in everywhere in this library.
func onChrome(p tokens.PlatformColors, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, p.SidebarMaterial, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// chromeStates is the matrix the chrome variant is recorded in: at rest, under
// the pointer, held, focused and switched off, in both appearances.
var chromeStates = []struct {
	name  string
	state button.RenderState
}{
	{"normal", button.RenderState{}},
	{"hovered", button.RenderState{Hovered: true}},
	{"pressed", button.RenderState{Pressed: true}},
	{"focused", button.RenderState{Focused: true}},
	{"disabled", button.RenderState{Disabled: true}},
}

// TestChromeButtonGolden records the platform's bordered toolbar control with
// a symbol for its label, standing on the chrome material in both appearances.
func TestChromeButtonGolden(t *testing.T) {
	size := image.Pt(120, 90)
	for _, sc := range []struct {
		name   string
		colors tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		for _, st := range chromeStates {
			name := "chrome-" + sc.name + "-" + st.name
			t.Run(name, func(t *testing.T) {
				s := st.state
				s.Surface = sc.colors.SidebarMaterial
				w := button.RenderChrome(crossIcon, sc.colors, tokens.Comfortable, s)
				golden.Render(t, name, size, onChrome(sc.colors, centred(w)))
			})
		}
	}
}

// centred stands a control clear of the frame's edges so the shadow it casts
// has room in the recorded image.
func centred(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		defer op.Offset(image.Pt(40, 27)).Push(gtx.Ops).Pop()
		return w(gtx)
	}
}

// TestChromeButtonIsTheMeasuredControl holds the drawn box to the platform's:
// the toolbar control's height, the mark's box with the measured room on each
// side of it across, and the capsule corner that follows the height.
func TestChromeButtonIsTheMeasuredControl(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(200, 200)),
		Ops:         &ops,
	}
	dims := button.RenderChrome(crossIcon, tokens.PlatformLight, tokens.Comfortable, button.RenderState{})(gtx)
	if want := int(tokens.Comfortable.ToolbarControlHeight); dims.Size.Y != want {
		t.Errorf("chrome button height = %d, want the toolbar control's %d", dims.Size.Y, want)
	}
	if dims.Size.Y == int(tokens.Comfortable.ControlHeight) {
		t.Errorf("chrome button height = %d, the dialog control's; the platform draws a toolbar control taller", dims.Size.Y)
	}
	if want := 38; dims.Size.X != want {
		t.Errorf("chrome button width = %d, want the measured %d", dims.Size.X, want)
	}
}

// TestChromeButtonCastsTheMeasuredShadow reads the band around the control:
// the platform's shadow darkens it, more under the control than over it,
// because the rectangle casting it is sunk below the control.
func TestChromeButtonCastsTheMeasuredShadow(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(sc.name, func(t *testing.T) {
			p := sc.colors
			w := button.RenderChrome(crossIcon, p, tokens.Comfortable, button.RenderState{Surface: p.SidebarMaterial})
			img := golden.Capture(t, image.Pt(120, 90), onChrome(p, centred(w)))
			at := func(x, y int) color.NRGBA {
				r, g, b, a := img.At(x, y).RGBA()
				return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
			}
			band := p.SidebarMaterial
			// The control runs y 27–62 at x 40–77; one row under it and one
			// over it, on its own middle column.
			under, over := at(59, 63), at(59, 26)
			if under == band {
				t.Errorf("the row under the control reads the bare band %v; the platform's control darkens it", under)
			}
			if want := vgcolor.Flatten(p.ToolbarControlShadow, band); under != want {
				t.Errorf("the row under the control = %v, want the band under this shadow's peak %v", under, want)
			}
			// The row over it is lighter than the row under it — the
			// rectangle casting the shadow is sunk below the control — and in
			// the dark appearance that ramp rounds away against a band this
			// dark, which is what the platform's own captures show: the dark
			// control is told from its band by its fill and its rim.
			if darken(band, over) >= darken(band, under) {
				t.Errorf("the row over the control (%v) is no lighter than the row under it (%v); the shadow is sunk below the control", over, under)
			}
			if sc.name == "light" && over == band {
				t.Errorf("the row over the control reads the bare band %v; the light shadow reaches over the control too", over)
			}
		})
	}
}

// TestChromeIgnoresEmphasis holds the ruling that there is ONE bordered
// toolbar control: the platform draws it one way, so the emphasis a form
// button wears reaches nothing here.
func TestChromeIgnoresEmphasis(t *testing.T) {
	size := image.Pt(120, 90)
	p := tokens.PlatformLight
	base := golden.Capture(t, size, onChrome(p, centred(
		button.RenderChrome(crossIcon, p, tokens.Comfortable, button.RenderState{}))))
	for _, e := range []button.Emphasis{button.Tonal, button.Ghost} {
		got := golden.Capture(t, size, onChrome(p, centred(
			button.RenderChrome(crossIcon, p, tokens.Comfortable, button.RenderState{Emphasis: e}))))
		if n := golden.PixelDiff(base, got); n != 0 {
			t.Errorf("the %s emphasis moves %d pixel(s) of the chrome variant; the platform draws one bordered toolbar control", e, n)
		}
	}
}

// TestVariantNamesAreTheVocabularys pins the two names to the words the
// Language uses for where a control lives.
func TestVariantNamesAreTheVocabularys(t *testing.T) {
	if got := button.Form.String(); got != "form" {
		t.Errorf("Form.String() = %q, want %q", got, "form")
	}
	if got := button.Chrome.String(); got != "chrome" {
		t.Errorf("Chrome.String() = %q, want %q", got, "chrome")
	}
	if button.Form != 0 {
		t.Errorf("Form = %d, want the zero value: a button among content is what a button is by default", button.Form)
	}
}

// darken is how many 255ths of its red channel a band lost under a shadow.
func darken(band, got color.NRGBA) int { return int(band.R) - int(got.R) }
