// Gallery demonstrates the Components Icon component by rendering one SVG and one
// IVG icon side by side.
//
// Run with: go run ./gallery
package main

import (
	"log"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	svgio "github.com/vibrantgio/svg/driver/gio"
	"github.com/vibrantgio/svg/parser"

	ivgraster "github.com/vibrantgio/ivg/raster/gio"

	"github.com/vibrantgio/components/icon"
)

// circleSVG is a minimal 24×24 filled circle.
const circleSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <circle cx="12" cy="12" r="10" fill="#1a1a2e"/>
</svg>`

// actionInfoIVG is the action-info icon in IVG low-res format.
var actionInfoIVG = []byte{
	0x89, 0x49, 0x56, 0x47, 0x02, 0x0a, 0x00, 0x50, 0x50, 0xb0, 0xb0, 0xc0,
	0x80, 0x58, 0xa0, 0xf5, 0x74, 0x58, 0x58, 0xf5, 0x74, 0x58, 0x80, 0x91,
	0xf5, 0x88, 0xa8, 0xa8, 0xa8, 0xa8, 0x0d, 0x77, 0xa8, 0x58, 0x80, 0x0d,
	0x8b, 0x58, 0x80, 0x58, 0xe3, 0x84, 0xbc, 0xe7, 0x78, 0xe8, 0x7c, 0xe7,
	0x88, 0xe9, 0x98, 0xe3, 0x80, 0x60, 0xe7, 0x78, 0xe9, 0x78, 0xe7, 0x88,
	0xe9, 0x88, 0xe1,
}

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Components — Icon Gallery"),
			app.Size(unit.Dp(240), unit.Dp(120)),
		)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	p := parser.NewParser(parser.IgnoreErrorMode)
	svgIcon, err := p.ParseStream(strings.NewReader(circleSVG))
	if err != nil {
		return err
	}

	ivgWidget, err := ivgraster.Widget(actionInfoIVG, 64, 64)
	if err != nil {
		return err
	}

	r := icon.New()
	r.Register("circle", icon.FromSVG(svgIcon))
	r.Register("info", icon.FromIVG(actionInfoIVG))

	svgWidget := svgio.IconWidget(svgIcon, 64, 64, 1.0)

	var ops op.Ops
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			layout.UniformInset(unit.Dp(16)).Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceAround}.Layout(gtx,
						layout.Rigid(svgWidget),
						layout.Rigid(ivgWidget),
					)
				},
			)
			e.Frame(gtx.Ops)
		}
	}
}
