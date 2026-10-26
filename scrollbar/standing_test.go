package scrollbar

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// The standing bar as the platform draws it, MEASURED in
// reference/macos/controls.md, "What the standing scrollbar measures", off
// Finder's list under Show scroll bars: Always at 1x. The lane is 17 px with
// no fill of its own; the track is an 11 px pill 3 px inside the lane on
// every side; the thumb fills the track's width. Over the content's #ffffff
// light and #1e1e1e dark the track reads #f3f3f3 and #292929, and the thumb
// #7a7a7a and #9f9f9f.
var standingReadings = []struct {
	name         string
	p            tokens.PlatformColors
	track, thumb uint8
}{
	{"light", tokens.PlatformLight, 0xf3, 0x7a},
	{"dark", tokens.PlatformDark, 0x29, 0x9f},
}

const (
	standingLane  = 17
	standingInset = 3
	standingTrack = 11
)

// standingBar renders the bar at Always over the content's fill, at 1x, in a
// plane long along axis, with the thumb on the last third of the track. It
// returns the pixels in axis-independent space: x along the major axis, y
// across the lane.
func standingBar(t *testing.T, p tokens.PlatformColors, axis layout.Axis) (func(along, across int) color.NRGBA, layout.Dimensions) {
	t.Helper()
	size := axis.Convert(image.Pt(400, 40))
	style := FromTokens(p, p.ControlBackground).Always()
	var dims layout.Dimensions
	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
		paint.FillShape(gtx.Ops, p.ControlBackground, clip.Rect{Max: gtx.Constraints.Max}.Op())
		dims = style.Layout(gtx, NewState(), axis, 0.6, 0.9)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	return func(along, across int) color.NRGBA {
		pt := axis.Convert(image.Pt(along, across))
		return at(img, pt.X, pt.Y)
	}, dims
}

func grey(v uint8) color.NRGBA { return color.NRGBA{R: v, G: v, B: v, A: 0xff} }

// TestTheStandingBarIsTheMeasuredLaneTrackAndThumb draws the bar at Always,
// upright and sideways, and reads back the measured lane width, the track
// as a pill inside the padding at its coverage, and the thumb across the
// track's full width at its coverage per scheme.
func TestTheStandingBarIsTheMeasuredLaneTrackAndThumb(t *testing.T) {
	for _, tc := range standingReadings {
		for _, axis := range []layout.Axis{layout.Vertical, layout.Horizontal} {
			name := tc.name + "/upright"
			if axis == layout.Horizontal {
				name = tc.name + "/sideways"
			}
			t.Run(name, func(t *testing.T) {
				px, dims := standingBar(t, tc.p, axis)
				if got := axis.Convert(dims.Size).Y; got != standingLane {
					t.Errorf("the lane is %d px across, want the measured %d", got, standingLane)
				}
				content := tc.p.ControlBackground
				track, thumb := grey(tc.track), grey(tc.thumb)

				// Across the lane at a point on the track clear of the
				// thumb: the content's fill, the track's 11 px, the
				// content's fill. The lane has no fill of its own.
				const onTrack = 100
				for across := range standingLane {
					want := track
					if across < standingInset || across >= standingInset+standingTrack {
						want = content
					}
					if got := px(onTrack, across); !near(got, want) {
						t.Errorf("across the lane at %d the bar paints %v, want %v", across, got, want)
					}
				}
				// Across the lane at a point on the thumb: the thumb fills
				// the track's width, with no inset inside it.
				const onThumb = 300
				for across := standingInset; across < standingInset+standingTrack; across++ {
					if got := px(onThumb, across); !near(got, thumb) {
						t.Errorf("across the thumb at %d the bar paints %v, want %v", across, got, thumb)
					}
				}
				// Along the lane the track starts 3 px inside its end, and
				// its end is rounded: the corner of the track's square is
				// the content's fill, its middle the track's.
				middle := standingInset + standingTrack/2
				if got := px(standingInset-1, middle); !near(got, content) {
					t.Errorf("before the track's end the bar paints %v, want the content's %v", got, content)
				}
				if got := px(standingInset+1, middle); !near(got, track) {
					t.Errorf("at the track's end the bar paints %v, want the track's %v", got, track)
				}
				if got := px(standingInset, standingInset); !near(got, content) {
					t.Errorf("the track's corner paints %v, want the content's %v: the end is not rounded", got, content)
				}
			})
		}
	}
}

// TestTheStandingBarDrawsNothingWhenTheContentFits holds the platform's
// list under Always: when the content fits, there is no track, no thumb and
// no lane. The bar records no operation and takes no space.
func TestTheStandingBarDrawsNothingWhenTheContentFits(t *testing.T) {
	for _, tc := range standingReadings {
		for _, axis := range []layout.Axis{layout.Vertical, layout.Horizontal} {
			style := FromTokens(tc.p, tc.p.ControlBackground).Always()
			var ops op.Ops
			gtx := testContext(&ops, axis.Convert(image.Pt(400, 40)))
			dims := style.Layout(gtx, NewState(), axis, 0, 1)
			if dims != (layout.Dimensions{}) {
				t.Errorf("%s: a standing bar over content that fits takes %v, want no lane", tc.name, dims.Size)
			}
			if n := len(recordOps(func(o *op.Ops) { style.Layout(testContext(o, gtx.Constraints.Max), NewState(), axis, 0, 1) })); n != 0 {
				t.Errorf("%s: a standing bar over content that fits records %d op bytes, want none", tc.name, n)
			}
		}
	}
}
