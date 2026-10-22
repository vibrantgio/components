package scrollbar

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// fadeAt runs one frame at animation time now and returns the resulting
// opacity multiplier. It drives the real Layout so the value under test is
// the one the thumb is actually painted with, not a parallel calculation.
func fadeAt(style Style, state *State, now time.Time, start, end float32) float32 {
	var ops op.Ops
	gtx := testContext(&ops, image.Pt(40, 400))
	gtx.Now = now
	return style.fade(gtx, state, start, end)
}

// TestFadeTimeline walks one scrollbar through the whole timeline on a test
// clock: nothing at rest, opaque the moment a scroll moves the content,
// opaque for the delay, part-way through the fade at the delay's midpoint,
// invisible after, and opaque again the moment the content moves next.
func TestFadeTimeline(t *testing.T) {
	style := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	state := NewState()
	t0 := time.Unix(1700000000, 0)

	// At rest. The first frame has no previous viewport to compare against
	// and is not a scroll, so nothing is drawn.
	if got := fadeAt(style, state, t0, 0.2, 0.5); got != 0 {
		t.Errorf("first frame opacity = %v, want 0", got)
	}
	if got := fadeAt(style, state, t0.Add(time.Hour), 0.2, 0.5); got != 0 {
		t.Errorf("opacity of a bar nothing has touched = %v, want 0", got)
	}

	// A scroll: the viewport moves, and the bar is there on that frame.
	t1 := t0.Add(2 * time.Hour)
	if got := fadeAt(style, state, t1, 0.3, 0.6); got != 1 {
		t.Errorf("opacity on a scrolling frame = %v, want 1", got)
	}
	// Still inside the delay: nothing has faded yet.
	if got := fadeAt(style, state, t1.Add(style.FadeDelay-time.Millisecond), 0.3, 0.6); got != 1 {
		t.Errorf("opacity just before the delay expires = %v, want 1", got)
	}
	// Halfway through the fade.
	mid := t1.Add(style.FadeDelay + style.FadeDuration/2)
	got := fadeAt(style, state, mid, 0.3, 0.6)
	if got <= 0.4 || got >= 0.6 {
		t.Errorf("opacity halfway through the fade = %v, want about 0.5", got)
	}
	// Fade complete.
	if got := fadeAt(style, state, t1.Add(style.FadeDelay+style.FadeDuration), 0.3, 0.6); got != 0 {
		t.Errorf("opacity after the fade = %v, want 0", got)
	}
	// The next scroll restores it immediately, on the same frame, without
	// waiting for a fade-in, and the delay is re-armed from that moment.
	later := t1.Add(time.Hour)
	if got := fadeAt(style, state, later, 0.4, 0.7); got != 1 {
		t.Errorf("opacity on the next scrolling frame = %v, want 1", got)
	}
	if got := fadeAt(style, state, later.Add(style.FadeDelay-time.Millisecond), 0.4, 0.7); got != 1 {
		t.Errorf("opacity after a scroll re-armed the delay = %v, want 1", got)
	}
}

// TestAtRestDrawsNoThumb asserts the at-rest state as the drawing and not as
// the multiplier: a bar nothing has scrolled or hovered leaves the fill it
// rides exactly as it found it.
func TestAtRestDrawsNoThumb(t *testing.T) {
	style := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	rides := tokens.PlatformLight.ControlBackground
	size := image.Pt(24, 400)
	state := NewState()
	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
		gtx.Now = time.Unix(1700000000, 0)
		paint.FillShape(gtx.Ops, rides, clip.Rect{Max: gtx.Constraints.Max}.Op())
		style.Layout(gtx, state, layout.Vertical, 0.35, 0.65)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	for y := range size.Y {
		for x := range size.X {
			got := img.RGBAAt(x, y)
			if got.R != rides.R || got.G != rides.G || got.B != rides.B {
				t.Fatalf("a resting bar painted %v at (%d, %d), want the fill it rides, %v", got, x, y, rides)
			}
		}
	}
}

// TestFadeDisabled asserts the zero FadeDelay opt-out: a Style with no delay
// keeps the thumb fully opaque however long the content sits still.
func TestFadeDisabled(t *testing.T) {
	style := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	style.FadeDelay = 0
	state := NewState()
	t0 := time.Unix(1700000000, 0)

	fadeAt(style, state, t0, 0.2, 0.5)
	if got := fadeAt(style, state, t0.Add(time.Hour), 0.2, 0.5); got != 1 {
		t.Errorf("opacity with fading disabled = %v, want 1", got)
	}
}

// TestFadeLeavesGeometryAlone asserts that a fully faded bar still occupies
// its gutter. The alternative — collapsing to nothing — would reflow the
// content it sits beside every time the reader paused.
func TestFadeLeavesGeometryAlone(t *testing.T) {
	style := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	state := NewState()
	t0 := time.Unix(1700000000, 0)

	var ops op.Ops
	gtx := testContext(&ops, image.Pt(40, 400))
	gtx.Now = t0
	first := style.Layout(gtx, state, layout.Vertical, 0.2, 0.5)

	ops.Reset()
	gtx = testContext(&ops, image.Pt(40, 400))
	gtx.Now = t0.Add(time.Hour)
	faded := style.Layout(gtx, state, layout.Vertical, 0.2, 0.5)

	if first.Size != faded.Size {
		t.Errorf("faded dims = %v, want the unfaded %v", faded.Size, first.Size)
	}
}

// TestFadeGolden records the thumb halfway through its fade, which is the
// only part of the timeline a still image can show: the opaque ends are the
// existing light-mid and a blank gutter. Two throwaway frames walk the state
// to the fade's midpoint — the first gives the bar a viewport to compare
// against, the second moves it, which is the scroll the fade times from.
func TestFadeGolden(t *testing.T) {
	style := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	state := NewState()
	t0 := time.Unix(1700000000, 0)
	fadeAt(style, state, t0, 0.3, 0.6)
	fadeAt(style, state, t0, 0.35, 0.65)
	mid := t0.Add(style.FadeDelay + style.FadeDuration/2)
	surface := tokens.PlatformLight.ControlBackground

	golden.Render(t, "fading-mid", image.Pt(24, 400), func(gtx layout.Context) layout.Dimensions {
		gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
		gtx.Now = mid
		paint.FillShape(gtx.Ops, surface, clip.Rect{Max: gtx.Constraints.Max}.Op())
		style.Layout(gtx, state, layout.Vertical, 0.35, 0.65)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

// TestFadeDefaults pins the timings FromTokens hands out, so a change to the
// platform-matching second-then-fade behaviour is a deliberate edit.
func TestFadeDefaults(t *testing.T) {
	s := FromTokens(tokens.PlatformLight, tokens.PlatformLight.ControlBackground)
	if s.FadeDelay != time.Second {
		t.Errorf("FadeDelay = %v, want %v", s.FadeDelay, time.Second)
	}
	if s.FadeDuration != tokens.Motion.DurSlow {
		t.Errorf("FadeDuration = %v, want %v", s.FadeDuration, tokens.Motion.DurSlow)
	}
}
