package scrollbar

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
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

// TestTransparentTrackPaintsNothing asserts that a track whose colour has
// alpha 0 adds nothing to the op stream. The reference is the same bar with
// an opaque track and that track's paint cut out of the recording: the paint
// is recorded on its own into a fresh op list, found once in the opaque bar's
// stream and removed, and what remains must equal the transparent bar's
// stream byte for byte.
func TestTransparentTrackPaintsNothing(t *testing.T) {
	base := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	track := color.NRGBA{R: 0xff, G: 0x80, B: 0x40, A: 0xff}

	transparent := base
	transparent.TrackColor = color.NRGBA{}
	opaque := base
	opaque.TrackColor = track

	var paintOps op.Ops
	// The bar is 10 px wide (Width at one pixel per dp) and 400 px tall.
	paint.FillShape(&paintOps, track, clip.Rect(image.Rect(0, 0, 10, 400)).Op())
	trackPaint := bytes.Clone(reflect.ValueOf(&paintOps.Internal).Elem().FieldByName("data").Bytes())

	got, withTrack := opBytes(t, transparent), opBytes(t, opaque)
	at := bytes.Index(withTrack, trackPaint)
	if at < 0 || bytes.Index(withTrack[at+1:], trackPaint) >= 0 {
		t.Fatalf("the opaque track's paint (%d bytes) is not found exactly once in the opaque bar's %d op bytes", len(trackPaint), len(withTrack))
	}
	want := append(bytes.Clone(withTrack[:at]), withTrack[at+len(trackPaint):]...)
	if !bytes.Equal(got, want) {
		t.Errorf("a transparent track recorded %d op bytes, want the %d of the same bar with no track paint", len(got), len(want))
	}
}

// The first byte of Gio's macro and call operations in the serialised op
// list. A macro records the end of its operations as a data offset and a
// reference count (each a little-endian uint32 after the type byte); a call
// records the start and the end of the macro it replays in the same form.
const (
	opMacro = 0xc8
	opCall  = 0xc9
	callLen = 1 + 4*4
	// A path operation is its type byte and an 8-byte hash of the path.
	pathLen = 1 + 8
)

// recordOps runs draw into a fresh op list and returns its serialised
// operations and the number of references they hold.
func recordOps(draw func(ops *op.Ops)) ([]byte, int) {
	var ops op.Ops
	draw(&ops)
	internal := reflect.ValueOf(&ops.Internal).Elem()
	return bytes.Clone(internal.FieldByName("data").Bytes()), internal.FieldByName("refs").Len()
}

// addUint32 adds delta to the little-endian uint32 at b[:4].
func addUint32(b []byte, delta int) {
	binary.LittleEndian.PutUint32(b, uint32(int(binary.LittleEndian.Uint32(b))+delta))
}

// TestFadedBarPaintsNothing asserts that a bar at rest, whose thumb has
// faded to alpha 0 and whose track is transparent, adds no paint to the op
// stream. The reference is the same bar with the fade turned off, an opaque
// track and an opaque thumb, with both paints cut out of its recording:
//
//   - The track's paint is recorded on its own into a fresh op list, found
//     once in the opaque bar's stream and removed, as in
//     TestTransparentTrackPaintsNothing.
//   - The thumb's paint is recorded on its own the same way. Its rounded
//     rectangle is a path whose data is kept in a macro, closed by the
//     path operation and replayed by a call; the macro and the call record
//     absolute data offsets, so each candidate place in the stream is tried
//     with those offsets moved there. Nothing before the thumb holds a
//     reference, so the reference counts need no change. The paint is found
//     exactly once and removed.
//   - The thumb is drawn inside layout.Background's macro, which opens the
//     stream and is replayed by the call that closes it. Both record the
//     macro's end, which moves back by the removed bytes and references.
//
// What remains must equal the faded bar's stream byte for byte. The pointer
// areas stay in both recordings, so a faded bar can still be hovered.
func TestFadedBarPaintsNothing(t *testing.T) {
	base := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	track := color.NRGBA{R: 0xff, G: 0x80, B: 0x40, A: 0xff}
	thumb := color.NRGBA{R: 0x20, G: 0x40, B: 0xc0, A: 0xff}
	if base.FadeDelay <= 0 {
		t.Fatal("the default style does not fade; the faded bar needs a FadeDelay")
	}

	// A bar that has not been scrolled or hovered has faded out.
	faded := base
	faded.TrackColor = color.NRGBA{}
	faded.ThumbColor = thumb
	opaque := base
	opaque.FadeDelay = 0
	opaque.TrackColor = track
	opaque.ThumbColor = thumb

	// The bar is 10 px wide and 400 px tall at one pixel per dp. The thumb
	// is 6 px wide with a 3 px corner radius; inside the 2 px padding the
	// track is 396 px long, and the viewport 0.25 to 0.5 makes the thumb
	// 99 px long. Its offset is pushed outside the paint and stays.
	trackPaint, _ := recordOps(func(ops *op.Ops) {
		paint.FillShape(ops, track, clip.Rect(image.Rect(0, 0, 10, 400)).Op())
	})
	thumbPaint, thumbRefs := recordOps(func(ops *op.Ops) {
		paint.FillShape(ops, thumb, clip.RRect{
			Rect: image.Rect(0, 0, 6, 99),
			SW:   3, NW: 3, NE: 3, SE: 3,
		}.Op(ops))
	})
	if len(thumbPaint) < 9 || thumbPaint[0] != opMacro {
		t.Fatalf("the thumb's paint does not open with its path's macro: % x", thumbPaint[:min(len(thumbPaint), 9)])
	}
	pathCall := int(binary.LittleEndian.Uint32(thumbPaint[1:])) + pathLen
	if pathCall+callLen > len(thumbPaint) || thumbPaint[pathCall] != opCall {
		t.Fatalf("the thumb's path macro is not followed by its path operation and call")
	}

	got, want := opBytes(t, faded), opBytes(t, opaque)

	at := bytes.Index(want, trackPaint)
	if at < 0 || bytes.Index(want[at+1:], trackPaint) >= 0 {
		t.Fatalf("the opaque track's paint (%d bytes) is not found exactly once in the opaque bar's %d op bytes", len(trackPaint), len(want))
	}
	want = append(bytes.Clone(want[:at]), want[at+len(trackPaint):]...)

	at = -1
	for p := 0; p+len(thumbPaint) <= len(want); p++ {
		moved := bytes.Clone(thumbPaint)
		for _, offset := range []int{1, pathCall + 1, pathCall + 9} {
			addUint32(moved[offset:], p)
		}
		if bytes.Equal(want[p:p+len(moved)], moved) {
			if at >= 0 {
				t.Fatalf("the opaque thumb's paint is found at %d and at %d", at, p)
			}
			at = p
		}
	}
	if at < 0 {
		t.Fatalf("the opaque thumb's paint (%d bytes) is not found in the opaque bar's %d op bytes", len(thumbPaint), len(want))
	}
	want = append(bytes.Clone(want[:at]), want[at+len(thumbPaint):]...)

	if want[0] != opMacro || len(want) < callLen || want[len(want)-callLen] != opCall {
		t.Fatalf("the opaque bar's stream is not one macro and its call: % x ... % x", want[:9], want[len(want)-callLen:])
	}
	for _, end := range [][]byte{want[1:9], want[len(want)-8:]} {
		addUint32(end[0:], -len(thumbPaint))
		addUint32(end[4:], -thumbRefs)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("a faded bar recorded %d op bytes, want the %d of the same bar with no paint", len(got), len(want))
	}
}
