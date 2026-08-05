// Package scrollbar provides a visible scrollbar for scrollable regions.
//
// The API is immediate-mode, matching prism/list: allocate a State once per
// scrollable region and reuse it every frame, while a Style is a plain
// snapshot of resolved colours and metrics derived per frame (typically via
// FromTokens). It pairs with prism/list through list.LayoutScrollbar so
// virtual lists can show their scroll position.
package scrollbar

import (
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/vibrantgio/spectrum/tokens"
)

// State holds the scrollbar's interaction state across frames.
// Allocate once per scrollbar instance and reuse on every frame.
//
// It embeds gioui.org/widget.Scrollbar, so Update, ScrollDistance,
// Dragging, IndicatorHovered and TrackHovered are promoted.
type State struct {
	widget.Scrollbar
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
	}
}
