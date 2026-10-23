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
func ruler(p tokens.PlatformColors, width int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		const pitch, mark = 22, 14
		for x := 0; x < width; x += pitch {
			paint.FillShape(gtx.Ops, p.Label,
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
// overflowing band also carries the bar at the platform's Always setting.
// The bar stands in a lane of its own below the content, at the area's
// bottom edge.
func TestFadeGolden(t *testing.T) {
	const viewport, content, band = 280, 900, 60
	p := tokens.PlatformLight
	style := FromTokens(p)
	bar := scrollbar.FromTokens(p, p.ControlBackground).Always()

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
		style.LayoutScrollbar(testContext(&scratch, image.Pt(viewport, band)), states[i], bar, ruler(p, b.width))
		states[i].SetOffset(b.offset)
	}

	golden.Render(t, "fade-light", image.Pt(viewport, len(bands)*band), func(gtx layout.Context) layout.Dimensions {
		// The backdrop is the surface FromTokens fades into: an area whose
		// fade names a colour its host does not paint reads as a band laid
		// across the content, so the image only shows a dissolve if the two
		// are asked for together.
		paint.FillShape(gtx.Ops, p.ControlBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
		for i, b := range bands {
			bgtx := gtx
			bgtx.Constraints = layout.Constraints{Max: image.Pt(viewport, band)}
			tr := op.Offset(image.Pt(0, i*band)).Push(gtx.Ops)
			style.LayoutScrollbar(bgtx, states[i], bar, ruler(p, b.width))
			tr.Pop()
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

// TestStandingBarHasALane pins where the area draws its bar. At Always the
// bar stands, so the area is taller than its content by the lane and the
// thumb is drawn in that lane, inside the area's bottom edge. A bar that
// fades takes no lane, and content that fits takes none at Always either.
func TestStandingBarHasALane(t *testing.T) {
	const viewport, height = 200, 40
	p := tokens.PlatformLight
	style := FromTokens(p)
	style.Fade = 0
	standing := scrollbar.FromTokens(p, p.ControlBackground).Always()
	fading := scrollbar.FromTokens(p, p.ControlBackground)
	lane := int(standing.Width())

	sizes := func(bar scrollbar.Style, width int) image.Point {
		var ops op.Ops
		return style.LayoutScrollbar(testContext(&ops, image.Pt(viewport, 1000)), NewState(), bar, block(image.Pt(width, height))).Size
	}
	if got, want := sizes(standing, 3*viewport), image.Pt(viewport, height+lane); got != want {
		t.Errorf("overflowing area at Always is %v, want %v: taller than its content by the %d px lane", got, want, lane)
	}
	if got, want := sizes(fading, 3*viewport), image.Pt(viewport, height); got != want {
		t.Errorf("overflowing area with a fading bar is %v, want %v: no lane", got, want)
	}
	if got, want := sizes(standing, viewport/2), image.Pt(viewport/2, height); got != want {
		t.Errorf("fitting area at Always is %v, want %v: a lane exists only for a bar that is drawn", got, want)
	}

	// The content draws nothing, so every pixel that differs from the
	// backdrop belongs to the bar.
	state := NewState()
	var dims layout.Dimensions
	img := golden.Capture(t, image.Pt(viewport, height+2*lane), func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, p.ControlBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
		gtx.Constraints = layout.Constraints{Max: image.Pt(viewport, 1000)}
		dims = style.LayoutScrollbar(gtx, state, standing, block(image.Pt(3*viewport, height)))
		return dims
	})
	bottom := dims.Size.Y
	back := img.RGBAAt(0, img.Bounds().Dy()-1)
	drawn, outside := 0, 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < viewport; x++ {
			if img.RGBAAt(x, y) == back {
				continue
			}
			if y >= height && y < bottom {
				drawn++
			} else {
				outside++
			}
		}
	}
	if drawn == 0 {
		t.Errorf("no thumb drawn in the lane, rows %d to %d", height, bottom-1)
	}
	if outside != 0 {
		t.Errorf("%d bar pixels drawn outside the lane: over the content or past the area's bottom edge at row %d", outside, bottom)
	}
}
