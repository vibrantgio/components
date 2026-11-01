package golden_test

import (
	"fmt"
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

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// platformSmallStems are the platform's stem rows as stored, in px at 1x,
// for text darker than its fill and text lighter than it. They are
// MEASURED in the reference's section "What the platform's small text
// measures at 1x", from Finder's file names.
var platformSmallStems = map[float32][2]float64{
	10: {0.97, 1.16},
	11: {1.06, 1.29},
	13: {1.26, 1.52},
}

// platformStem28 is the platform's stem row at 28 pt, MEASURED in the
// reference's section "What the stem measures at the title sizes and at
// mid contrast", for darker and lighter text.
var platformStem28 = [2]float64{2.15, 2.44}

// derivedStem is the DERIVED line through the 13 pt and 28 pt rows. No
// stored capture checks a size between the two, so the line holds above
// 13 pt only.
func derivedStem(size float32, lighter int) float64 {
	a, b := platformSmallStems[13][lighter], platformStem28[lighter]
	return a + float64(size-13)*(b-a)/15
}

// TestTheStoredStemMeetsThePlatformAtEachSize: in both schemes, the golden
// test window draws "Vibrant Gio" in the label colour and in the secondary
// label colour on the window background, and in white on the default
// button's fill. The stem is read as stored, per channel, over the
// channels in which the text and the fill differ, and it is the mean of
// four readings at the offsets 0, ¼, ½ and ¾ px. At 10, 11 and 13 pt it is
// within 0.06 px of the platform's measured row. Above 13 pt it is within
// 0.1 px of the DERIVED line through the 13 pt and 28 pt rows.
func TestTheStoredStemMeetsThePlatformAtEachSize(t *testing.T) {
	white := stdcolor.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	for _, sc := range []struct {
		name string
		set  tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		typ, err := golden.StaticTheme(sc.set).Typography.First()
		if err != nil {
			t.Fatalf("typography: %v", err)
		}
		sh := typ.DeterministicShaper()
		for _, tc := range []struct {
			name     string
			fg, fill stdcolor.NRGBA
		}{
			{"label colour", sc.set.Label, sc.set.WindowBackground},
			{"secondary label colour", sc.set.SecondaryLabel, sc.set.WindowBackground},
			{"white on the default button", white, sc.set.DefaultButtonFill},
		} {
			fill := tc.fill
			fill.A = 0xff
			fg := color.Flatten(tc.fg, fill)
			lighter := 0
			if color.RelativeLuminance(fg) >= color.RelativeLuminance(fill) {
				lighter = 1
			}
			for _, size := range []float32{10, 11, 13, 17, 22, 28} {
				t.Run(fmt.Sprintf("%s/%s/%vpt", sc.name, tc.name, size), func(t *testing.T) {
					role := typ.SystemFont
					role.Size = size
					w, h := stemLeft+int(8*size), int(2*size)
					var stems []float64
					for _, off := range stemOffsets {
						var dims layout.Dimensions
						line := golden.Capture(t, image.Pt(w, h), func(gtx layout.Context) layout.Dimensions {
							paint.FillShape(gtx.Ops, fill, clip.Rect{Max: image.Pt(w, h)}.Op())
							gtx = typeset.WithFill(gtx, fill)
							tr := op.Affine(f32.Affine2D{}.Offset(f32.Pt(stemLeft+off, 0))).Push(gtx.Ops)
							defer tr.Pop()
							dims = typeset.Text(gtx, sh, stemWord, role, fg, 1)
							return dims
						})
						base := dims.Size.Y - dims.Baseline
						f := typeset.FontOn(typeset.WithFill(gtxNone, fill), role, font.Normal, fg)
						stems = append(stems, readStem13With(line, sh, f, role, base, float64(stemLeft+off), storedCoverage(line, fill, fg)))
					}
					mean := (stems[0] + stems[1] + stems[2] + stems[3]) / 4
					want, tol := derivedStem(size, lighter), 0.1
					if row, ok := platformSmallStems[size]; ok {
						want, tol = row[lighter], 0.06
					}
					t.Logf("stems %.2f %.2f %.2f %.2f, mean %.3f, platform %.3f", stems[0], stems[1], stems[2], stems[3], mean, want)
					if d := math.Abs(mean - want); d > tol+1e-9 {
						t.Errorf("the mean stem is %.3f px, %.3f px from the platform's %.3f", mean, d, want)
					}
				})
			}
		}
	}
}

// storedCoverage returns the coverage of a pixel of img as stored: per
// channel (fill − pixel) / (fill − text), averaged over the channels in
// which fg and fill differ.
func storedCoverage(img *image.RGBA, fill, fg stdcolor.NRGBA) func(x, y int) float64 {
	f := [3]float64{float64(fill.R), float64(fill.G), float64(fill.B)}
	g := [3]float64{float64(fg.R), float64(fg.G), float64(fg.B)}
	return func(x, y int) float64 {
		c := img.RGBAAt(x, y)
		p := [3]float64{float64(c.R), float64(c.G), float64(c.B)}
		sum, n := 0.0, 0
		for i := range 3 {
			if f[i] == g[i] {
				continue
			}
			sum += (f[i] - p[i]) / (f[i] - g[i])
			n++
		}
		if n == 0 {
			return 0
		}
		return sum / float64(n)
	}
}
