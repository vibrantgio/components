// Package paragraph draws content as a run of styled text wrapped into lines:
// the links in it are carried controls, the rest is read. It is the Components
// inline primitive — a span model with interactive link spans — and it lives in
// the component library; a goldmark document renderer lives a module above it.
//
// # Entry points
//
// There are two ways to lay out a paragraph, both driven by the same span
// slice:
//
//   - [Layout] is the live path: it drains link events (pointer clicks,
//     keyboard activation, focus changes) from state, fires
//     [Style].OnLinkClick and [Style].OnLink, and renders hover/focus
//     treatments from the real interaction state.
//   - [Render] is the static path: it renders an explicit [RenderState]
//     without any event processing. Intended for golden-image testing and
//     static demonstrations.
//
// # Links and accessibility
//
// A span with a non-empty URL is a hyperlink. Consecutive spans sharing the
// same URL form one link. Links render underlined in [Style].LinkColor, show
// the pointer cursor on hover, and participate in window Tab traversal: each
// link registers a focus tag, so Gio's default Tab handling moves focus
// across them, and the focused link draws a visible focus ring. Space or
// Enter on a focused link fires OnLinkClick, which carries the frame's live
// layout.Context so consumers can emit mvu.MessageOp{Message: ...}.Add(gtx.Ops)
// inside the callback. An inline link is text-sized: its pointer target is the
// run of text it marks, which is what the platform gives a link in prose.
//
// # Zero dependencies
//
// The package is built directly on Gio's text shaper. gioui.org/x/richtext
// and gioui.org/x/styledtext served as reference material for the span-model
// shape and the wrapping algorithm; neither is a dependency.
package paragraph

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/theme/tokens"
)

// SpanStyle describes one run of styled text within a paragraph: the span
// model {font, weight, style, size, colour, link URL, chip}.
type SpanStyle struct {
	// Typeface names the font families to try (e.g. "Go Mono, monospace").
	// Empty uses the shaper's default face.
	Typeface font.Typeface
	// Weight is the font weight. The zero value is font.Normal.
	Weight font.Weight
	// Style is the slant. The zero value is font.Regular.
	Style font.Style
	// Size is the text size. Zero falls back to the paragraph default
	// ([Style].Size, Body when derived via FromTokens).
	Size unit.Sp
	// Color is the text colour. The zero value falls back to the paragraph
	// default: [Style].Color, or [Style].LinkColor when URL is set.
	Color color.NRGBA
	// Content is the text of the span.
	Content string
	// URL, when non-empty, marks the span as a hyperlink. Consecutive spans
	// with the same URL are grouped into a single link for interaction.
	URL string
	// Strikethrough draws a horizontal line through the span's glyphs in the
	// span's text colour (e.g. GFM ~~deleted~~ text).
	Strikethrough bool
	// Chip sets the span on a rounded fill behind its glyphs. The zero value
	// draws none.
	Chip Chip
	// Fills are fields painted behind runs of the span's own glyphs. They
	// are byte ranges into Content, which is what keeps them off the
	// shaping: the span is shaped whole whether it carries fills or not, so
	// filling part of it moves no glyph and rewraps no line.
	Fills []Fill
}

// Fill is a field painted behind one run of a span's glyphs: the run's own
// width and the height of the line box it sits in. It is drawn before
// everything else the span draws — before a [Chip], before the glyphs — so a
// span already sitting on a fill keeps it, and the text stays on top.
//
// A run that wraps is filled on every line it reaches, each line's fragment
// covering that line's box.
type Fill struct {
	// Start and End are byte offsets into [SpanStyle].Content. An empty or
	// reversed range fills nothing, and the range is clamped to the content.
	Start, End int
	// Color fills the run. A zero alpha paints nothing and is reported to
	// [Style].OnFill all the same, which is how a caller asks where a run of
	// its own text landed without marking it: where a run lies is a result
	// of the wrapping and is not knowable before the paragraph is laid out.
	Color color.NRGBA
}

