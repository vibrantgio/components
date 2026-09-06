package toast

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/vibrantgio/components/golden"
	vcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

const intFrameW, intFrameH = 320, 240

var intFrame = image.Pt(intFrameW, intFrameH)

func intTok() resolvedTokens {
	return resolvedTokens{
		color:   tokens.DefaultLight,
		spacing: tokens.Spacing,
		radius:  tokens.RadiusScale{},
		style:   tokens.DefaultTypography.LabelMedium,
	}
}

// paint1 draws one toast at the frame's origin, which is where a placement
// would offset it to, and returns the frame it painted.
func paint1(t *testing.T, props Props, tok resolvedTokens) *image.RGBA {
	t.Helper()
	shaper := tokens.DefaultTypography.DeterministicShaper()
	return golden.Capture(t, intFrame, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Constraints{
			Min: image.Pt(0, gtx.Dp(unit.Dp(MinHeightDp))),
			Max: intFrame,
		}
		return draw(gtx, shaper, props, tok)
	})
}

// surfaceFill is the flat, opaque colour draw fills every toast with,
// whatever its status role: the inverse surface. It cannot tell two toasts
// apart — that is what roleEdge is for.
func surfaceFill(tok resolvedTokens) color.NRGBA {
	return Fill(tok.color)
}

// roleEdge is the colour of the leading edge a toast of the given status
// role paints — the one place on an otherwise identical surface that says
// which role is speaking.
func roleEdge(r Role, tok resolvedTokens) color.NRGBA {
	return Edge(tok.color, r)
}

// toastBounds returns the rectangle enclosing every pixel img paints in
// any of cs. A toast's surface is one flat, opaque, axis-aligned rectangle
// at the zero radius these tests render with, so its extent can be read
// back off the image: pass the fill together with the role's edge for the
// whole rectangle, or the edge alone to find the edge inside it.
//
// The match carries a tolerance of one step per channel, because the fill
// reaches the framebuffer through a rasteriser and demanding the exact byte
// would be asserting its arithmetic rather than the geometry. Nothing is
// grown back: the surface carries no outline since the inverse fill
// separates it, so the fill and the edge reach the outermost row and column
// themselves.
func toastBounds(img *image.RGBA, cs ...color.NRGBA) image.Rectangle {
	var box image.Rectangle
	r := img.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			got := img.RGBAAt(x, y)
			hit := false
			for _, c := range cs {
				if nearColor(got, c) {
					hit = true
					break
				}
			}
			if !hit {
				continue
			}
			px := image.Rect(x, y, x+1, y+1)
			if box.Empty() {
				box = px
				continue
			}
			box = box.Union(px)
		}
	}
	return box
}

func nearColor(got color.RGBA, want color.NRGBA) bool {
	off := func(a, b uint8) int {
		if a > b {
			return int(a) - int(b)
		}
		return int(b) - int(a)
	}
	return got.A == want.A &&
		off(got.R, want.R) <= 1 && off(got.G, want.G) <= 1 && off(got.B, want.B) <= 1
}

