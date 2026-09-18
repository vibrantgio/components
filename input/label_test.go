package input_test

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// labelledSize is the plane a labelled control is captured on: wide enough
// for the longest label below and tall enough that the body role's line box,
// which stands taller than the measured 22 px row, is not cut off.
var labelledSize = image.Pt(220, 40)

// TestLabelledControlGolden records or diffs a checkbox and a radio carrying
// their own labels, in both schemes and in every state the label itself
// moves in: set, unset and switched off.
func TestLabelledControlGolden(t *testing.T) {
	shaper := defaultShaper(t)

	cases := []struct {
		name     string
		platform tokens.PlatformColors
		w        func(tokens.PlatformColors) layout.Widget
	}{
		{"checkbox-light-labelled", tokens.PlatformLight, func(p tokens.PlatformColors) layout.Widget {
			return input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Label: "Show startup screen"})
		}},
		{"checkbox-dark-labelled", tokens.PlatformDark, func(p tokens.PlatformColors) layout.Widget {
			return input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Label: "Show startup screen"})
		}},
		{"checkbox-light-labelled-checked", tokens.PlatformLight, func(p tokens.PlatformColors) layout.Widget {
			return input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Checked: true, Label: "Show startup screen"})
		}},
		{"checkbox-light-labelled-disabled", tokens.PlatformLight, func(p tokens.PlatformColors) layout.Widget {
			return input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Disabled: true, Label: "Show startup screen"})
		}},
		{"checkbox-dark-labelled-disabled", tokens.PlatformDark, func(p tokens.PlatformColors) layout.Widget {
			return input.RenderCheckbox(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Disabled: true, Label: "Show startup screen"})
		}},
		{"radio-light-labelled-selected", tokens.PlatformLight, func(p tokens.PlatformColors) layout.Widget {
			return input.RenderRadio(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.RadioRenderState{Selected: true, Label: "Based on the pointer"})
		}},
		{"radio-dark-labelled-disabled", tokens.PlatformDark, func(p tokens.PlatformColors) layout.Widget {
			return input.RenderRadio(shaper, p, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.RadioRenderState{Disabled: true, Label: "Based on the pointer"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			golden.Render(t, tc.name, labelledSize, tc.w(tc.platform))
		})
	}
}

// capturedLabel is the save dialog's own first checkbox label. The gap the
// control spends is the reading itself rather than an origin behind it, so
// the capture's string is what can be compared to it column for column: its
// S carries no left side bearing in the body role's face, and the reference
// reads that same letter.
const capturedLabel = "Show startup screen"

// coveredColumns and coveredRows answer which columns and which rows of img
// carry a drawing, over the clear plane a capture with no surface leaves.
func coveredColumns(img *image.RGBA) []int {
	var cols []int
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if img.RGBAAt(x, y).A != 0 {
				cols = append(cols, x)
				break
			}
		}
	}
	return cols
}

func coveredRows(img *image.RGBA, fromX, toX int) (first, last int) {
	first, last = -1, -1
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := fromX; x < toX && x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A != 0 {
				if first < 0 {
					first = y
				}
				last = y
				break
			}
		}
	}
	return
}