// Chip is the rounded fill a span may sit on: the treatment a reading surface
// gives inline code, which needs to read as a thing quoted out of the prose
// rather than as prose in another face.
//
// It cannot change the height of the line it sits in. The fill takes the
// span's own shaped line box — the ascent and descent its face asks for at its
// size — so a span set smaller than the prose around it sits on a chip shorter
// than the line, and one set at the prose's own size sits on a chip the line's
// own height. Padding is horizontal only, and it is reserved: the line's
// wrapping counts it, so the words on either side clear the fill instead of
// running under it.
//
// Padding is spent against words, though, and at two edges there is no word to
// clear. A chip that begins a line holds its left flush — fill and glyphs both
// start at the margin — so a list whose items open with a chip keeps one left
// edge instead of stair-stepping down it. And a chip whose text is immediately
// followed by closing punctuation (. , ; : ! ? ) ] } ' " ’ ” » › …) holds its
// right flush, so the mark hugs the fill instead of reading as a space nobody
// typed. Neither is a hole in the reservation: what a chip does not spend, the
// line does not count.
//
// A chipped span wide enough to wrap carries the fill onto every line it
// occupies, each fragment reading as its own chip, with the padding each
// fragment's own edges call for.
type Chip struct {
	// Color fills the chip. A zero alpha draws nothing.
	Color color.NRGBA
	// Border strokes a hairline just inside the fill's rounded edge, one dp
	// wide at every density — the same line width every other edge in this
	// library draws. A zero alpha draws none, which is what a fill that
	// stands off its page on its own needs.
	//
	// It is here because a fill does not always stand off its page: the
	// platform's own control fill is the page's own white in the light
	// appearance, and what says where such a chip is has to be its edge. The
	// caller decides whether the fill needs the help; this only draws it.
	Border color.NRGBA
	// Padding is the horizontal space between the fill's edge and the glyphs,
	// on each side.
	Padding unit.Dp
	// Radius rounds the fill's four corners.
	Radius unit.Dp
}

// Font assembles the gio font selector from the span's typeface, slant, and
// weight fields.
func (s SpanStyle) Font() font.Font {
	return font.Font{Typeface: s.Typeface, Style: s.Style, Weight: s.Weight}
}

// Style holds the themed paragraph defaults and the link callback. Derive the
// token-themed default with [FromTokens], then set OnLinkClick.
type Style struct {
	// Color is the text colour for spans with a zero Color.
	Color color.NRGBA
	// LinkColor is the text colour for link spans with a zero Color.
	LinkColor color.NRGBA
	// FocusColor is the focus-ring colour drawn around the focused link.
	FocusColor color.NRGBA
	// HoverColor is laid over a link's own colour while the pointer is on
	// it, in the space the platform composites in. It carries its own
	// coverage, so one value moves a link of any colour in the direction the
	// appearance calls for; it is never painted, only blended, so nothing
	// downstream is handed a coverage.
	HoverColor color.NRGBA
	// Size is the text size for spans with a zero Size.
	Size unit.Sp
	// LineHeight is the height of one line box — the whole box, in the CSS
	// sense, not the gap between lines and not a multiplier. Every wrapped
	// line occupies it whatever its glyphs measure, and the space left over
	// after the tallest ascent and descent on the line is split evenly above
	// and below them, half-leading fashion, on the first and last lines as
	// much as on the ones between. A span set smaller than the paragraph —
	// a quoted word in another face — therefore leaves the box alone: the
	// box is the paragraph's, and the line's own metrics only decide how
	// much leading is left to split.
	//
	// Zero, the default, leaves the box to the shaped metrics: each line is
	// exactly its own tallest ascent plus descent, which is what a paragraph
	// laid out before line heights were honoured measured.
	//
	// A value below the natural line is not a squeeze. There is no leading to
	// distribute at that point, and lines drawn into a box shorter than their
	// glyphs would overlap, so the metrics stand.
	LineHeight unit.Sp
	// OnFill, when non-nil, is called for every [Fill] the layout places —
	// the colourless ones included — with the index of the span the fill
	// belongs to in the slice laid out, the fill's index within that span's
	// Fills, and the rectangle it covers in the paragraph's own coordinates.
	// A fill that wraps is reported once per line.
	//
	// It is how a caller that put a fill somewhere learns where it landed,
	// which is what it takes to scroll to it: the position is a result of
	// wrapping and is not knowable before the paragraph is laid out.
	OnFill func(span, fill int, r image.Rectangle)
	// OnLinkClick is called when a link is activated by pointer click or by
	// Space/Enter while focused. The gtx argument is the live layout.Context
	// active on the frame the activation is processed, allowing consumers to
	// emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback.
	OnLinkClick func(gtx layout.Context, url string)
	// OnLink is called when the link the paragraph is saying the address
	// of changes, and only then: with the link's URL on the frame it is
	// reached, with the empty string on the frame it is left, and with the
	// new URL on a frame one link is left for another. A frame that reaches
	// the same link again calls nothing, so a consumer may do real work in
	// it.
	//
	// A link is reached by the pointer moving onto it or by the keyboard
	// focus landing on it, and left when that pointer or that focus goes.
	// THE POINTER WINS WHILE BOTH STAND: a reader moving the pointer over
	// the prose is reading the address of what is under it, and the link the
	// keyboard left behind is not what they are asking about. A change in
	// either is evaluated against that rule, so focus moving between two
	// links under a resting pointer says nothing and the pointer leaving the
	// prose gives the address back to the focused link.
	//
	// It is how a window says where a link goes while the reader is on it —
	// the address at the foot of the window every browser on this platform
	// shows, for a hovered link and a focused one alike. The gtx argument is
	// the live layout.Context of that frame, so consumers can emit
	// mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the callback as they do
	// from OnLinkClick.
	OnLink func(gtx layout.Context, url string)
}

