package scrollbar

import (
	"image"
	"image/color"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	golden "github.com/vibrantgio/components/golden"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// matchSize is the plane every test here renders into: tall enough that a
// fraction of the track lands on a row a sample can name, and at a 1:1 device
// pixel ratio so a dp is a pixel and the arithmetic below is the code's.
var matchSize = image.Pt(24, 400)

// The track a 400px-tall bar with the default 2dp padding lays out: rows 2
// through 397, so 396 long. Spelled out rather than recomputed, because what
// these tests hold is that a fraction lands on a known row.
const (
	trackTop = 2
	trackLen = 396
	// sampleX is a column inside the bar's minor extent — the track runs
	// from x=2 to x=8 — clear of both padded edges.
	sampleX = 5
)

// barPixels renders a vertical bar over the scheme's surface at the given
// animation time and returns the pixels. state is reused so a caller can walk
// a bar through more than one frame.
func barPixels(t *testing.T, p tokens.PlatformColors, style Style, state *State, now time.Time, viewStart, viewEnd float32) *image.RGBA {
	t.Helper()
	return golden.Capture(t, matchSize, func(gtx layout.Context) layout.Dimensions {
		gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
		gtx.Now = now
		paint.FillShape(gtx.Ops, p.ControlBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
		style.Layout(gtx, state, layout.Vertical, viewStart, viewEnd)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

// at reads one pixel of a capture as an opaque colour.
func at(img *image.RGBA, x, y int) color.NRGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// Every fill the bar paints carries a coverage — the thumb and both match
// fills are translucent by design — so a pixel gate reads the composite the
// rasterizer wrote and not the fill it was handed. vgcolor.Over blends in
// linear light, which is where Gio blends.
//
// onSurface is what one fill lands as directly on the plane behind the bar;
// onThumb is what it lands as where the thumb is under it.
func onSurface(fill color.NRGBA, p tokens.PlatformColors) color.NRGBA {
	return vgcolor.Over(fill, p.ControlBackground)
}

func onThumb(fill color.NRGBA, p tokens.PlatformColors) color.NRGBA {
	return vgcolor.Over(fill, vgcolor.Over(p.ScrollbarThumb, p.ControlBackground))
}

// near reports whether two pixels agree to within one 255th per channel:
// the rasterizer and vgcolor.Over both blend in linear light but quantize
// independently.
func near(got, want color.NRGBA) bool {
	diff := func(a, b uint8) int {
		if a > b {
			return int(a) - int(b)
		}
		return int(b) - int(a)
	}
	return diff(got.R, want.R) <= 1 && diff(got.G, want.G) <= 1 &&
		diff(got.B, want.B) <= 1 && diff(got.A, want.A) <= 1
}

// TestMatchesPaintWhereTheyLieInTheTrack pins both halves of the contract at
// once: the row a fraction of the content lands on, and the fill it is
// painted in. A match at 0.25 paints a quarter of the way down the track, the
// last one ends level with the track's end rather than running past the bar's
// own inset, and the one the caller is on wears the stronger fill.
func TestMatchesPaintWhereTheyLieInTheTrack(t *testing.T) {
	// The viewport is deliberately the middle fifth, so the thumb covers the
	// match at 0.5 and the sample proves it is painted over the thumb.
	const viewStart, viewEnd = 0.4, 0.6
	rows := []struct {
		fraction   float32
		first, end int // the first and last row the match paints
	}{
		{0, trackTop, trackTop + 2},
		{0.25, 100, 102},
		{0.5, 199, 201},
		{1, 395, trackTop + trackLen - 1},
	}
	const current = 2 // the match at 0.5

	for _, scheme := range []struct {
		name string
		p    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(scheme.name, func(t *testing.T) {
			style := FromTokens(scheme.p)
			style.Current = current
			for _, r := range rows {
				style.Matches = append(style.Matches, r.fraction)
			}
			img := barPixels(t, scheme.p, style, NewState(), time.Time{}, viewStart, viewEnd)

			// The match at 0.5 lies under the thumb, so its composite has
			// the thumb in it; the other three land straight on the plane.
			marked := onSurface(style.MatchFill, scheme.p)
			markedOnThumb := onThumb(style.MatchFill, scheme.p)
			currentOnThumb := onThumb(style.CurrentMatchFill, scheme.p)
			for i, r := range rows {
				want := marked
				if i == current {
					want = currentOnThumb
				}
				for y := r.first; y <= r.end; y++ {
					if got := at(img, sampleX, y); !near(got, want) {
						t.Errorf("match %d at %v: row %d is %v, want %v", i, r.fraction, y, got, want)
					}
				}
				for _, y := range []int{r.first - 1, r.end + 1} {
					got := at(img, sampleX, y)
					if near(got, marked) || near(got, markedOnThumb) || near(got, currentOnThumb) {
						t.Errorf("match %d at %v: row %d outside it carries a match fill %v", i, r.fraction, y, got)
					}
				}
			}
			if near(markedOnThumb, currentOnThumb) {
				t.Errorf("the current match lands as %v, the same as the rest", currentOnThumb)
			}

			// Nothing reaches past the track's own minor extent — x=2 to
			// x=8, the thumb's width inside the padding — so a match never
			// paints into the content the bar stands beside.
			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
					if x >= trackTop && x < trackTop+6 {
						continue
					}
					got := at(img, x, y)
					if near(got, marked) || near(got, markedOnThumb) || near(got, currentOnThumb) {
						t.Fatalf("pixel (%d,%d) is %v, outside the track's minor extent", x, y, got)
					}
				}
			}
		})
	}
}

// TestTheThumbNeverHidesAMatch is the invariant the drawing order exists for:
// a match the thumb happens to sit on is painted over it, not under it. Both
// fills carry a coverage, so the two orders land on different composites and
// the pixel says which one was drawn.
func TestTheThumbNeverHidesAMatch(t *testing.T) {
	for _, scheme := range []struct {
		name string
		p    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(scheme.name, func(t *testing.T) {
			style := FromTokens(scheme.p)
			style.Matches = []float32{0.5}
			img := barPixels(t, scheme.p, style, NewState(), time.Time{}, 0.4, 0.6)
			// The match over the thumb, and the same match had it been
			// drawn under the thumb instead.
			want := onThumb(style.MatchFill, scheme.p)
			under := vgcolor.Over(scheme.p.ScrollbarThumb, onSurface(style.MatchFill, scheme.p))
			if near(want, under) {
				t.Fatalf("the two drawing orders land on the same pixel %v; this test cannot tell them apart", want)
			}
			// The thumb runs from row 160 to row 240 at this viewport, so
			// rows 199..201 are inside it.
			for y := 199; y <= 201; y++ {
				if got := at(img, sampleX, y); !near(got, want) {
					t.Errorf("row %d over the thumb is %v, want the match over the thumb %v", y, got, want)
				}
			}
			if got := at(img, sampleX, 180); near(got, want) {
				t.Errorf("row 180 is the match over the thumb, so the thumb is not where this test needs it")
			}
		})
	}
}

// TestNothingIsPaintedWithoutMatches holds the other half of "they die with
// the query": a bar handed no matches paints none, so a caller that drops the
// query by dropping the slice leaves nothing behind.
func TestNothingIsPaintedWithoutMatches(t *testing.T) {
	for _, scheme := range []struct {
		name string
		p    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(scheme.name, func(t *testing.T) {
			style := FromTokens(scheme.p)
			img := barPixels(t, scheme.p, style, NewState(), time.Time{}, 0.4, 0.6)
			bounds := img.Bounds()
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					got := at(img, x, y)
					if near(got, onSurface(style.MatchFill, scheme.p)) ||
						near(got, onThumb(style.MatchFill, scheme.p)) ||
						near(got, onThumb(style.CurrentMatchFill, scheme.p)) {
						t.Fatalf("pixel (%d,%d) is %v with no matches handed to the bar", x, y, got)
					}
				}
			}
		})
	}
}

// TestTheMatchesStayWhileTheThumbFades pins the platform behaviour the doc
// claims: the thumb goes when the content stops moving, and where the matches
// lie stays for as long as the query does.
func TestTheMatchesStayWhileTheThumbFades(t *testing.T) {
	p := tokens.PlatformLight
	style := FromTokens(p)
	style.Matches = []float32{0.25}
	state := NewState()
	t0 := time.Unix(1700000000, 0)

	// One frame to start the timeline, then a frame past the whole fade.
	var ops op.Ops
	gtx := testContext(&ops, matchSize)
	gtx.Now = t0
	style.Layout(gtx, state, layout.Vertical, 0.4, 0.6)

	img := barPixels(t, p, style, state, t0.Add(style.FadeDelay+style.FadeDuration), 0.4, 0.6)
	if got := at(img, sampleX, 180); !near(got, p.ControlBackground) {
		t.Errorf("row 180 is %v after the fade, want the bare plane %v — the thumb is still there", got, p.ControlBackground)
	}
	want := onSurface(style.MatchFill, p)
	for y := 100; y <= 102; y++ {
		if got := at(img, sampleX, y); !near(got, want) {
			t.Errorf("row %d is %v after the fade, want the match's composite %v", y, got, want)
		}
	}
}
