package golden_test

import (
	"image"
	stdcolor "image/color"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/font"
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

// TestTheDefaultButtonTitleDrawsInTheFaceForLighterText: the default
// button's title is white on DefaultButtonFill. The title is lighter than
// that fill in both schemes, so TextStemWeight gives the lighter text's
// weight and FontOn gives the face as shipped, not the darker text family.
func TestTheDefaultButtonTitleDrawsInTheFaceForLighterText(t *testing.T) {
	for _, sc := range []struct {
		name string
		set  tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(sc.name, func(t *testing.T) {
			typ, err := golden.StaticTheme(sc.set).Typography.First()
			if err != nil {
				t.Fatalf("typography: %v", err)
			}
			role := typ.SystemFont
			fill := sc.set.DefaultButtonFill
			title := sc.set.AlternateSelectedControlText
			if title != (stdcolor.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}) {
				t.Fatalf("the default button's title is %v, want white", title)
			}
			if got := tokens.TextStemWeight(title, fill); got != tokens.StemWeightLighterText {
				t.Errorf("the stem weight of the title is %v, want %v", got, tokens.StemWeightLighterText)
			}
			got := typeset.FontOn(typeset.WithFill(layout.Context{}, fill), role, font.Normal, title)
			if want := typeset.Font(role, font.Normal); got.Typeface != want.Typeface {
				t.Errorf("the title draws in %q, want the face as shipped, %q", got.Typeface, want.Typeface)
			}
		})
	}
}

// TestTheSecondaryLabelMeetsTheLabelStem: on the window background, text in
// the secondary label colour draws in the same face as text in the label
// colour, in both schemes. The stem of the n in "Vibrant", read
// linearised, is the label colour's within 0.02 px. Each stem is the mean of
// four readings at the offsets 0, ¼, ½ and ¾ px.
//
// The linearised reading is a test method only. The platform's rows at
// 13 and 10 pt stay the acceptance for the stem.
func TestTheSecondaryLabelMeetsTheLabelStem(t *testing.T) {
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
			label := color.Flatten(sc.set.Label, fill)
			labelTwo := color.Flatten(sc.set.SecondaryLabel, fill)

			onFill := typeset.WithFill(layout.Context{}, fill)
			lf := typeset.FontOn(onFill, role, font.Normal, label)
			sf := typeset.FontOn(onFill, role, font.Normal, labelTwo)
			if lf.Typeface != sf.Typeface {
				t.Errorf("the secondary label draws in %q and the label in %q", sf.Typeface, lf.Typeface)
			}

			mean := func(fg stdcolor.NRGBA) float64 {
				var sum float64
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
					fr, fgr, fb := linear(fill.R), linear(fill.G), linear(fill.B)
					tr, tg, tb := linear(channelPlateau(line, fill.R, 0)), linear(channelPlateau(line, fill.G, 1)), linear(channelPlateau(line, fill.B, 2))
					sum += readStem13With(line, sh, role, base, float64(stemLeft+off), func(x, y int) float64 {
						c := line.RGBAAt(x, y)
						return ((fr-linear(c.R))/(fr-tr) + (fgr-linear(c.G))/(fgr-tg) + (fb-linear(c.B))/(fb-tb)) / 3
					})
				}
				return sum / float64(len(stemOffsets))
			}
			ls, ss := mean(label), mean(labelTwo)
			t.Logf("label stem %.3f px, secondary label stem %.3f px", ls, ss)
			if d := math.Abs(ss - ls); d > 0.02+1e-9 {
				t.Errorf("the secondary label's stem is %.3f px and the label's %.3f px, %.3f px apart", ss, ls, d)
			}
		})
	}
}

// linear is the linear-light value of one encoded sRGB channel, in 0..1.
func linear(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// channelPlateau is the text value of one channel of a line: the value of
// channel ch (0 red, 1 green, 2 blue) farthest from the fill's.
func channelPlateau(img *image.RGBA, fill uint8, ch int) uint8 {
	p := fill
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			v := [3]uint8{c.R, c.G, c.B}[ch]
			if math.Abs(float64(v)-float64(fill)) > math.Abs(float64(p)-float64(fill)) {
				p = v
			}
		}
	}
	return p
}
