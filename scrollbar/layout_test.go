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
