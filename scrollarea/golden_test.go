package scrollarea

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// ruler draws a band of evenly spaced marks, so where the content has been
// scrolled to is readable off the image and a dissolved edge is obviously a
// dissolve and not a clip.
func ruler(c tokens.ColorTokens, width int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		const pitch, mark = 22, 14
		for x := 0; x < width; x += pitch {
			paint.FillShape(gtx.Ops, c.Primary,
				clip.Rect(image.Rect(x, 8, min(x+mark, width), 36)).Op())
		}
		return layout.Dimensions{Size: image.Pt(width, 44)}
	}
}

// TestFadeGolden records or diffs everything a scroll area says about its
// edges, stacked in one image: content that fits (no dissolve, no bar, drawn
// exactly as it would be with no area at all), content resting at its start
// (the trailing edge dissolves, the leading one does not), content in the
// middle (both edges), and content at its end (only the leading edge). Each
// overflowing band also carries the bar, which a first-frame capture catches
// at full opacity.
func TestFadeGolden(t *testing.T) {
	const viewport, content, band = 280, 900, 60
	c := tokens.DefaultLight
	style := FromTokens(c)
	bar := scrollbar.FromTokens(c)

	bands := []struct {
		width  int
		offset int
	}{
		{width: 200},
		{width: content, offset: 0},
		{width: content, offset: (content - band) / 3},
		{width: content, offset: content},
	}
	states := make([]*State, len(bands))

	// Measure first, off-screen, so every offset below is clamped against
	// real geometry and the drawn frame is the settled one.
	var scratch op.Ops
	for i, b := range bands {
		states[i] = NewState()
		style.LayoutScrollbar(testContext(&scratch, image.Pt(viewport, band)), states[i], bar, ruler(c, b.width))
		states[i].SetOffset(b.offset)
	}

	golden.Render(t, "fade-light", image.Pt(viewport, len(bands)*band), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.Surface, clip.Rect{Max: gtx.Constraints.Max}.Op())
		for i, b := range bands {
			bgtx := gtx
			bgtx.Constraints = layout.Constraints{Max: image.Pt(viewport, band)}
			tr := op.Offset(image.Pt(0, i*band)).Push(gtx.Ops)
			style.LayoutScrollbar(bgtx, states[i], bar, ruler(c, b.width))
			tr.Pop()
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}
