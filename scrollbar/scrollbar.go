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

	tcolor "github.com/vibrantgio/theme/color"
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

// Contrast floors the thumb is derived against.
//
// At rest the thumb is a graphic that carries meaning without being text —
// nothing about the position it reports is spelled out anywhere — so it owes
// the page WCAG 1.4.11's 3:1 and no more. Under the pointer it owes more: a
// hovered or dragged thumb has stopped reporting a position and become a
// target, something the reader is aiming at rather than glancing at, so that
// state takes WCAG 1.4.3's 4.5:1 text floor instead. One derivation, two
// floors, and that difference is the whole of what separates the two states.
const (
	restFloor   = 3.0
	activeFloor = 4.5
)

// The coverage the overlay intends: 39% at rest, 67% while hovered or
// dragged — the alphas this bar has always been drawn at, and the reason it
// is an overlay at all. They are the least the thumb is ever covered by
// rather than a setting: the derivation raises coverage when a ground asks
// for it and never lowers it, so a scheme whose thumb already clears its
// floor keeps exactly the bar it had.
const (
	restCoverage   = 100
	activeCoverage = 170
)

// inkStep is where the thumb's ink starts: ADR-007's low-contrast-text step,
// which is what chrome that must be noticed without being read is drawn in.
// The derivation walks deeper from here and never shallower.
const inkStep = 700

// thumbInk derives one of the thumb's two states: the neutral ramp's
// low-contrast-text rung deepened as far as floor demands over the grounds
// this bar rides, and coverage raised past the overlay's intent only once
// the ramp's deepest ink still falls short.
//
// The order the two dials are spent in is the design. Deepening the ink
// costs a reader nothing — two inks that reach the same composited contrast
// over the same ground *are* the same colour on that ground — while raising
// coverage costs precisely what an overlay is for, since coverage is how
// much of the content underneath stops showing through. So the ink is spent
// first, all the way to the ramp's end, and coverage only after; for a given
// floor that is the most translucent thumb there is. Concretely, the light
// scheme's resting thumb reaches 3:1 at 82% coverage in the low-contrast-text
// rung and at 71% in the ramp's end rung, and the two land on the same grey.
//
// It is derived against both grounds an overlay bar rides — the window's own
// page and the furniture the panes are filled with — and answers whichever
// asks more, so one bar reads on both. Which storey the second one is, is
// ADR-022's: chrome furniture is the floor beneath the paper rather than a
// storey above it, and the floor is the harder of the two grounds in both
// schemes for the same reason in each — the thumb's ink is dark, the floor
// is the darker ground, and a dark ink over a darker ground has the less to
// spare. Handing the bar its ground the way the AQ-era components take one
// would buy little here: over the whole seed sweep the two grounds never
// disagree about the rung and differ only in coverage, because an ink at the
// ramp's end is far from both of them. A bar riding a raised or floating
// surface is another matter, and the ground is a parameter of this function,
// so that day costs a call-site argument and not a redesign.
//
// The measurement is taken over the composite, not over the ink: a
// translucent colour has no contrast of its own, and reading the ink's own
// ratio off the ramp is exactly how a thumb measuring 1.49:1 against the
// light page came to be believed legible.
//
// A floor no ink at any coverage reaches is answered with the ramp's end
// rung, opaque, so a caller always has a colour: a thumb too weak for its
// floor is a contrast defect the gates report, not a reason to paint an
// unset colour.
func thumbInk(c tokens.ColorTokens, floor float64, coverage uint8) color.NRGBA {
	grounds := [...]color.NRGBA{
		c.SurfaceAt(tokens.Level0),
		c.SurfaceAt(tokens.LevelFloor),
	}
	clears := func(ink color.NRGBA) bool {
		for _, ground := range grounds {
			if tcolor.ContrastRatio(tcolor.Over(ink, ground), ground) < floor {
				return false
			}
		}
		return true
	}
	for step := inkStep; step <= 900; step += 100 {
		ink := c.Ramps.Neutral.Step(step)
		ink.A = coverage
		if clears(ink) {
			return ink
		}
	}
	ink := c.Ramps.Neutral.Step(900)
	for a := int(coverage); a < 255; a++ {
		ink.A = uint8(a)
		if clears(ink) {
			return ink
		}
	}
	ink.A = 255
	return ink
}

// FromTokens derives the default scrollbar look from colour tokens.
//
// The thumb is translucent, and that translucency is its identity — content
// shows through an overlay bar — not an MD3 state layer, so it survives
// ADR-007's move to ramp-step states. What the translucency is not is a free
// choice: a translucent ink has no colour until it is composited, and the
// composite of the low-contrast-text step at 39% coverage over the light
// page is #CCCCCC, which measures 1.49:1 against that page. A bar at 1.49:1
// is invisible exactly when a reader looks for it, which is the one moment it
// exists for. So both states are derived instead — see thumbInk — as the
// most translucent thumb that still clears its floor over the grounds an
// overlay bar rides.
//
// The two schemes answer differently and the difference is the derivation
// working rather than a special case. Over a dark page a pale ink at low
// coverage lifts the composite a long way, so the dark scheme keeps the
// low-contrast-text step at the coverage it always had and measures 4.49:1.
// Over a light page the same arithmetic runs backwards — a light ground
// dominates a linear-light blend — so the light scheme spends the ink all
// the way to the ramp's end and then buys the rest with coverage, landing at
// 71% for 3:1 and 84% for 4.5:1. Translucency is simply dearer on a light
// page than on a dark one, and one rule pricing it correctly is what tells
// the two apart. The track is transparent by default.
//
// The bar also fades: it is fully present while the content scrolls or the
// pointer is on the gutter, then fades out a second after the last of
// either, which is how the desktop platforms' overlay scrollbars behave.
// The floors above are what the bar owes while it is present; a bar in the
// middle of fading out is on its way to invisible on purpose. The gutter
// keeps its hit areas while faded, so moving the pointer onto it brings the
// bar back. Set FadeDelay to zero for a bar that stays visible.
func FromTokens(c tokens.ColorTokens) Style {
	return Style{
		ThumbColor:        thumbInk(c, restFloor, restCoverage),
		ThumbHoverColor:   thumbInk(c, activeFloor, activeCoverage),
		TrackColor:        color.NRGBA{},
		ThumbMinorWidth:   6,
		TrackPadding:      2,
		ThumbCornerRadius: 3,
		ThumbMinLen:       16,
		FadeDelay:         time.Second,
		FadeDuration:      tokens.Motion.DurSlow,
	}
}