// TestLeadingEdgeIsWiderThanTheHairlineBandAndNarrowerThanItsOwnAir measures
// the role edge on a rendered toast and holds it between the two bounds the
// width was judged against.
//
// The floor is that a mark identified by its colour cannot be drawn at the
// width the desktop keeps for hairlines, separators and insets — one to
// three pixels. The ceiling is the toast's own air: the message stands one
// horizontal pad clear of the edge, and an edge as wide as that gap reads
// as a panel the message sits beside rather than as the leading edge. So the
// assertion is not "8 px" for its own sake — it is that the mark is at
// least twice the platform's hairline band and still narrower than the air
// it holds the text off by, with the pixel value logged so a later change
// of scale can be read off a test run.
func TestLeadingEdgeIsWiderThanTheHairlineBandAndNarrowerThanItsOwnAir(t *testing.T) {
	tok := intTok()
	// The widest band the platform draws when it does not want the mark
	// looked at: a pane stroke, a separator hairline, a scroll thumb's
	// inset. A role edge has to clear it by a margin, not by a pixel.
	const hairlineBand = 3

	for _, r := range []Role{Info, Success, Warning, Error} {
		img := paint1(t, Props{Role: r, Text: "Rescanned: 2 notes"}, tok)
		edge := toastBounds(img, roleEdge(r, tok))
		if edge.Empty() {
			t.Fatalf("role %d painted no leading edge", r)
		}
		if edge.Dx() <= 2*hairlineBand {
			t.Errorf("role %d edge is %d px wide; a mark read by its colour cannot be drawn at the %d px the platform keeps for hairlines",
				r, edge.Dx(), hairlineBand)
		}
		air := int(tok.spacing.S3) // the message's inset from the edge
		if edge.Dx() >= air {
			t.Errorf("role %d edge is %d px wide against %d px of air before the message; an edge as wide as its own air reads as a panel",
				r, edge.Dx(), air)
		}
		// The air is real, not just arithmetic: find the first column right
		// of the edge carrying anything that is neither the fill nor the
		// edge, which is the message's first drawn pixel.
		fill := surfaceFill(tok)
		firstDrawn := -1
		for x := edge.Max.X; x < edge.Max.X+4*air && firstDrawn < 0; x++ {
			for y := edge.Min.Y; y < edge.Max.Y; y++ {
				got := img.RGBAAt(x, y)
				if !nearColor(got, fill) && !nearColor(got, roleEdge(r, tok)) {
					firstDrawn = x
					break
				}
			}
		}
		if firstDrawn < 0 {
			t.Fatalf("role %d: no message pixels found beside the edge", r)
		}
		if gap := firstDrawn - edge.Max.X; gap <= edge.Dx() {
			t.Errorf("role %d: %d px of air between the edge and the message against a %d px edge; the mark must not out-measure the space it keeps",
				r, gap, edge.Dx())
		}
		t.Logf("role %d: edge %d px wide, message starts %d px past it", r, edge.Dx(), firstDrawn-edge.Max.X)
	}
}

// TestLeadingEdgeReadsOnTheSurfaceInBothSchemes is the colour half of the
// same claim: the edge is the only thing on a toast that says which status
// role this is, so it owes the inverse surface the floor edgeFloor names, in
// both schemes and at every role. The step each scheme lands on is logged
// rather than asserted — which step answers is the ramp's business, not this
// package's — but the contrast it reaches is this package's, because it is
// what the component asked for.
func TestLeadingEdgeReadsOnTheSurfaceInBothSchemes(t *testing.T) {
	for _, sc := range []struct {
		name string
		c    tokens.ColorTokens
	}{{"light", tokens.DefaultLight}, {"dark", tokens.DefaultDark}} {
		for _, r := range []Role{Info, Success, Warning, Error} {
			edge := Edge(sc.c, r)
			got := vcolor.ContrastRatio(edge, sc.c.InverseSurface)
			if got < edgeFloor {
				t.Errorf("%s scheme, role %d: edge %v on the surface measures %.2f:1; want at least %.1f:1",
					sc.name, r, edge, got, edgeFloor)
			}
			t.Logf("%s scheme, role %d: edge %v at %.2f:1", sc.name, r, edge, got)
		}
	}
}

// TestTheFillSaysNothingAboutTheRole pins the half of the Language the fill
// carries: level 2 is where a toast is placed, not what it is filled with,
// so every role fills identically and only the leading edge differs.
func TestTheFillSaysNothingAboutTheRole(t *testing.T) {
	tok := intTok()
	fill := surfaceFill(tok)
	var first image.Rectangle
	for i, r := range []Role{Info, Success, Warning, Error} {
		box := toastBounds(paint1(t, Props{Role: r, Text: "Rescanned: 2 notes"}, tok), fill)
		if box.Empty() {
			t.Fatalf("role %d painted no inverse fill", r)
		}
		if i == 0 {
			first = box
			continue
		}
		if box != first {
			t.Errorf("role %d fills %v against Info's %v; the fill must not tell the roles apart", r, box, first)
		}
	}
}