// FromTokens derives the default paragraph style from the platform's colour
// set and the Body text style: body text in Text, links in Link, the
// focus ring in the one colour every control in this library rings a focused
// element with, and the role's own size and line height. Pass
// tokens.DefaultTypography.Body for the default desktop look.
//
// standsOn is the opaque fill the paragraph is drawn on. The focus ring
// carries a coverage rather than a colour, so what it lands as depends on
// what is under it; the zero value — no colour — is the window's own plane.
//
// Of the role's style Size and LineHeight land in [Style]: a paragraph's
// typeface, weight and slant are per-span properties, carried by each
// [SpanStyle], while the size and the line box belong to the paragraph as a
// whole. A role that names no line height — a zero — leaves the box to the
// shaped metrics, so a typography that carries none renders exactly as it did.
// FromTokens takes the whole [tokens.TextStyle] anyway so the role stays one
// value from theme to paragraph — and takes no [tokens.Density], which sizes
// controls and so has nothing to say about a paragraph.
func FromTokens(p tokens.PlatformColors, body tokens.TextStyle, standsOn color.NRGBA) Style {
	return Style{
		Color:      p.Text,
		LinkColor:  p.Link,
		FocusColor: focus.RingColor(p, surface.Or(standsOn, p.WindowBackground)),
		HoverColor: p.HoverOverlay,
		Size:       unit.Sp(body.Size),
		LineHeight: unit.Sp(body.LineHeight),
	}
}

// NoLink marks the absence of a link index in a [RenderState].
const NoLink = -1

// RenderState holds explicit visual interaction state for static rendering
// via [Render]. Link indices count links (not spans) in document order;
// consecutive spans sharing a URL are one link. Use [Idle] for the state with
// no interaction — the zero value refers to link 0.
type RenderState struct {
	// HoveredLink is the index of the link drawn in its hovered treatment;
	// NoLink for none.
	HoveredLink int
	// FocusedLink is the index of the link drawn with the focus ring; NoLink
	// for none.
	FocusedLink int
}

// Idle returns the RenderState with no link hovered or focused.
func Idle() RenderState {
	return RenderState{HoveredLink: NoLink, FocusedLink: NoLink}
}

// State holds the interaction state of a paragraph's links across frames.
// Allocate once per paragraph instance and reuse on every frame.
type State struct {
	links []*linkState
	// reported is the link last reported to [Style].OnLink, offset by
	// one: zero is no link, i+1 is link i. The offset is what lets the zero
	// value mean nothing has been reported, so a paragraph neither the
	// pointer nor the keyboard has ever been on reports no departure from
	// it.
	reported int
	// reportedURL is that link's URL as reported, so a link whose URL changes
	// under a resting pointer is reported again.
	reportedURL string
}

// NewState returns a State for a paragraph with no interaction history.
func NewState() *State { return &State{} }

// linkState is the per-link persistent interaction state. Its pointer
// identity doubles as the link's focus/event tag, so links keep focus and
// in-flight gestures across frames and re-layouts.
type linkState struct {
	click      gesture.Click
	url        string
	pressedKey key.Name
	// box is where the last draw put this link, in the paragraph's own
	// coordinates: the union of its runs, so a link carried onto a second
	// line reports the block around both. It is reset at the top of every
	// draw and filled as the runs are registered.
	box image.Rectangle
	// runs are the link's runs as the last draw placed them, one per line it
	// reaches, and nruns how many of them that draw filled in. They are kept
	// by index across frames and never dropped, because each run's hover
	// tracks its own area by the address of that entry.
	runs  []*linkRun
	nruns int
}

