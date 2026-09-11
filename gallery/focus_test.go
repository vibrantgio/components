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
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// The focused-specimens sheet's own measurements. Every number here belongs to
// the sheet rather than to a component: a specimen pads itself and nothing
// around it.
const (
	focusSheetW            = 760 // the capture's width in px, at 1 px per dp
	focusPanelPadX unit.Dp = 16  // air a surface panel holds left and right
	focusPanelPadY unit.Dp = 14  // air a surface panel holds above and below
	focusCaptionW  unit.Dp = 92  // the column the surface's name is set in
	focusCellGap   unit.Dp = 14  // space between two specimens
	focusFieldW    unit.Dp = 150 // the width the text field is laid out at
	focusButtonW   unit.Dp = 108 // the width the button is laid out at
	focusTriggerW  unit.Dp = 150 // the width the dropdown trigger is bounded to
)

// focusSurfaces are the fills the sheet shows a focused control on. Three
// rather than one because the claim under review is that the ring does not
// move with the surface, and a specimen on one fill cannot carry a claim
// about three. The focus ring carries a coverage rather than a colour, so
// every cell has to say what it is composited onto or the ring lands on the
// window's plane whatever the panel under it is painted in.
func focusSurfaces(c tokens.PlatformColors) []struct {
	name string
	fill stdcolor.NRGBA
} {
	return []struct {
		name string
		fill stdcolor.NRGBA
	}{
		{"On the content", c.ControlBackground},
		{"On a card", c.CardFill},
		{"In the chrome", c.SidebarMaterial},
	}
}

// TestFocusedSpecimensGolden stores one image per appearance of every
// focusable family in this library, focused, standing side by side on each of
// three surfaces. It is the image the single-colour rule is reviewed against:
// fifteen cells whose rings either agree or visibly do not.
//
// The ring is the platform's own keyboardFocusIndicator, which carries a
// coverage, so each cell flattens it onto the fill it stands on; what the
// sheet shows is that one name lands consistently on three fills rather than
// three names being chosen.
//
// One image per appearance rather than one per surface, because the claim is
// about what the rows share; a per-surface image would show each row agreeing
// with itself and say nothing about the next.
func TestFocusedSpecimensGolden(t *testing.T) {
	for _, sc := range schemes() {
		sheet := focusSheet(t, sc.colors)
		size := measure(sheet, focusSheetW, 1<<20)
		golden.Render(t, "focus-"+sc.name, size, onBackground(sc.colors, sheet))
	}
}

// focusSheet builds the sheet: one surface panel per row, each carrying its own
// name and one focused specimen of every family that can take the keyboard.
func focusSheet(t *testing.T, c tokens.PlatformColors) layout.Widget {
	t.Helper()
	shaper := tokens.DefaultTypography.DeterministicShaper()
	surfaces := focusSurfaces(c)
	rows := make([]layout.Widget, 0, len(surfaces))
	for _, lv := range surfaces {
		rows = append(rows, focusPanel(c, lv.name, lv.fill, shaper))
	}
	return inventory.Column(rows)
}

// focusPanel is one surface's row: that fill behind a caption and the
// specimens standing on it. The caption stands inside the panel rather than
// beside it — a label naming a surface while sitting on a different one is a
// label about the row and not about the surface.
func focusPanel(c tokens.PlatformColors, name string, fill stdcolor.NRGBA, shaper *text.Shaper) layout.Widget {
	specimens := []layout.Widget{
		fixedWidth(focusButtonW, button.Render(shaper, "Button", c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
			button.RenderState{Focused: true, Surface: fill})),
		fixedWidth(focusFieldW, input.Render(shaper, "Field", c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
			input.RenderState{Focused: true, Surface: fill})),
		input.RenderCheckbox(c, tokens.Spacing, tokens.Radius,
			input.CheckboxRenderState{Focused: true, Surface: fill}),
		bounded(120, chip.Render(shaper, "Chip", chip.Assist, nil, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
			chip.RenderState{Focused: true, Surface: fill})),
		// The trigger states no surface: it carries a fill of its own, so
		// its ring composites onto that rather than onto the panel.
		fixedWidth(focusTriggerW, picker.RenderField(shaper, c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
			picker.FieldState{Focused: true, Options: []string{"Apple", "Banana"}})),
	}
	body := func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(focusCaptionW)
			gtx.Constraints.Max.X = gtx.Dp(focusCaptionW)
			return inventory.LabelAt(gtx, shaper, name, vgcolor.Flatten(c.SecondaryLabel, fill), 11, font.Font{})
		})}
		for _, w := range specimens {
			cs = append(cs, layout.Rigid(hspace(focusCellGap)), layout.Rigid(w))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
	return panelOn(fill, inset(focusPanelPadX, focusPanelPadY, body))
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
