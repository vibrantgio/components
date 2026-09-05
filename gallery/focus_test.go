package main

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/button"
	"github.com/vibrantgio/components/chip"
	"github.com/vibrantgio/components/gallery/inventory"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/theme/tokens"
)

// The focused-specimens sheet's own measurements. Every number here belongs to
// the sheet rather than to a component: a specimen pads itself and nothing
// around it.
const (
	focusSheetW            = 760 // the capture's width in px, at 1 px per dp
	focusPanelPadX unit.Dp = 16  // air a level panel holds left and right
	focusPanelPadY unit.Dp = 14  // air a level panel holds above and below
	focusCaptionW  unit.Dp = 92  // the column the level's name is set in
	focusCellGap   unit.Dp = 14  // space between two specimens
	focusFieldW    unit.Dp = 150 // the width the text field is laid out at
	focusButtonW   unit.Dp = 108 // the width the button is laid out at
	focusTriggerW  unit.Dp = 150 // the width the dropdown trigger is bounded to
)

// focusLevels are the levels the sheet shows a focused control on, in the
// order they stack. Three rather than one because the claim under review is
// that the ring does not move with the level, and a specimen on one surface
// cannot carry a claim about two.
var focusLevels = []struct {
	name  string
	level tokens.ElevationLevel
}{
	{"On the paper", tokens.Level0},
	{"On a card", tokens.Level1},
	{"In a dialog", tokens.Level2},
}

// TestFocusedSpecimensGolden stores one image per scheme of every focusable
// family in this library, focused, standing side by side on each of three
// levels. It is the image the single-colour rule is reviewed against: fifteen
// cells whose rings either agree or visibly do not.
//
// The button is the one cell that may disagree, and only at Filled emphasis:
// its band lies inside a solid primary fill, where no step of the primary ramp
// reads, so it is walked against that fill instead (components/internal/focus,
// RingOn). Every other cell draws the scheme's ring.
//
// One image per scheme rather than one per level, because the claim is about
// what the rows share; a per-level image would show each row agreeing with
// itself and say nothing about the next.
func TestFocusedSpecimensGolden(t *testing.T) {
	for _, sc := range schemes() {
		sheet := focusSheet(t, sc.colors)
		size := measure(sheet, focusSheetW, 1<<20)
		golden.Render(t, "focus-"+sc.name, size, ground(sc.colors, sheet))
	}
}

// focusSheet builds the sheet: one level panel per row, each carrying its own
// name and one focused specimen of every family that can take the keyboard.
func focusSheet(t *testing.T, c tokens.ColorTokens) layout.Widget {
	t.Helper()
	shaper := tokens.DefaultTypography.DeterministicShaper()
	rows := make([]layout.Widget, 0, len(focusLevels))
	for _, lv := range focusLevels {
		rows = append(rows, focusPanel(c, lv.name, lv.level, shaper))
	}
	return inventory.Column(rows)
}

// focusPanel is one level's row: that level's own fill behind a caption and
// the specimens standing on it. The caption stands inside the panel rather
// than beside it — a label naming a surface while sitting on a different one is
// a label about the row and not about the surface.
func focusPanel(c tokens.ColorTokens, name string, level tokens.ElevationLevel, shaper *text.Shaper) layout.Widget {
	specimens := []layout.Widget{
		fixedWidth(focusButtonW, button.Render(shaper, "Button", c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
			button.RenderState{Focused: true, Level: level})),
		fixedWidth(focusFieldW, input.Render(shaper, "Field", c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
			input.RenderState{Focused: true, Level: level})),
		input.RenderCheckbox(c, tokens.Spacing, tokens.Radius,
			input.CheckboxRenderState{Focused: true, Level: level}),
		bounded(120, chip.Render(shaper, "Chip", chip.Assist, nil, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
			chip.RenderState{Focused: true, Level: level})),
		fixedWidth(focusTriggerW, picker.RenderField(shaper, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
			picker.FieldState{Focused: true, Level: level, Options: []string{"Apple", "Banana"}})),
	}
	body := func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(focusCaptionW)
			gtx.Constraints.Max.X = gtx.Dp(focusCaptionW)
			return inventory.LabelAt(gtx, shaper, name, c.Ramps.Neutral.Step(600), 11, font.Font{})
		})}
		for _, w := range specimens {
			cs = append(cs, layout.Rigid(hspace(focusCellGap)), layout.Rigid(w))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
	return panelOn(c.SurfaceAt(level), inset(focusPanelPadX, focusPanelPadY, body))
}

// panelOn paints fill behind w, sized to what w draws.
func panelOn(fill stdcolor.NRGBA, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		m := op.Record(gtx.Ops)
		dims := w(gtx)
		call := m.Stop()
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: dims.Size}.Op())
		call.Add(gtx.Ops)
		return dims
	}
}

func inset(x, y unit.Dp, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: x, Right: x, Top: y, Bottom: y}.Layout(gtx, w)
	}
}

func hspace(d unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Dp(d), 0)}
	}
}

// bounded caps what w may lay itself out to without demanding it: a chip is
// sized to its own content and would report a banner's width if handed one.
func bounded(d unit.Dp, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = 0
		gtx.Constraints.Max.X = gtx.Dp(d)
		return w(gtx)
	}
}

func fixedWidth(d unit.Dp, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Dp(d)
		gtx.Constraints.Max.X = gtx.Dp(d)
		return w(gtx)
	}
}
