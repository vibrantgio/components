// Gallery demonstrates the Prism Button component in every visual state:
// light/dark × normal, hovered, focused, pressed, disabled.
//
// Run with: go run ./gallery
package main

import (
	"image"
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/vibrantgio/prism/button"
	"github.com/vibrantgio/spectrum/tokens"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Prism — Button Gallery"),
			app.Size(unit.Dp(640), unit.Dp(540)),
		)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	shaper := tokens.DefaultTypography.Shaper()
	var ops op.Ops
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			frame(gtx, shaper)
			e.Frame(gtx.Ops)
		}
	}
}

type row struct {
	label string
	state button.RenderState
	dark  bool
}

var rows = []row{
	{label: "Normal (light)", state: button.RenderState{}},
	{label: "Hovered (light)", state: button.RenderState{Hovered: true}},
	{label: "Focused (light)", state: button.RenderState{Focused: true}},
	{label: "Pressed (light)", state: button.RenderState{Pressed: true}},
	{label: "Disabled (light)", state: button.RenderState{Disabled: true}},
	{label: "Normal (dark)", state: button.RenderState{}, dark: true},
	{label: "Focused (dark)", state: button.RenderState{Focused: true}, dark: true},
	{label: "Pressed (dark)", state: button.RenderState{Pressed: true}, dark: true},
	{label: "Disabled (dark)", state: button.RenderState{Disabled: true}, dark: true},
}

func frame(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xf1, G: 0xf5, B: 0xf9, A: 0xff},
		clip.Rect{Max: gtx.Constraints.Max}.Op())

	list := &layout.List{Axis: layout.Vertical}
	return list.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
		r := rows[i]
		colors := tokens.DefaultLight
		rowBg := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		if r.dark {
			colors = tokens.DefaultDark
			rowBg = tokens.DefaultDark.Background
		}

		const (
			rowHeightDp = unit.Dp(60)
			padHDp      = unit.Dp(24)
			padVDp      = unit.Dp(8)
			labelWDp    = unit.Dp(200)
			btnWDp      = unit.Dp(200)
		)

		rh := gtx.Dp(rowHeightDp)
		size := image.Pt(gtx.Constraints.Max.X, rh)

		// Row background
		paint.FillShape(gtx.Ops, rowBg, clip.Rect{Max: size}.Op())

		// Layout row content
		layout.Inset{
			Left:   padHDp,
			Right:  padHDp,
			Top:    padVDp,
			Bottom: padVDp,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				// State label column
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lw := gtx.Dp(labelWDp)
					gtx.Constraints.Min.X = lw
					gtx.Constraints.Max.X = lw

					m := op.Record(gtx.Ops)
					paint.ColorOp{Color: colors.Text}.Add(gtx.Ops)
					mat := m.Stop()

					lbl := widget.Label{MaxLines: 1}
					return lbl.Layout(gtx, shaper, font.Font{}, unit.Sp(14), r.label, mat)
				}),
				// Button column
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					bw := gtx.Dp(btnWDp)
					gtx.Constraints.Min.X = bw
					gtx.Constraints.Max.X = bw
					return button.Render(
						shaper, "Click me",
						colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypeScale,
						r.state,
					)(gtx)
				}),
			)
		})

		return layout.Dimensions{Size: size}
	})
}
