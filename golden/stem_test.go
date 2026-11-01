package golden_test

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	fx "golang.org/x/image/math/fixed"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// The platform's System font row at 13 pt, as stored, from the reference's
// section "What the same text measures in a platform window and a library
// window": the lowercase stem of "Vibrant Gio" in px, and the other stem
// where the platform gives two.
var platformStem13 = map[bool][2]float64{
	false: {1.26, 1.26},
	true:  {1.51, 1.57},
}

// stemOffsets are the four horizontal offsets the word is set at, in px.
var stemOffsets = []float32{0, 0.25, 0.5, 0.75}

const (
	stemWord  = "Vibrant Gio"
	stemLineH = 24
	stemLeft  = 20
	stemW     = 120
)

// TestTheStemFollowsTheTextAgainstItsFill: the golden test window of each
// scheme draws "Vibrant Gio" in the System font, in the label colour on
// controlBackground. The text is told that fill. Text darker than the fill
// in light and lighter than it in dark each meet their own row. The stem is
// the mean of four readings at the offsets 0, ¼, ½ and ¾ px. It is within
// 0.06 px of the platform's row, or of the nearer row where the platform
// gives two.
func TestTheStemFollowsTheTextAgainstItsFill(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(tc.name, func(t *testing.T) {
			typ, err := golden.StaticTheme(tc.set).Typography.First()
			if err != nil {
				t.Fatalf("typography: %v", err)
			}
			sh := typ.DeterministicShaper()
			role := typ.SystemFont
			fill := tc.set.ControlBackground
			foreground := color.Flatten(tc.set.Label, fill)
			var stems []float64
			for _, off := range stemOffsets {
				var dims layout.Dimensions
				line := golden.Capture(t, image.Pt(stemW, stemLineH), func(gtx layout.Context) layout.Dimensions {
					paint.FillShape(gtx.Ops, fill, clip.Rect{Max: image.Pt(stemW, stemLineH)}.Op())
					gtx = typeset.WithFill(gtx, fill)
					tr := op.Affine(f32.Affine2D{}.Offset(f32.Pt(stemLeft+off, 0))).Push(gtx.Ops)
					defer tr.Pop()
					dims = typeset.Text(gtx, sh, stemWord, role, foreground, 1)
					return dims
				})
				base := dims.Size.Y - dims.Baseline
				stems = append(stems, readStem13(line, sh, role, base, float64(stemLeft+off), fill.R, plateau(line, fill.R)))
			}
			mean := (stems[0] + stems[1] + stems[2] + stems[3]) / 4
			want := platformStem13[tc.set.Dark()]
			t.Logf("stems %.2f %.2f %.2f %.2f, mean %.3f", stems[0], stems[1], stems[2], stems[3], mean)
			if d := min(math.Abs(mean-want[0]), math.Abs(mean-want[1])); d > 0.06+1e-9 {
				t.Errorf("the mean stem is %.3f px, %.3f px from the platform's %.2f", mean, d, want[0])
			}
		})
	}
}

// plateau is the text colour of a line: the red channel farthest from the
// fill's.
func plateau(img *image.RGBA, fill uint8) uint8 {
	p := fill
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if c := img.RGBAAt(x, y).R; math.Abs(float64(c)-float64(fill)) > math.Abs(float64(p)-float64(fill)) {
				p = c
			}
		}
	}
	return p
}

// readStem13 reads one 13 pt line the reference's way: the left stem of the
// n in "Vibrant", four rows over the line's own baseline. The stem is the
// coverage summed over the run of covered columns.
func readStem13(img *image.RGBA, sh *text.Shaper, role tokens.TextStyle, base int, left float64, fill, foreground uint8) float64 {
	fv, tv := float64(fill), float64(foreground)
	return readStem13With(img, sh, role, base, left, func(x, y int) float64 {
		c := img.RGBAAt(x, y)
		return ((fv-float64(c.R))/(fv-tv) + (fv-float64(c.G))/(fv-tv) + (fv-float64(c.B))/(fv-tv)) / 3
	})
}

// readStem13With is readStem13 with the coverage of a pixel given by cov.
func readStem13With(img *image.RGBA, sh *text.Shaper, role tokens.TextStyle, base int, left float64, cov func(x, y int) float64) float64 {
	sh.LayoutString(text.Parameters{
		Font:     typeset.Font(role, font.Normal),
		PxPerEm:  fx.Int26_6(role.Size * 64),
		MaxLines: 1, MaxWidth: 10000,
	}, stemWord)
	var nx, nadv float64
	for i := 0; ; i++ {
		g, ok := sh.NextGlyph()
		if !ok {
			break
		}
		if i == 5 {
			nx = float64(g.X)/64 + left
			nadv = float64(g.Advance) / 64
		}
	}
	row := base - 4
	xa, xb := int(math.Floor(nx+0.56)), int(math.Floor(nx+nadv/2))
	for xa <= xb && cov(xa, row) < 0.005 {
		xa++
	}
	e := xa
	peak := false
	for e+1 <= xa+3 {
		c, n := cov(e, row), cov(e+1, row)
		if n < 0.01 {
			break
		}
		if n < c {
			peak = true
		} else if peak {
			break
		}
		e++
	}
	stem := 0.0
	for x := xa; x <= e; x++ {
		stem += cov(x, row)
	}
	return stem
}
