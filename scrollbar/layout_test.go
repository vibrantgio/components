package scrollbar

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/theme/tokens"
)

// testContext returns a bare layout context at the given size with a
// 1:1 dp-to-px metric.
func testContext(ops *op.Ops, size image.Point) layout.Context {
	return layout.Context{
		Ops:         ops,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: size},
	}
}

func TestLayoutScrollableRange(t *testing.T) {
	style := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	size := image.Pt(40, 400)

	cases := []struct {
		name string
		axis layout.Axis
		want image.Point
	}{
		// The bar pins itself to the full major axis and Width() (10dp)
		// along the minor axis, regardless of the incoming minor extent.
		{"Vertical", layout.Vertical, image.Pt(10, 400)},
		{"Horizontal", layout.Horizontal, image.Pt(40, 10)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState()
			var ops op.Ops
			gtx := testContext(&ops, size)

			dims := style.Layout(gtx, state, tc.axis, 0.25, 0.5)
			if dims.Size != tc.want {
				t.Errorf("Layout(0.25, 0.5) dims = %v, want %v", dims.Size, tc.want)
			}
		})
	}
}

func TestLayoutUnscrollableRange(t *testing.T) {
	style := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	state := NewState()
	var ops op.Ops
	gtx := testContext(&ops, image.Pt(40, 400))

	dims := style.Layout(gtx, state, layout.Vertical, 0, 1)
	if dims != (layout.Dimensions{}) {
		t.Errorf("Layout(0, 1) dims = %v, want zero dimensions", dims)
	}
}

// opBytes records one frame of style's bar into a fresh op list and returns
// the serialised operations. Gio keeps them in an unexported byte slice; the
// test reads that slice through reflection so two frames can be compared byte
// for byte. Each frame gets its own State, so the pointer tags differ only in
// the reference list and never in these bytes.
func opBytes(t *testing.T, style Style) []byte {
	t.Helper()
	var ops op.Ops
	gtx := testContext(&ops, image.Pt(40, 400))
	gtx.Now = time.Unix(1700000000, 0)
	style.Layout(gtx, NewState(), layout.Vertical, 0.25, 0.5)
	data := reflect.ValueOf(&ops.Internal).Elem().FieldByName("data")
	if !data.IsValid() {
		t.Fatal("gioui.org/internal/ops.Ops has no data field; update opBytes")
	}
	return bytes.Clone(data.Bytes())
}

// opColor is the first byte of Gio's colour operation in the serialised op
// list. The four bytes after it are the colour's R, G, B and A.
const opColor = 0xd1

// colorOps counts the colour operations in stream that set c.
func colorOps(stream []byte, c color.NRGBA) int {
	return bytes.Count(stream, []byte{opColor, c.R, c.G, c.B, c.A})
}

// recordOps runs draw into a fresh op list and returns its serialised
// operations.
func recordOps(draw func(ops *op.Ops)) []byte {
	var ops op.Ops
	draw(&ops)
	return bytes.Clone(reflect.ValueOf(&ops.Internal).Elem().FieldByName("data").Bytes())
}

// The standing bar at one pixel per dp in a 400 px column: the 11 px track
// inside the 3 px padding is 394 px long, and the viewport 0.25 to 0.5 makes
// the thumb 99 px long. Each paint is recorded at the origin: its offset is
// pushed outside the paint, so a paint has the same length wherever it lands.
func standingPaints(track, thumb color.NRGBA) (trackPaint, thumbPaint []byte) {
	trackPaint = recordOps(func(ops *op.Ops) {
		paint.FillShape(ops, track, roundRect(ops, image.Rect(0, 0, 11, 394), 5.5))
	})
	thumbPaint = recordOps(func(ops *op.Ops) {
		paint.FillShape(ops, thumb, roundRect(ops, image.Rect(0, 0, 11, 99), 5.5))
	})
	return trackPaint, thumbPaint
}

// TestTransparentTrackPaintsNothing asserts that a track whose colour has
// alpha 0 adds nothing to the op stream: no colour operation, and the
// stream is shorter than the same bar's with an opaque track by exactly the
// track's paint, its clip included.
func TestTransparentTrackPaintsNothing(t *testing.T) {
	base := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground).Always()
	track := color.NRGBA{R: 0xff, G: 0x80, B: 0x40, A: 0xff}

	transparent := base
	transparent.TrackColor = color.NRGBA{R: 0xff, G: 0x80, B: 0x40}
	opaque := base
	opaque.TrackColor = track

	trackPaint, _ := standingPaints(track, base.ThumbColor)
	got, withTrack := opBytes(t, transparent), opBytes(t, opaque)
	if n := colorOps(withTrack, track); n != 1 {
		t.Fatalf("the opaque bar sets the track's colour %d times, want once", n)
	}
	if n := colorOps(got, transparent.TrackColor); n != 0 {
		t.Errorf("a transparent track sets its colour %d times, want none", n)
	}
	if len(withTrack)-len(got) != len(trackPaint) {
		t.Errorf("a transparent track recorded %d op bytes, want the %d of the same bar with no track paint", len(got), len(withTrack)-len(trackPaint))
	}
}

// TestFadedBarPaintsNothing asserts that a bar at rest, whose thumb has
// faded to alpha 0 and whose track is transparent, adds no paint to the op
// stream. The reference is the same bar standing, with an opaque track and
// an opaque thumb: the faded bar sets neither colour, and its stream is
// shorter by exactly the two paints, their clips included. The pointer
// areas stay in both recordings, so a faded bar can still be hovered.
func TestFadedBarPaintsNothing(t *testing.T) {
	base := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	track := color.NRGBA{R: 0xff, G: 0x80, B: 0x40, A: 0xff}
	thumb := color.NRGBA{R: 0x20, G: 0x40, B: 0xc0, A: 0xff}
	if base.FadeDelay <= 0 {
		t.Fatal("the default style does not fade; the faded bar needs a FadeDelay")
	}

	// A bar that has not been scrolled or hovered has faded out. It has
	// the standing bar's geometry, so the two streams differ in the
	// paints alone.
	standing := base.Always()
	faded := standing
	faded.FadeDelay = base.FadeDelay
	faded.TrackColor = color.NRGBA{}
	faded.ThumbColor = thumb
	opaque := standing
	opaque.TrackColor = track
	opaque.ThumbColor = thumb

	trackPaint, thumbPaint := standingPaints(track, thumb)
	got, want := opBytes(t, faded), opBytes(t, opaque)
	for _, c := range []color.NRGBA{track, thumb} {
		if n := colorOps(want, c); n != 1 {
			t.Fatalf("the opaque bar sets %v %d times, want once", c, n)
		}
		if n := colorOps(got, c); n != 0 {
			t.Errorf("a faded bar sets %v %d times, want none", c, n)
		}
	}
	faded.ThumbColor.A = 0
	if n := colorOps(got, faded.ThumbColor); n != 0 {
		t.Errorf("a faded bar sets its faded thumb's colour %d times, want none", n)
	}
	if len(want)-len(got) != len(trackPaint)+len(thumbPaint) {
		t.Errorf("a faded bar recorded %d op bytes, want the %d of the same bar with no paint", len(got), len(want)-len(trackPaint)-len(thumbPaint))
	}
}
