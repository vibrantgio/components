// Package scrollbar provides a visible scrollbar for scrollable regions.
//
// The API is immediate-mode, matching components/list: allocate a State once per
// scrollable region and reuse it every frame, while a Style is a plain
// snapshot of resolved colours and metrics derived per frame (typically via
// FromTokens). It pairs with components/list through list.LayoutScrollbar so
// virtual lists can show their scroll position.
package scrollbar

import (
	"image/color"
	"time"

	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/vibrantgio/theme/tokens"
)

// State holds the scrollbar's interaction state across frames.
// Allocate once per scrollbar instance and reuse on every frame.
//
// It embeds gioui.org/widget.Scrollbar, so Update, ScrollDistance,
// Dragging, IndicatorHovered and TrackHovered are promoted.
type State struct {
	widget.Scrollbar

	// seen reports whether a frame has been laid out yet. The first frame
	// of a scrollbar's life counts as activity, so a bar is fully opaque
	// the moment it appears — which is also what keeps single-frame
	// golden captures independent of the fade.
	seen bool
	// start and end are the viewport fractions the previous frame drew.
	// A change between frames is what "the content is scrolling" means
	// here: the bar has no other way to know.
	start, end float32
	// lastActive is the animation time of the most recent activity —
	// a scroll, a hover over the gutter or thumb, or a drag.
	lastActive time.Time
}

// NewState returns a fresh scrollbar State.
func NewState() *State {
	return &State{}
}

// Style describes how a scrollbar is drawn for one frame: resolved colours
// plus metrics. Derive defaults with FromTokens and override fields as needed.
type Style struct {
	// ThumbColor fills the thumb at rest.
	ThumbColor color.NRGBA
	// ThumbHoverColor fills the thumb while hovered or dragged.
	ThumbHoverColor color.NRGBA
	// TrackColor fills the track gutter. The zero value draws nothing.
	TrackColor color.NRGBA

	// ThumbMinorWidth is the thumb's extent along the minor axis.
	ThumbMinorWidth unit.Dp
	// TrackPadding is the gutter padding on each side of the thumb.
	TrackPadding unit.Dp
	// ThumbCornerRadius rounds the thumb's corners.
	ThumbCornerRadius unit.Dp
	// ThumbMinLen is the minimum thumb length along the major axis.
	ThumbMinLen unit.Dp

	// FadeDelay is how long the bar stays fully opaque after the last
	// activity — a scroll, a hover over the gutter or thumb, or a drag —
	// before it begins to fade. The zero value disables fading: the bar
	// stays visible for as long as the content overflows.
	FadeDelay time.Duration
	// FadeDuration is how long the fade to invisible takes once FadeDelay
	// has elapsed. Ignored when FadeDelay is zero; a zero value here with
	// a non-zero delay makes the bar vanish rather than fade.
	FadeDuration time.Duration
}

// Width returns the total gutter width along the minor axis:
// the thumb width plus padding on both sides.
func (s Style) Width() unit.Dp {
	return s.ThumbMinorWidth + 2*s.TrackPadding
}

// FromTokens derives the default scrollbar look from colour tokens.
// The thumb is the Neutral 700 step (ADR-007's low-contrast-text step)
// alpha-composited (~40% at rest, ~67% while hovered or dragged), so it
// tracks light and dark schemes automatically. The translucency is the
// thumb's identity — content shows through the overlay — not an MD3 state
// layer, so it survives ADR-007's move to ramp-step states.
// The track is transparent by default.
//
// The bar also fades: it is opaque while the content scrolls or the pointer
// is on the gutter, then fades out a second after the last of either, which
// is how the desktop platforms' overlay scrollbars behave. The gutter keeps
// its hit areas while faded, so moving the pointer onto it brings the bar
// back. Set FadeDelay to zero for a bar that stays visible.
func FromTokens(c tokens.ColorTokens) Style {
	thumb := c.Ramps.Neutral.Step(700)
	thumb.A = 100
	hover := c.Ramps.Neutral.Step(700)
	hover.A = 170
	return Style{
		ThumbColor:        thumb,
		ThumbHoverColor:   hover,
		TrackColor:        color.NRGBA{},
		ThumbMinorWidth:   6,
		TrackPadding:      2,
		ThumbCornerRadius: 3,
		ThumbMinLen:       16,
		FadeDelay:         time.Second,
		FadeDuration:      tokens.Motion.DurSlow,
	}
}
