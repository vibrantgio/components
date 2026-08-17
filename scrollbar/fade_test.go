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

// TestFadeTimeline walks one scrollbar through the whole timeline: opaque on
// the frame it appears, opaque for the delay, part-way through the fade at
// the delay's midpoint, invisible after, and opaque again the moment the
// content moves.
func TestFadeTimeline(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	state := NewState()
	t0 := time.Unix(1700000000, 0)

	if got := fadeAt(style, state, t0, 0.2, 0.5); got != 1 {
		t.Errorf("first frame opacity = %v, want 1", got)
	}
	// Still inside the delay: nothing has faded yet.
	if got := fadeAt(style, state, t0.Add(style.FadeDelay-time.Millisecond), 0.2, 0.5); got != 1 {
		t.Errorf("opacity just before the delay expires = %v, want 1", got)
	}
	// Halfway through the fade.
	mid := t0.Add(style.FadeDelay + style.FadeDuration/2)
	got := fadeAt(style, state, mid, 0.2, 0.5)
	if got <= 0.4 || got >= 0.6 {
		t.Errorf("opacity halfway through the fade = %v, want about 0.5", got)
	}
	// Fade complete.
	if got := fadeAt(style, state, t0.Add(style.FadeDelay+style.FadeDuration), 0.2, 0.5); got != 0 {
		t.Errorf("opacity after the fade = %v, want 0", got)
	}
	// A scroll — a changed viewport — restores it immediately, on the same
	// frame, without waiting for a fade-in.
	later := t0.Add(time.Hour)
	if got := fadeAt(style, state, later, 0.3, 0.6); got != 1 {
		t.Errorf("opacity on a scrolling frame = %v, want 1", got)
	}
	// And it starts fading again from that moment, not from t0.
	if got := fadeAt(style, state, later.Add(style.FadeDelay-time.Millisecond), 0.3, 0.6); got != 1 {
		t.Errorf("opacity after a scroll re-armed the delay = %v, want 1", got)
	}
}

// TestFadeDisabled asserts the zero FadeDelay opt-out: a Style with no delay
// keeps the thumb fully opaque however long the content sits still.
func TestFadeDisabled(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
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
	style := FromTokens(tokens.DefaultLight)
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
// existing light-mid and a blank gutter. The state is walked to the fade's
// midpoint on a throwaway frame first, because the frame a scrollbar is born
// on is always opaque.
func TestFadeGolden(t *testing.T) {
	style := FromTokens(tokens.DefaultLight)
	state := NewState()
	t0 := time.Unix(1700000000, 0)
	fadeAt(style, state, t0, 0.35, 0.65)
	mid := t0.Add(style.FadeDelay + style.FadeDuration/2)
	surface := tokens.DefaultLight.Surface

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
	s := FromTokens(tokens.DefaultLight)
	if s.FadeDelay != time.Second {
		t.Errorf("FadeDelay = %v, want %v", s.FadeDelay, time.Second)
	}
	if s.FadeDuration != tokens.Motion.DurSlow {
		t.Errorf("FadeDuration = %v, want %v", s.FadeDuration, tokens.Motion.DurSlow)
	}
}