// linkRun is one line's worth of a link: the rectangle that line's run of the
// link occupies and the hover over that rectangle alone. The pointer stands
// on a run, not on a link, which is what lets a wrapped link report the run
// under the pointer rather than the block around all of them.
type linkRun struct {
	hov gesture.Hover
	box image.Rectangle
	on  bool
}

// run returns the i-th run of this link, growing the slice to reach it. The
// entries are pointers so that growing the slice moves no [gesture.Hover]:
// its address is the tag its area is registered under.
func (l *linkState) run(i int) *linkRun {
	for len(l.runs) <= i {
		l.runs = append(l.runs, &linkRun{})
	}
	return l.runs[i]
}

// sync makes the state track exactly n links, preserving the identity (and
// therefore focus and gesture state) of surviving indices, and records each
// link's URL for event dispatch.
func (s *State) sync(resolved []resolvedSpan, n int) {
	if len(s.links) > n {
		s.links = s.links[:n]
	}
	for len(s.links) < n {
		s.links = append(s.links, &linkState{})
	}
	for _, l := range s.links {
		l.box, l.nruns = image.Rectangle{}, 0
	}
	for _, r := range resolved {
		if r.link >= 0 {
			s.links[r.link].url = r.url
		}
	}
}

// HoveredLink returns the index of the link currently under the pointer, or
// NoLink if none. Valid after the previous frame's Layout.
func (s *State) HoveredLink() int {
	for i, l := range s.links {
		if l.click.Hovered() {
			return i
		}
	}
	return NoLink
}

// HoveredLinkBox returns where the last layout drew the run of the link the
// pointer is on, in the paragraph's own coordinates, and false when no link
// is under the pointer or the paragraph has not been laid out yet.
//
// The box is the run under the pointer and not the union of the link's runs:
// a link carried onto a second line is annotated beside the line the reader
// is pointing at, which is where they are looking. The union answers for the
// keyboard instead, where there is no run to point at; see
// [State.FocusedLinkBox].
//
// It is how a window says where the link under the pointer IS, as
// [Style].OnLink says where it GOES: a tooltip or any other annotation
// standing beside a link needs the rectangle, and the position is a result
// of wrapping and is not knowable before the paragraph is laid out. Read it
// after [Layout] for this frame's geometry; read from inside OnLink it
// holds the previous frame's, which is the layout the pointer was tested
// against.
func (s *State) HoveredLinkBox() (image.Rectangle, bool) {
	for _, l := range s.links {
		if !l.click.Hovered() {
			continue
		}
		for i := 0; i < l.nruns; i++ {
			if r := l.runs[i]; r.on && !r.box.Empty() {
				return r.box, true
			}
		}
		// The pointer is on the link and on none of the runs the last draw
		// placed: the wrapping moved under it on this very frame. The block
		// around all of them is where the link is either way.
		if l.box.Empty() {
			return image.Rectangle{}, false
		}
		return l.box, true
	}
	return image.Rectangle{}, false
}

// FocusedLinkBox returns where the last layout drew the link holding the
// keyboard focus, in the paragraph's own coordinates, and false when no link
// holds it or the paragraph has not been laid out yet.
//
// The box is the union of the link's runs. A link carried onto a second line
// reports the block around all of it, because the keyboard reached the link
// and not one line of it: nothing picks a run when no pointer points at one.
func (s *State) FocusedLinkBox(gtx layout.Context) (image.Rectangle, bool) {
	for _, l := range s.links {
		if !gtx.Focused(l) {
			continue
		}
		if l.box.Empty() {
			return image.Rectangle{}, false
		}
		return l.box, true
	}
	return image.Rectangle{}, false
}

// FocusedLink returns the index of the link currently holding keyboard
// focus, or NoLink if none. Valid after the previous frame's Layout.
func (s *State) FocusedLink(gtx layout.Context) int {
	for i, l := range s.links {
		if gtx.Focused(l) {
			return i
		}
	}
	return NoLink
}