// TestLabelStandsAtTheMeasuredGap asserts the label's first covered column is
// six clear of the glyph's last, which is what the platform draws.
//
// MEASURED, save-dialog-{light,dark}.png: both "Options:" squares end at
// column 279 and both labels' first covered column is 286 — six columns
// clear, in both appearances and both rows.
//
// The gap is spent as the reading, so the capture's own label is what the
// assertion uses: its S bears no column in the body role's face, and the six
// lands the first covered column on the platform's exactly. A label whose
// first glyph does carry a bearing stands one column further out, as it
// would on the platform.
func TestLabelStandsAtTheMeasuredGap(t *testing.T) {
	shaper := defaultShaper(t)
	for _, tc := range []struct {
		name string
		w    layout.Widget
	}{
		{"checkbox", input.RenderCheckbox(shaper, tokens.PlatformLight, tokens.Spacing,
			tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{Label: capturedLabel})},
		{"radio", input.RenderRadio(shaper, tokens.PlatformLight, tokens.Spacing,
			tokens.DefaultTypography.BodyLarge, input.RadioRenderState{Label: capturedLabel})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := golden.Capture(t, labelledSize, tc.w)
			if img == nil {
				return
			}
			cols := coveredColumns(img)
			if len(cols) < 2 {
				t.Fatalf("%s: the capture carries no drawing", tc.name)
			}
			// The glyph and the label are the only two runs of drawing
			// in the row, so the one break between consecutive covered
			// columns is the gap.
			glyphLast, labelFirst := -1, -1
			for i := 1; i < len(cols); i++ {
				if cols[i] != cols[i-1]+1 {
					glyphLast, labelFirst = cols[i-1], cols[i]
					break
				}
			}
			if glyphLast < 0 {
				t.Fatalf("%s: the glyph and its label are not separated by clear columns", tc.name)
			}
			if clear := labelFirst - glyphLast - 1; clear != 6 {
				t.Errorf("%s: %d clear columns between the glyph (last column %d) and its label (first column %d), want 6 as the save dialog measures",
					tc.name, clear, glyphLast, labelFirst)
			}
		})
	}
}

// TestLabelCapBandIsCentredOnTheGlyphsRow asserts the label's cap band is
// centred on the row the glyph stands in, with the platform's rounding.
//
// MEASURED, save-dialog-{light,dark}.png: "Show startup screen" caps run
// y 375–385 against a square of y 372–387 — a band centre of 380.0 against
// the square's 379.5 — and "Stay open after run handler" agrees. The band is
// centred on the square and the rounding falls half a pixel low, never high.
func TestLabelCapBandIsCentredOnTheGlyphsRow(t *testing.T) {
	shaper := defaultShaper(t)
	// Capitals only: the band the reading is taken on runs from the baseline
	// up to the cap height, so a descender or an x-height letter would cover
	// rows beyond it.
	w := input.RenderCheckbox(shaper, tokens.PlatformLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge, input.CheckboxRenderState{Label: "HI"})
	img := golden.Capture(t, labelledSize, w)
	if img == nil {
		return
	}
	row := int(tokens.Comfortable.CheckboxRowHeight)
	glyphFirst, glyphLast := coveredRows(img, 0, row)
	labelFirst, labelLast := coveredRows(img, row+4, labelledSize.X)
	if labelFirst < 0 {
		t.Fatal("the label drew nothing")
	}
	glyphCentre := float64(glyphFirst+glyphLast+1) / 2
	labelCentre := float64(labelFirst+labelLast+1) / 2
	if d := labelCentre - glyphCentre; d < 0 || d > 1 {
		t.Errorf("the label's cap band runs y %d–%d (centre %.1f) against a glyph of y %d–%d (centre %.1f): the band must be centred on the glyph's row, the rounding falling low and never high",
			labelFirst, labelLast, labelCentre, glyphFirst, glyphLast, glyphCentre)
	}
}

