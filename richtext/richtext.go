// Package richtext provides the Components inline styled-text primitive: a span
// model with wrapped paragraph layout and interactive link spans. The
// layering decision that put it here — the inline primitive in the
// component library, the goldmark document renderer a module above — is
// recorded in §Markdown of the design document's first edition,
// https://github.com/vibrantgio/design/blob/master/DESIGN-v1.md.
//
// # Entry points
//
// There are two ways to lay out a paragraph, both driven by the same span
// slice:
//
//   - [Layout] is the live path: it drains link events (pointer clicks,
//     keyboard activation, focus changes) from state, fires
//     [Style].OnLinkClick, and renders hover/focus treatments from the real
//     interaction state.
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
// across them, and the focused link draws a visible focus ring
// (per DESIGN-v1.md §Accessibility). Space or Enter on a focused link fires
// OnLinkClick, which carries the frame's layout.Context per GX.8 so
// consumers can emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the
// callback. Inline links are text-sized: the 44 dp hit-target rule does not
// apply to inline text links (WCAG 2.5.5 inline exception).
//
// # Zero dependencies
//
// The package is built directly on Gio's text shaper. gioui.org/x/richtext
// and gioui.org/x/styledtext served as reference material for the span-model
// shape and the wrapping algorithm; neither is a dependency.
package richtext

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/internal/focus"
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
	// ([Style].Size, BodyLarge when derived via FromTokens).
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
	// It is here because a fill does not always stand off its page. Since
	// ADR-022 the elevation ladder climbs toward the light in both schemes,
	// and a light scheme has almost no room above its paper to climb into:
	// a raised chip there is a whisper — a fraction of a step — and what
	// says where it is has to be its edge. A chip carries no shadow and
	// takes no storey of its own beyond that whisper, so the edge is the
	// only thing left to say it. The caller decides whether the fill needs
	// the help; this only draws it.
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
	// ink would overlap, so the metrics stand.
	LineHeight unit.Sp
	// OnLinkClick is called when a link is activated by pointer click or by
	// Space/Enter while focused. The gtx argument is the layout.Context
	// active on the frame the activation is processed (GX.8), allowing
	// consumers to emit mvu.MessageOp{Message: ...}.Add(gtx.Ops) inside the
	// callback.
	OnLinkClick func(gtx layout.Context, url string)
}

// FromTokens derives the default paragraph style from colour tokens and the
// BodyLarge text style: body text in Text at body.Size, lines in the role's own
// line height, links in Primary, and the focus ring in the one colour every
// control in this library rings a focused element with — the rung of the
// primary ramp measured to clear the non-text contrast floor against the
// paragraph ground the ring is drawn on. Pass
// tokens.DefaultTypography.BodyLarge for the default desktop look.
//
// Of the role's style Size and LineHeight land in [Style]: a paragraph's
// typeface, weight and slant are per-span properties, carried by each
// [SpanStyle], while the size and the line box belong to the paragraph as a
// whole. A role that names no line height — a zero — leaves the box to the
// shaped metrics, so a typography that carries none renders exactly as it did.
// FromTokens takes the whole [tokens.TextStyle] anyway so the role stays one
// value from theme to paragraph — and takes no [tokens.Density], which sizes
// controls and so has nothing to say about a paragraph.
func FromTokens(c tokens.ColorTokens, body tokens.TextStyle) Style {
	return Style{
		Color:     c.Text,
		LinkColor: c.Primary,
		// The ground a link's ring is drawn on is the paper the paragraph
		// is set on — the ladder's level 0, asked of the palette. It used
		// to name c.Surface, which is a neutral-ramp alias rather than a
		// storey (ADR-022); over the whole seed sweep the two answer the
		// same rung, so this moves no pixel and stops claiming that prose
		// lies on the ramp's first rung.
		FocusColor: focus.Ring(c, focus.Ground(c, tokens.Level0)),
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
// activation, focus), fires style.OnLinkClick, and lays out the wrapped
// paragraph with hover and focus treatments from the real interaction state.
//
// Spans wrap within gtx.Constraints.Max.X. The returned baseline is that of
// the paragraph's last line.
func Layout(gtx layout.Context, state *State, shaper *text.Shaper, style Style, spans []SpanStyle) layout.Dimensions {
	rs := processInput(gtx, state, style)
	dims := draw(gtx, shaper, style, spans, rs, state)
	// Links created by this frame's draw (their first layout) must register
	// their event filters within the same frame their areas appear, or the
	// router would drop events arriving before the next frame. Draining
	// again is idempotent for links that already registered above.
	processInput(gtx, state, style)
	return dims
}

// Render produces a layout.Widget for a paragraph in an explicit visual
// state, without any event processing. Intended for golden-image testing and
// static demonstrations; production code should use [Layout].
func Render(shaper *text.Shaper, style Style, spans []SpanStyle, s RenderState) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, style, spans, s, nil)
	}
}

// processInput drains this frame's events for every link registered on the
// previous frame and dispatches them: pointer click or Space/Enter while
// focused → style.OnLinkClick(gtx, url). It returns the live RenderState
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
		if l.click.Hovered() {
			rs.HoveredLink = i
		}
		if gtx.Focused(l) {
			rs.FocusedLink = i
		}
	}
	return rs
}
