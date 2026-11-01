package golden_test

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/reactivego/rx"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// TestTextOnTheWindowMeetsThePlatformStem: a layer of a golden window built
// on StaticTheme draws "Vibrant Gio" in the System font in the label colour
// on the window background. The harness tells the layer the window
// background as its fill. In each scheme the mean stem of the n over four
// offsets is within 0.06 px of the platform's row for the n, or of the
// nearer row where the platform gives two.
func TestTextOnTheWindowMeetsThePlatformStem(t *testing.T) {
	for _, sc := range []struct {
		name string
		set  tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(sc.name, func(t *testing.T) {
			th := golden.StaticTheme(sc.set)
			typ, err := th.Typography.First()
			if err != nil {
				t.Fatalf("typography: %v", err)
			}
			sh := typ.DeterministicShaper()
			role := typ.SystemFont
			fill := sc.set.WindowBackground
			fg := color.Flatten(sc.set.Label, fill)
			var stems []float64
			for _, off := range stemOffsets {
				var dims layout.Dimensions
				line := golden.ThemeWindow(t, image.Pt(stemW, stemLineH), th, func(rx.Observable[theme.Theme]) []rx.Observable[layout.Widget] {
					return []rx.Observable[layout.Widget]{rx.Of[layout.Widget](func(gtx layout.Context) layout.Dimensions {
						paint.FillShape(gtx.Ops, fill, clip.Rect{Max: image.Pt(stemW, stemLineH)}.Op())
						tr := op.Affine(f32.Affine2D{}.Offset(f32.Pt(stemLeft+off, 0))).Push(gtx.Ops)
						defer tr.Pop()
						dims = typeset.Text(gtx, sh, stemWord, role, fg, 1)
						return dims
					})}
				})
				base := dims.Size.Y - dims.Baseline
				stems = append(stems, readStem13(line, sh, role, base, float64(stemLeft+off), fill.R, plateau(line, fill.R)))
			}
			mean := (stems[0] + stems[1] + stems[2] + stems[3]) / 4
			want := platformStem13[sc.set.Dark()]
			t.Logf("stems %.2f %.2f %.2f %.2f, mean %.3f", stems[0], stems[1], stems[2], stems[3], mean)
			if d := min(math.Abs(mean-want[0]), math.Abs(mean-want[1])); d > 0.06+1e-9 {
				t.Errorf("the mean stem is %.3f px, %.3f px from the platform's %.2f", mean, d, want[0])
			}
		})
	}
}