// Layout is the live path: it processes link input (clicks, keyboard
// activation, focus), fires style.OnLinkClick and style.OnLink, and lays
// out the wrapped paragraph with hover and focus treatments from the real
// interaction state.
//
// Spans wrap within gtx.Constraints.Max.X. The returned baseline is that of
// the paragraph's last line.
func Layout(gtx layout.Context, state *State, shaper *text.Shaper, style Style, spans []SpanStyle) layout.Dimensions {
	rs := processInput(gtx, state, style)
	// A focused link's band straddles its glyphs, so it is collected while
	// the run is drawn and painted once the whole paragraph is down.
	dims := focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, style, spans, rs, state)
	})
	// Links created by this frame's draw (their first layout) must
	// register their event filters within the same frame their areas
	// appear, or the router would drop events arriving before the next
	// frame. Draining again is idempotent for links that already
	// registered above.
	processInput(gtx, state, style)
	return dims
}

// Render produces a layout.Widget for a paragraph in an explicit visual
// state, without any event processing. Intended for golden-image testing and
// static demonstrations; production code should use [Layout].
func Render(shaper *text.Shaper, style Style, spans []SpanStyle, s RenderState) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return focus.Around(gtx, func(gtx layout.Context) layout.Dimensions {
			return draw(gtx, shaper, style, spans, s, nil)
		})
	}
}

// processInput drains this frame's events for every link registered on the
// previous frame and dispatches them: pointer click or Space/Enter while
// focused → style.OnLinkClick(gtx, url), and a change of the link under the
// pointer → style.OnLink(gtx, url). It returns the live RenderState
// (hovered and focused link indices) for the subsequent draw.
func processInput(gtx layout.Context, state *State, style Style) RenderState {
	rs := Idle()
	if state == nil {
		return rs
	}
	for i, l := range state.links {
		// Pointer clicks.
		for {
			e, ok := l.click.Update(gtx.Source)
			if !ok {
				break
			}
			if e.Kind == gesture.KindClick && style.OnLinkClick != nil {
				style.OnLinkClick(gtx, l.url)
			}
		}
		// Focus and keyboard activation. Registering the FocusFilter makes
		// the link focusable, so the window's default Tab handling traverses
		// it. Space/Enter activates on release after a press while focused,
		// mirroring widget.Clickable.
		for {
			e, ok := gtx.Event(
				key.FocusFilter{Target: l},
				key.Filter{Focus: l, Name: key.NameReturn},
				key.Filter{Focus: l, Name: key.NameSpace},
			)
			if !ok {
				break
			}
			switch e := e.(type) {
			case key.FocusEvent:
				if e.Focus {
					l.pressedKey = ""
				}
			case key.Event:
				if !gtx.Focused(l) {
					break
				}
				if e.Name != key.NameReturn && e.Name != key.NameSpace {
					break
				}
				switch e.State {
				case key.Press:
					l.pressedKey = e.Name
				case key.Release:
					if l.pressedKey != e.Name {
						break
					}
					l.pressedKey = ""
					if style.OnLinkClick != nil {
						style.OnLinkClick(gtx, l.url)
					}
				}
			}
		}
		for j := 0; j < l.nruns; j++ {
			l.runs[j].on = l.runs[j].hov.Update(gtx.Source)
		}
		if l.click.Hovered() {
			rs.HoveredLink = i
		}
		if gtx.Focused(l) {
			rs.FocusedLink = i
		}
	}
	reportLink(gtx, state, style, rs)
	return rs
}

// reportLink calls style.OnLink on a change of the link the paragraph is
// saying the address of and on nothing else. That link is the one under the
// pointer while there is one and the one holding the focus otherwise; see
// [Style].OnLink for why the pointer wins. The comparison is against
// what the state last reported rather than against the previous frame, which
// is what keeps the two drains [Layout] makes per frame to one call between
// them.
func reportLink(gtx layout.Context, state *State, style Style, rs RenderState) {
	if style.OnLink == nil {
		return
	}
	link, url := 0, ""
	switch {
	case rs.HoveredLink != NoLink:
		link, url = rs.HoveredLink+1, state.links[rs.HoveredLink].url
	case rs.FocusedLink != NoLink:
		link, url = rs.FocusedLink+1, state.links[rs.FocusedLink].url
	}
	if link == state.reported && url == state.reportedURL {
		return
	}
	state.reported, state.reportedURL = link, url
	style.OnLink(gtx, url)
}
