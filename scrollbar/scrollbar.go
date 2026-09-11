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

	"github.com/vibrantgio/components/internal/surface"
	vgcolor "github.com/vibrantgio/theme/color"
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

// Style describes how a scrollbar is drawn for one frame: resolved colours,
// metrics, and — while the caller has a search query — where its matches lie
// in the content. Derive defaults with FromTokens and override fields as
// needed.
type Style struct {
	// ThumbColor fills the thumb. FromTokens flattens the platform's knob
	// onto what the bar rides, so this is opaque; its alpha is what the fade
	// scales, and nothing but the fade makes the bar translucent.
	ThumbColor color.NRGBA
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

	// Matches says where the matches of the caller's search query lie in the
	// content: one entry per match, as a fraction of the content's length in
	// [0,1], in any order. The bar paints each of them inside the track at
	// that fraction of its major axis, so the reader sees where in the whole
	// content the query was found and not merely inside the viewport.
	//
	// An empty slice paints nothing. The caller sets this per frame and drops
	// it when it drops the query, which is what makes what is painted live
	// exactly as long as the query does.
	Matches []float32
	// Current indexes Matches: the one match the caller is on, painted in
	// CurrentMatchFill while the rest take MatchFill. An index outside
	// Matches — the -1 FromTokens starts at — paints every match alike.
	Current int

	// MatchFill paints one match of the caller's query in the track.
	//
	// FromTokens takes the platform's find highlight at a coverage, so what
	// the bar paints and what the content itself is highlighted with are the
	// same fill, and flattens it onto what the bar rides.
	MatchFill color.NRGBA
	// CurrentMatchFill paints the match named by Current, so the reader can
	// tell it from the others while stepping through them.
	//
	// FromTokens takes the same highlight at a higher coverage.
	CurrentMatchFill color.NRGBA
	// MatchLen is the extent along the major axis of what one match is
	// painted as; it spans the track's minor extent, the thumb's own width.
	// It has to survive a 1:1 device pixel ratio, so it is a few dp and not
	// one.
	MatchLen unit.Dp

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

// The coverages the platform's find highlight is laid on at: one match among
// many, and the one the caller is on. The two are one fill at two strengths
// rather than two colours, so a reader tells the current match from the rest
// without learning a second mark.
const (
	matchCoverage        = 0x66
	currentMatchCoverage = 0xb2
)

// FromTokens derives the default scrollbar look from the platform's colour
// set.
//
// rides is the opaque fill the bar rides — the content's own, unless the
// caller put the bar on something else. The platform's knob is its label's
// black or white at a measured coverage, and the platform composites that in
// encoded sRGB where Gio's rasterizer would composite it in linear light, so
// the coverage is flattened onto rides here and the bar paints an opaque
// thumb. What shows through an overlay bar is therefore the fill the caller
// named and not the glyphs under it; only the fade makes the thumb
// translucent again.
//
// The bar fades: it is fully present while the content scrolls or the pointer
// is on the gutter, then fades out a second after the last of either, which is
// how the desktop platforms' overlay scrollbars behave. The gutter keeps its
// hit areas while faded, so moving the pointer onto it brings the bar back.
// Set FadeDelay to zero for a bar that stays visible.
//
// The track is transparent by default: the platform's overlay bar shows no
// gutter until it is being operated.
//
// What the bar paints for a search query does not fade with the thumb: the
// places of the matches stay in the track for as long as the query does, and a
// reader who has just searched is looking for exactly them. Both fills are
// flattened onto rides, for the same reason the thumb is.
func FromTokens(p tokens.PlatformColors, rides color.NRGBA) Style {
	rides = surface.Or(rides, p.ControlBackground)
	match, currentMatch := p.FindHighlight, p.FindHighlight
	match.A, currentMatch.A = matchCoverage, currentMatchCoverage
	return Style{
		ThumbColor:        vgcolor.Flatten(p.ScrollbarThumb, rides),
		TrackColor:        color.NRGBA{}, // transparent: the content shows through
		ThumbMinorWidth:   6,
		TrackPadding:      2,
		ThumbCornerRadius: 3,
		ThumbMinLen:       16,
		Current:           -1,
		MatchFill:         vgcolor.Flatten(match, rides),
		CurrentMatchFill:  vgcolor.Flatten(currentMatch, rides),
		MatchLen:          3,
		FadeDelay:         time.Second,
		FadeDuration:      tokens.Motion.DurSlow,
	}
}
