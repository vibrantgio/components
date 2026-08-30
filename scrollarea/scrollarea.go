// Package scrollarea provides a horizontally scrolling viewport for content
// that must not be reflowed to fit.
//
// It is the counterpart to components/list, and it exists for the opposite
// reason. A list virtualises a tall column: the content is a sequence of rows
// and only the visible ones lay out. A scroll area has one child, lays it out
// at its natural width however wide that is, and shows the slice of it the
// viewport holds. That is what content with its own line breaks needs — a
// fenced code block, a preformatted table, an over-wide diagram — where
// wrapping would change what the content says and clipping would silently
// lose it.
//
// The area claims the horizontal axis only. Its scroll gesture is registered
// with horizontal bounds and an empty vertical range, so a vertical wheel or
// trackpad gesture over the area is never delivered here and keeps flowing to
// whatever scrolls the column the area sits in. Nesting one inside a vertical
// list is therefore the expected arrangement, not a conflict.
//
// The API is immediate-mode, matching components/list and components/scrollbar:
// allocate a State once per scrollable region and reuse it every frame, while a
// Style is a plain snapshot of resolved colours and metrics derived per frame
// (typically via FromTokens).
package scrollarea

import (
	"image/color"

	"gioui.org/gesture"
	"gioui.org/unit"

	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// State holds a scroll area's offset and the geometry the last layout
// resolved. Allocate once per scrollable region and reuse on every frame.
//
// The zero State is ready to use: an area scrolled to its leading edge that
// has not been laid out yet.
type State struct {
	// scroll is the pointer gesture, bound to the horizontal axis.
	scroll gesture.Scroll
	// sb holds the scrollbar gesture state for LayoutScrollbar. It is a
	// scrollbar.State (which embeds gioui.org/widget.Scrollbar) rather than
	// widget.Scrollbar directly so &sb can be passed to scrollbar.Style.Layout
	// while ScrollDistance remains reachable via promotion.
	sb scrollbar.State

	// offset is how far the viewport's leading edge sits into the content,
	// in pixels, always within [0, MaxOffset].
	offset int
	// content and viewport are the widths the most recent layout measured:
	// the child at its natural width, and the window onto it.
	content, viewport int
}

// NewState returns a scroll area State at its leading edge.
func NewState() *State { return &State{} }

// Offset returns how far the viewport's leading edge sits into the content,
// in pixels. It is zero at the leading edge and [State.MaxOffset] at the
// trailing one.
func (s *State) Offset() int { return s.offset }

// SetOffset moves the viewport to px pixels into the content, clamped to
// [0, MaxOffset] against the geometry the last layout measured.
//
// The clamp is why an offset is set after a layout and not before: until a
// frame has measured the content there is no overflow to speak of, so every
// offset clamps to zero. A caller opening a region at a position lays it out
// once — off-screen, or on the frame before — and sets the offset then.
func (s *State) SetOffset(px int) { s.offset = clamp(px, 0, s.MaxOffset()) }

// ScrollBy moves the viewport px pixels along the content: positive toward
// the trailing edge, negative toward the leading one. It clamps exactly as
// [State.SetOffset] does, so a caller may ask for more than there is and get
// the end rather than a viewport of nothing.
func (s *State) ScrollBy(px int) { s.SetOffset(s.offset + px) }

// Content returns the child's natural width in pixels as the most recent
// layout measured it, and zero before the first layout.
func (s *State) Content() int { return s.content }

// Viewport returns the width in pixels of the window the most recent layout
// showed the content through, and zero before the first layout.
func (s *State) Viewport() int { return s.viewport }

// MaxOffset returns the largest offset the content allows: the width that
// does not fit, and zero when all of it does.
func (s *State) MaxOffset() int { return max(s.content-s.viewport, 0) }

// Overflows reports whether the content is wider than the viewport, which is
// the same question as "is there anything to scroll to".
func (s *State) Overflows() bool { return s.MaxOffset() > 0 }

// Fractions returns where the viewport sits on the content as two values in
// [0,1] with start <= end — the form components/scrollbar takes. Content that
// fits reports the whole range (0, 1), which is how a scrollbar is told to
// draw nothing.
func (s *State) Fractions() (start, end float32) {
	if s.content <= 0 || s.viewport >= s.content {
		return 0, 1
	}
	total := float32(s.content)
	return float32(s.offset) / total, float32(s.offset+s.viewport) / total
}

// Style describes how a scroll area is drawn for one frame. Derive defaults
// with FromTokens and override fields as needed.
type Style struct {
	// Fade is the length of the dissolve drawn over a cut edge while
	// content remains hidden past it. The zero value draws no fade, leaving
	// the cut hard.
	Fade unit.Dp
	// FadeColor is the colour the content dissolves into: the surface the
	// area is drawn on, so the hidden end reads as passing under the edge
	// rather than as a band laid over it. The zero value draws no fade.
	FadeColor color.NRGBA
}

// FromTokens derives the default scroll-area look from colour tokens: the
// content dissolves into the window ground over an S4 run at each cut edge.
//
// # The fade takes no storey
//
// A fade is an overlay, not a surface: it does not climb the elevation ladder
// when the thing under it rises. It is painted over the content, inside the
// area's clip, as the area's own ground run out to zero alpha, and its whole
// job is to be indistinguishable from what lies behind the content so the
// hidden end reads as passing under the edge. A fill that rose a rung here
// would paint a lighter band across the content, which is exactly what
// [Style.FadeColor] forbids.
//
// The default names the window ground, SurfaceAt(Level0) — the convention
// every Ground field in this library gives its zero value. It may not name a
// ramp step instead: the paired ramps realize a given step at the same
// perceptual depth from opposite ends, so one step is a storey in one scheme
// and no storey at all in the other, and a fade that dissolved a light page's
// content into #E8E8E8 while the page was #F6F6F6 is a grey smear.
//
// The fade is the affordance, and it is deliberately not a scrollbar. A
// desktop overlay bar is absent at rest — it appears while the content moves
// and fades out a second later — so a bar alone leaves a resting viewport
// saying nothing about the content it is hiding, and horizontal overflow is
// exactly the case a reader has no other way to suspect: a wrapped column
// announces itself by being tall, a clipped line looks like a line that ended.
// The dissolve is on for as long as there is something past the edge, costs no
// layout and no chrome, and stacks with a bar rather than replacing it — see
// [Style.LayoutScrollbar].
//
// Override FadeColor whenever the area sits on something other than the window
// ground — c.SurfaceAt(the host's own storey), never a rung walked from it. The
// fade must match what is behind the content or it reads as a smear.
func FromTokens(c tokens.ColorTokens) Style {
	return Style{
		Fade:      unit.Dp(tokens.Spacing.S4),
		FadeColor: c.SurfaceAt(tokens.Level0),
	}
}

// clamp limits v to [lo, hi]. hi below lo yields lo, which is what an empty
// range means here: nowhere to scroll to.
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return max(hi, lo)
	}
	return v
}
