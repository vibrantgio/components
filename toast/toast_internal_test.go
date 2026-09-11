package toast

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/vibrantgio/components/golden"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

const intFrameW, intFrameH = 320, 240

var intFrame = image.Pt(intFrameW, intFrameH)

func intTok() resolvedTokens {
	return resolvedTokens{
		platform: tokens.PlatformLight,
		spacing:  tokens.Spacing,
		radius:   tokens.RadiusScale{},
		style:    tokens.DefaultTypography.LabelMedium,
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

// surfaceFill is the flat colour draw fills every toast with, whatever its
// status: the window's own plane. It cannot tell two toasts apart — that is
// what statusEdge is for.
func surfaceFill(tok resolvedTokens) color.NRGBA {
	return Fill(tok.platform)
}

// statusEdge is the colour of the leading edge a toast of the given status
// paints — the one place on an otherwise identical surface that says which
// status the toast indicates.
func statusEdge(status Status, tok resolvedTokens) color.NRGBA {
	return Edge(tok.platform, status)
}

// toastBounds returns the rectangle enclosing every pixel img paints in
// any of cs. A toast's surface is one flat, opaque, axis-aligned rectangle
// at the zero radius these tests render with, so its extent can be read
// back off the image: pass the fill together with the status's edge for the
// whole rectangle, or the edge alone to find the edge inside it.
//
// The match carries a tolerance of one step per channel, because the fill
// reaches the framebuffer through a rasteriser and demanding the exact byte
// would be asserting its arithmetic rather than the geometry. Nothing is
// grown back: the surface carries no outline — what separates a toast from
// what it covers is the shadow its placement lays under it — so the fill and
// the edge reach the outermost row and column themselves.
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
// the status edge on a rendered toast and holds it between the two bounds the
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
	// inset. A status edge has to clear it by a margin, not by a pixel.
	const hairlineBand = 3

	for _, status := range []Status{Info, Success, Warning, Error} {
		img := paint1(t, Props{Status: status, Text: "Rescanned: 2 notes"}, tok)
		edge := toastBounds(img, statusEdge(status, tok))
		if edge.Empty() {
			t.Fatalf("status %d painted no leading edge", status)
		}
		if edge.Dx() <= 2*hairlineBand {
			t.Errorf("status %d edge is %d px wide; a mark read by its colour cannot be drawn at the %d px the platform keeps for hairlines",
				status, edge.Dx(), hairlineBand)
		}
		air := int(tok.spacing.S3) // the message's inset from the edge
		if edge.Dx() >= air {
			t.Errorf("status %d edge is %d px wide against %d px of air before the message; an edge as wide as its own air reads as a panel",
				status, edge.Dx(), air)
		}
		// The air is real, not just arithmetic: find the first column right
		// of the edge carrying anything that is neither the fill nor the
		// edge, which is the message's first drawn pixel.
		fill := surfaceFill(tok)
		firstDrawn := -1
		for x := edge.Max.X; x < edge.Max.X+4*air && firstDrawn < 0; x++ {
			for y := edge.Min.Y; y < edge.Max.Y; y++ {
				got := img.RGBAAt(x, y)
				if !nearColor(got, fill) && !nearColor(got, statusEdge(status, tok)) {
					firstDrawn = x
					break
				}
			}
		}
		if firstDrawn < 0 {
			t.Fatalf("status %d: no message pixels found beside the edge", status)
		}
		if gap := firstDrawn - edge.Max.X; gap <= edge.Dx() {
			t.Errorf("status %d: %d px of air between the edge and the message against a %d px edge; the mark must not out-measure the space it keeps",
				status, gap, edge.Dx())
		}
		t.Logf("status %d: edge %d px wide, message starts %d px past it", status, edge.Dx(), firstDrawn-edge.Max.X)
	}
}

// TestEveryStatusIsThePlatformsColourForIt is the colour half of the same
// claim, read off both recorded appearances: the edge is the only thing on a
// toast that says which status this is, so it is the platform's system
// colour for that status and nothing derived from one.
func TestEveryStatusIsThePlatformsColourForIt(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		for _, tc := range []struct {
			status Status
			name   string
			want   color.NRGBA
		}{
			{Success, "SystemGreen", sc.p.SystemGreen},
			{Warning, "SystemOrange", sc.p.SystemOrange},
			{Error, "SystemRed", sc.p.SystemRed},
			{Info, "SystemBlue", sc.p.SystemBlue},
		} {
			if got := Edge(sc.p, tc.status); got != tc.want {
				t.Errorf("%s, status %d: edge = %v, want %s %v", sc.name, tc.status, got, tc.name, tc.want)
			}
		}
	}
}

// TestTheFillAndTheMessageAreThePlatformsOwn pins the other two names: a
// toast is filled with the window's own plane and its message reads in the
// platform's label colour over that plane, in both appearances.
func TestTheFillAndTheMessageAreThePlatformsOwn(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		if got := Fill(sc.p); got != sc.p.WindowBackground {
			t.Errorf("%s: Fill = %v, want WindowBackground %v", sc.name, got, sc.p.WindowBackground)
		}
		if want := vgcolor.Flatten(sc.p.Label, Fill(sc.p)); Foreground(sc.p) != want {
			t.Errorf("%s: Foreground = %v, want Label over the fill %v", sc.name, Foreground(sc.p), want)
		}
	}
}

// TestTheFillSaysNothingAboutTheStatus pins the half of the Language the
// fill carries: a toast is filled with the window's own plane whatever it
// says, so every status fills identically and only the leading edge differs.
func TestTheFillSaysNothingAboutTheStatus(t *testing.T) {
	tok := intTok()
	fill := surfaceFill(tok)
	var first image.Rectangle
	for i, status := range []Status{Info, Success, Warning, Error} {
		box := toastBounds(paint1(t, Props{Status: status, Text: "Rescanned: 2 notes"}, tok), fill)
		if box.Empty() {
			t.Fatalf("status %d painted no fill", status)
		}
		if i == 0 {
			first = box
			continue
		}
		if box != first {
			t.Errorf("status %d fills %v against Info's %v; the fill must not tell the statuses apart", status, box, first)
		}
	}
}
