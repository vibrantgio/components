package button_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/theme/tokens"
)

const symbolTitle = "New card"

var symbolTitleSize = image.Pt(160, 40)

// natural draws w at the width it needs: a button's box is a budget, and
// zero room leaves it the width of its own parts.
func natural(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Constraints{Max: image.Pt(0, gtx.Constraints.Max.Y)}
		return w(gtx)
	}
}

func plusButton(t *testing.T, p tokens.PlatformColors, icon func(layout.Context, int, color.NRGBA), s button.RenderState) layout.Widget {
	return button.RenderWithSymbol(defaultShaper(t), symbolTitle, icon,
		p, tokens.Spacing, tokens.RadiusScale{}, tokens.DefaultTypography.SystemFont, tokens.Comfortable, s)
}

// TestButtonWithSymbolGolden records or diffs a button whose title is a
// symbol and text, in both schemes.
func TestButtonWithSymbolGolden(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(sc.name, func(t *testing.T) {
			w := plusButton(t, sc.p, icons.Mark(icons.Plus), button.RenderState{Emphasis: button.Tonal})
			golden.Render(t, "symbol-title-tonal-"+sc.name, symbolTitleSize, onWindowSurface(sc.p, natural(w)))
		})
	}
}

// TestButtonWithoutSymbolDrawsTheTextButton asserts that the symbol path
// with no symbol draws the text button's pixels, at the button's natural
// width and at a width the caller gives it.
func TestButtonWithoutSymbolDrawsTheTextButton(t *testing.T) {
	p := tokens.PlatformLight
	for _, wrap := range []struct {
		name string
		f    func(layout.Widget) layout.Widget
	}{{"natural", natural}, {"given", func(w layout.Widget) layout.Widget { return w }}} {
		t.Run(wrap.name, func(t *testing.T) {
			text := button.Render(defaultShaper(t), symbolTitle, p, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable, button.RenderState{Focused: true})
			none := button.RenderWithSymbol(defaultShaper(t), symbolTitle, nil, p, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable, button.RenderState{Focused: true})
			a := golden.Capture(t, symbolTitleSize, onWindowSurface(p, wrap.f(text)))
			b := golden.Capture(t, symbolTitleSize, onWindowSurface(p, wrap.f(none)))
			if a == nil || b == nil {
				return
			}
			for y := 0; y < symbolTitleSize.Y; y++ {
				for x := 0; x < symbolTitleSize.X; x++ {
					if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
						t.Fatalf("pixel (%d,%d) is %v with no symbol, want the text button's %v", x, y, b.RGBAAt(x, y), a.RGBAAt(x, y))
					}
				}
			}
		})
	}
}

// columnCovered reports whether column x of img holds a pixel that differs from
// fill over rows y0 to y1.
func columnCovered(img *image.RGBA, fill color.RGBA, x, y0, y1 int) bool {
	for y := y0; y < y1; y++ {
		if img.RGBAAt(x, y) != fill {
			return true
		}
	}
	return false
}

// TestSymbolStandsBeforeTheTitle reads the symbol's place off the drawn
// pixels: four clear columns between the symbol's last covered column and
// the text's first, the symbol's covered rows centred on the button's
// centre row, and the button wider than the text button by the symbol and
// the gap.
func TestSymbolStandsBeforeTheTitle(t *testing.T) {
	p := tokens.PlatformLight
	s := button.RenderState{Emphasis: button.Tonal}
	with := plusButton(t, p, icons.Mark(icons.Plus), s)
	img := golden.Capture(t, symbolTitleSize, onWindowSurface(p, natural(with)))
	if img == nil {
		return
	}
	var ops op.Ops
	gtx := layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(0, 100)}, Ops: &ops}
	withDims := with(gtx)
	textDims := button.Render(defaultShaper(t), symbolTitle, p, tokens.Spacing, tokens.RadiusScale{}, tokens.DefaultTypography.SystemFont, tokens.Comfortable, s)(gtx)

	fill := img.RGBAAt(2, withDims.Size.Y/2)
	h := withDims.Size.Y
	// The symbol's covered columns, and the text's first covered column.
	symFirst, symLast, textFirst := -1, -1, -1
	x := 0
	for ; x < withDims.Size.X && !columnCovered(img, fill, x, 0, h); x++ {
	}
	symFirst = x
	for ; x < withDims.Size.X && columnCovered(img, fill, x, 0, h); x++ {
	}
	symLast = x - 1
	for ; x < withDims.Size.X && !columnCovered(img, fill, x, 0, h); x++ {
	}
	textFirst = x
	if symFirst <= 0 || textFirst >= withDims.Size.X {
		t.Fatalf("no symbol and text found: symbol from %d to %d, text from %d", symFirst, symLast, textFirst)
	}
	if gap := textFirst - symLast - 1; gap != 4 {
		t.Errorf("%d clear columns between the symbol (last column %d) and the text (first column %d), want the measured 4", gap, symLast, textFirst)
	}

	// The symbol's covered rows.
	top, bottom := -1, -1
	for y := 0; y < h; y++ {
		for x := symFirst; x <= symLast; x++ {
			if img.RGBAAt(x, y) != fill {
				if top < 0 {
					top = y
				}
				bottom = y
				break
			}
		}
	}
	if d := (top + bottom + 1) - h; d < -1 || d > 1 {
		t.Errorf("the symbol covers rows %d to %d in a button %d tall; its centre row is not the button's", top, bottom, h)
	}

	if extra := withDims.Size.X - textDims.Size.X; extra != textFirst-textFirstOf(t, p, s) {
		t.Errorf("the button with a symbol is %d px wider than the text button; want the symbol and the gap", extra)
	}
	if withDims.Size.Y != textDims.Size.Y {
		t.Errorf("the button with a symbol is %d px tall, the text button %d", withDims.Size.Y, textDims.Size.Y)
	}
}

// textFirstOf is the text button's first covered column at its natural
// width.
func textFirstOf(t *testing.T, p tokens.PlatformColors, s button.RenderState) int {
	t.Helper()
	w := button.Render(defaultShaper(t), symbolTitle, p, tokens.Spacing, tokens.RadiusScale{}, tokens.DefaultTypography.SystemFont, tokens.Comfortable, s)
	img := golden.Capture(t, symbolTitleSize, onWindowSurface(p, natural(w)))
	if img == nil {
		return 0
	}
	h := int(tokens.Comfortable.ControlHeight)
	fill := img.RGBAAt(2, h/2)
	x := 0
	for ; x < symbolTitleSize.X && !columnCovered(img, fill, x, 0, h); x++ {
	}
	return x
}