// TestLabelIsPartOfThePointerTarget asserts a click on the label operates the
// control, which is what the platform does: the label is part of the control
// and not text standing beside it.
func TestLabelIsPartOfThePointerTarget(t *testing.T) {
	label := "Show startup screen"
	for _, tc := range []struct {
		name string
		make func(*int) rx.Observable[layout.Widget]
	}{
		{"checkbox", func(n *int) rx.Observable[layout.Widget] {
			return input.Checkbox(rx.Of(theme.Default()), input.CheckboxProps{
				Label:    label,
				OnChange: func(_ layout.Context, _ bool) { *n++ },
			})
		}},
		{"radio", func(n *int) rx.Observable[layout.Widget] {
			return input.Radio(rx.Of(theme.Default()), input.RadioProps{
				Label:    label,
				OnChange: func(_ layout.Context, _ bool) { *n++ },
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var operated int
			w := materialize(t, tc.make(&operated))

			r := new(gioinput.Router)
			ops := new(op.Ops)
			var dims layout.Dimensions
			drive := func() {
				ops.Reset()
				gtx := layout.Context{
					Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
					Constraints: layout.Exact(image.Pt(300, 40)),
					Ops:         ops,
					Source:      r.Source(),
				}
				dims = w(gtx)
				r.Frame(ops)
			}
			drive() // register the input area

			row := float32(tokens.Comfortable.CheckboxRowHeight)
			if float32(dims.Size.X) <= row {
				t.Fatalf("a labelled %s measures %v, no wider than the glyph's %v px row", tc.name, dims.Size, row)
			}
			// Well inside the label, clear of the glyph and of the gap.
			pos := f32.Pt(float32(dims.Size.X)-2, row/2)
			r.Queue(
				pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
				pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			)
			drive()

			if operated != 1 {
				t.Errorf("a click at %v, on the %s's label, operated it %d times; want 1", pos, tc.name, operated)
			}
		})
	}
}

// TestSwitchedOffLabelFadesWithItsGlyph asserts the label of a switched-off
// control is drawn in the platform's tertiary label rather than its label
// colour — the same fade the glyph beside it takes, and the colour both
// checkbox labels read on the save dialog's sheet.
func TestSwitchedOffLabelFadesWithItsGlyph(t *testing.T) {
	shaper := defaultShaper(t)
	for _, sc := range switchedOffReadings {
		t.Run(sc.name, func(t *testing.T) {
			img := golden.Capture(t, labelledSize, onSheet(sc.sheet, input.RenderCheckbox(
				shaper, sc.platform, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
				input.CheckboxRenderState{Disabled: true, Label: "Show startup screen", Surface: sc.sheet},
			)))
			if img == nil {
				return
			}
			faded := vgcolor.Flatten(sc.platform.TertiaryLabel, sc.sheet)
			full := vgcolor.Flatten(sc.platform.Label, sc.sheet)

			// The label's darkest pixel against the sheet is the one the
			// reading is taken on: anti-aliasing only moves a glyph's pixels
			// toward the surface, never past the colour it was stroked in.
			row := int(tokens.Comfortable.CheckboxRowHeight)
			best, found := stdcolor.RGBA{}, false
			bestD := -1
			for y := 0; y < labelledSize.Y; y++ {
				for x := row + 4; x < labelledSize.X; x++ {
					c := img.RGBAAt(x, y)
					if d := dist(c, sc.sheet); d > bestD {
						bestD, best, found = d, c, true
					}
				}
			}
			if !found || bestD == 0 {
				t.Fatal("the label drew nothing on the sheet")
			}
			if !nearerTo(best, faded, full) {
				t.Errorf("the switched-off label's strongest pixel is %v, nearer the platform's label %v than its faded %v",
					best, full, faded)
			}
		})
	}
}

// onSheet paints the sheet the control was told it stands on behind it, so
// that the glyphs' anti-aliased coverage composites against a real surface
// instead of a clear plane and a reading can be taken off them.
func onSheet(sheet stdcolor.NRGBA, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, sheet, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// dist is the squared distance between a captured pixel and a named colour,
// in the encoded sRGB the platform composites in.
func dist(got stdcolor.RGBA, c stdcolor.NRGBA) int {
	dr := int(got.R) - int(c.R)
	dg := int(got.G) - int(c.G)
	db := int(got.B) - int(c.B)
	return dr*dr + dg*dg + db*db
}

// TestLiveAndStaticDrawTheSameLabel asserts the two paths draw one control.
// Both are given the same shaper — the theme's, which the live path takes on
// its own — so any difference left is the drawing's.
func TestLiveAndStaticDrawTheSameLabel(t *testing.T) {
	label := "Show startup screen"
	live := golden.Capture(t, labelledSize, materialize(t, input.Checkbox(rx.Of(theme.Default()), input.CheckboxProps{
		Label: label,
	})))
	static := golden.Capture(t, labelledSize, input.RenderCheckbox(
		tokens.DefaultTypography.Shaper(),
		tokens.PlatformLight, tokens.Spacing, tokens.DefaultTypography.BodyLarge,
		input.CheckboxRenderState{Label: label},
	))
	if live == nil || static == nil {
		return
	}
	if n := golden.PixelDiff(live, static); n != 0 {
		t.Errorf("the live and the static labelled checkbox differ in %d pixels; the two paths draw one control", n)
	}
}
