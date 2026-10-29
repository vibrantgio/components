package input_test

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op/paint"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/theme/tokens"
)

// TestTheCheckedBoxAndTheSelectedRadioAreTheMeasuredAccentFill reads the
// fill of the checked box and of the selected radio button at the values
// the stored captures measure, one per appearance. MEASURED,
// keyboard-shortcuts-sheet-{light,dark}.png and
// system-settings-radios-{light,dark}.png: #157bf5 light and #1b81fb dark.
// controlAccentColor reports #007aff in both appearances, and the platform
// draws neither control in it.
func TestTheCheckedBoxAndTheSelectedRadioAreTheMeasuredAccentFill(t *testing.T) {
	const size = 44
	c := int(tokens.Comfortable.CheckboxRowHeight) / 2
	for _, sc := range []struct {
		name     string
		platform tokens.PlatformColors
		want     stdcolor.NRGBA
	}{
		{"light", tokens.PlatformLight, stdcolor.NRGBA{R: 0x15, G: 0x7b, B: 0xf5, A: 0xff}},
		{"dark", tokens.PlatformDark, stdcolor.NRGBA{R: 0x1b, G: 0x81, B: 0xfb, A: 0xff}},
	} {
		t.Run(sc.name, func(t *testing.T) {
			surface := sc.platform.WindowBackground
			box := input.RenderCheckbox(nil, sc.platform, tokens.Spacing, tokens.DefaultTypography.SystemFont,
				input.CheckboxRenderState{Checked: true, Surface: surface})
			disc := input.RenderRadio(nil, sc.platform, tokens.Spacing, tokens.DefaultTypography.SystemFont,
				input.RadioRenderState{Selected: true, Surface: surface})
			for _, ctl := range []struct {
				what string
				w    layout.Widget
				// Points inside the symbol that neither the check mark nor
				// the dot covers, as offsets from the symbol's centre.
				at []image.Point
			}{
				{"the checked box", box, []image.Point{{-6, -6}, {5, 5}}},
				{"the selected radio button", disc, []image.Point{{-5, 0}, {4, 0}, {0, -5}, {0, 4}}},
			} {
				img := golden.Capture(t, image.Pt(size, size), func(gtx layout.Context) layout.Dimensions {
					paint.Fill(gtx.Ops, surface)
					return ctl.w(gtx)
				})
				if img == nil {
					return
				}
				for _, d := range ctl.at {
					got := img.RGBAAt(c+d.X, c+d.Y)
					want := stdcolor.RGBA{R: sc.want.R, G: sc.want.G, B: sc.want.B, A: 0xff}
					if got != want {
						t.Errorf("%s reads %v at (%d, %d) from its centre, want the measured %v", ctl.what, got, d.X, d.Y, sc.want)
					}
				}
			}
		})
	}
}
