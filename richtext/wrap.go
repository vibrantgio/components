package richtext

// Paragraph wrapping over Gio's text.Shaper. The algorithm follows the
// gioui.org/x/styledtext reference (evaluated 2026-07-20, DESIGN-v1.md §Markdown;
// reference material only, not a dependency): each span is shaped one line at
// a time with MaxLines=1 and a zero-width-space truncator against the width
// remaining on the current line, so the number of runes that fit can be read
// back; a span that does not fit entirely is split at that rune count and its
// remainder continues on the next line. Unlike the reference, committed lines
// are baseline-aligned: every segment on a line shares one baseline instead
// of being top-aligned, so mixed-size spans read as one line of text.
//
// Committed lines also occupy the paragraph's line box rather than their own
// shaped extent. The box is [Style].LineHeight; the surplus over the line's
// tallest ascent and descent is split half above and half below them, which is
// what makes the pitch between two lines the box and not the metrics, and what
// leaves the box alone when a smaller span joins a line.

import (
	"image"
	"image/color"
	"unicode/utf8"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/internal/focus"

	// fixed is part of Gio's text API surface (text.Parameters.PxPerEm and
	// text.Glyph metrics are fixed.Int26_6); it is already required by
	// gioui.org itself and introduces no new third-party dependency.
	"golang.org/x/image/math/fixed"
)

// resolvedSpan is a SpanStyle with the paragraph Style's defaults applied and
// its link group resolved.
type resolvedSpan struct {
	font    font.Font
	size    int // shaped px (gtx.Sp applied)
	color   color.NRGBA
	content string
	link    int // index in link order; -1 for non-link spans
	url     string
	strike  bool
	chip    resolvedChip
}

// resolvedChip is a [Chip] in pixels. A zero-alpha colour draws none, and
// then there is no padding to reserve either. A zero-alpha border draws no
// hairline; the fill is drawn either way.
type resolvedChip struct {
	color  color.NRGBA
	border color.NRGBA
	pad    int
	radius int
}

// segment is one laid-out fragment of a span: at most one line's worth of
// text (or, for a single word wider than the wrap width, one indivisible
// multi-line block).
type segment struct {
	call   op.CallOp
	x      int // offset within the line
	width  int // the glyphs plus whatever chip padding the fragment spends
	height int
	ascent int
	color  color.NRGBA
	link   int
	strike bool
	chip   resolvedChip
	// padL is the leading chip padding this fragment spends: the chip's
	// padding, or zero where the fragment holds that edge flush (see
	// spendPadding) and zero without a chip. The fill covers the whole width,
	// so the glyphs begin at padL; what the fragment spends on its right is
	// already in width.
	padL int
}

// resolve applies the paragraph defaults to every span, groups consecutive
// spans sharing a URL into links, and bakes the hover treatment for the
// hovered link into the resolved colour (glyph colour is recorded at shaping
// time). It returns the resolved spans and the number of links.
func resolve(gtx layout.Context, style Style, spans []SpanStyle, rs RenderState) ([]resolvedSpan, int) {
	out := make([]resolvedSpan, 0, len(spans))
	nLinks := 0
	prevURL := ""
	current := -1
	for _, s := range spans {
		if s.Content == "" {
			continue
		}
		link := -1
		if s.URL != "" {
			if s.URL == prevURL && current >= 0 {
				link = current
			} else {
				link = nLinks
				nLinks++
			}
		}
		current = link
		prevURL = s.URL

		col := s.Color
		if col == (color.NRGBA{}) {
			if link >= 0 {
				col = style.LinkColor
			} else {
				col = style.Color
			}
		}
		if link >= 0 && link == rs.HoveredLink {
			col = hoverBlend(col)
		}
		size := s.Size
		if size == 0 {
			size = style.Size
		}
		chip := resolvedChip{}
		if s.Chip.Color.A > 0 {
			chip = resolvedChip{
				color:  s.Chip.Color,
				border: s.Chip.Border,
				pad:    gtx.Dp(s.Chip.Padding),
				radius: gtx.Dp(s.Chip.Radius),
			}
		}
		out = append(out, resolvedSpan{
			font:    s.Font(),
			size:    gtx.Sp(size),
			color:   col,
			content: s.Content,
			link:    link,
			url:     s.URL,
			strike:  s.Strikethrough,
			chip:    chip,
		})
	}
	return out, nLinks
}

// hoverBlend is the hover treatment for link text: a ~10% white overlay,
// matching components/button's hover feedback.
func hoverBlend(base color.NRGBA) color.NRGBA {
	const a = float32(0x1a) / 255
	return color.NRGBA{
		R: uint8(float32(base.R)*(1-a) + 0xff*a + 0.5),
		G: uint8(float32(base.G)*(1-a) + 0xff*a + 0.5),
		B: uint8(float32(base.B)*(1-a) + 0xff*a + 0.5),
		A: base.A,
	}
}

// draw lays out the wrapped paragraph and paints it. rs selects the hover and
// focus treatments. When state is non-nil (the live path), each link segment
// additionally registers its pointer/click area, the pointer cursor, its
// focus tag, and link semantics.
func draw(gtx layout.Context, shaper *text.Shaper, style Style, spans []SpanStyle, rs RenderState, state *State) layout.Dimensions {
	resolved, nLinks := resolve(gtx, style, spans, rs)
	if state != nil {
		state.sync(resolved, nLinks)
	}
	if len(resolved) == 0 {
		return layout.Dimensions{}
	}

	maxWidth := gtx.Constraints.Max.X
	// The paragraph's line box, in pixels. Zero — the style names no line
	// height — leaves every line its own shaped metrics.
	lineBox := 0
	if style.LineHeight > 0 {
		lineBox = gtx.Sp(style.LineHeight)
	}

	var (
		segs      []segment
		lineWidth int
		overall   image.Point
		baseline  int // document y of the last committed line's baseline
	)

	commitLine := func() {
		if len(segs) == 0 {
			return
		}
		maxAscent, maxDescent := 0, 0
		for _, s := range segs {
			if s.ascent > maxAscent {
				maxAscent = s.ascent
			}
			if d := s.height - s.ascent; d > maxDescent {
				maxDescent = d
			}
		}
		// Half-leading: whatever the line box has over the tallest ascent and
		// descent on the line is split evenly around them, half above the
		// ascent and half below the descent, so the ink sits in the middle of
		// its box on every line including the first and the last. The half
		// above is rounded down. A box no taller than the metrics leaves both
		// halves zero, which is the metrics-only layout a paragraph with no
		// line height keeps.
		above, below := 0, 0
		if lead := lineBox - (maxAscent + maxDescent); lead > 0 {
			above = lead / 2
			below = lead - above
		}
		lineTop := overall.Y + above
		for _, s := range segs {
			// Baseline-align: shift each segment down so all baselines on
			// the line coincide at lineTop+maxAscent.
			off := image.Pt(s.x, lineTop+maxAscent-s.ascent)

			st := op.Offset(off).Push(gtx.Ops)
			if s.chip.color.A > 0 {
				drawChip(gtx, s)
			}
			// The glyphs sit inside the padding the fragment spends; where it
			// spends none — no chip, or an edge held flush — the offset is
			// zero and the two coincide.
			gt := op.Offset(image.Pt(s.padL, 0)).Push(gtx.Ops)
			s.call.Add(gtx.Ops)
			gt.Pop()
			if s.link >= 0 {
				drawUnderline(gtx, s)
			}
			if s.strike {
				drawStrikethrough(gtx, s)
			}
			st.Pop()

			if s.link >= 0 && s.link == rs.FocusedLink {
				drawFocusRing(gtx, style, off, s)
			}
			if state != nil && s.link >= 0 && s.link < len(state.links) {
				registerLinkArea(gtx, state.links[s.link], off, s)
			}
		}
		if lineWidth > overall.X {
			overall.X = lineWidth
		}
		baseline = lineTop + maxAscent
		overall.Y = lineTop + maxAscent + maxDescent + below
		segs = segs[:0]
		lineWidth = 0
	}

	// work is the mutable queue of spans still to lay out; a split span's
	// remainder replaces its entry and is re-processed on the next line.
	work := make([]resolvedSpan, len(resolved))
	copy(work, resolved)

	for i := 0; i < len(work); {
		span := work[i]
		remaining := maxWidth - lineWidth
		// A chip's padding is reserved rather than drawn over the words
		// beside it, so the glyphs wrap within what is left of the line after
		// the padding this fragment spends is taken out of it. What it spends
		// is what it paints: an edge held flush costs the line nothing.
		padL, padR := spendPadding(work, i, lineWidth == 0)
		res := layoutSpan(gtx, shaper, max(remaining-padL-padR, 0), span)
		width := res.width + padL + padR

		// The span's first segment does not fit and the line already holds
		// content: commit the line and retry on a fresh one. (On an empty
		// line the over-wide content is kept anyway — it will not fit on the
		// next line either.)
		if lineWidth > 0 && width > remaining {
			commitLine()
			continue
		}

		segs = append(segs, segment{
			call:   res.call,
			x:      lineWidth,
			width:  width,
			height: res.height,
			ascent: res.ascent,
			color:  span.color,
			link:   span.link,
			strike: span.strike,
			chip:   span.chip,
			padL:   padL,
		})
		lineWidth += width

		if res.multiLine {
			// Continue the split span on the next line.
			work[i].content = span.content[byteLen(span.content, res.runes):]
			commitLine()
			continue
		}
		if res.endedWithNewline {
			commitLine()
		}
		i++
	}
	commitLine()

	overall = gtx.Constraints.Constrain(overall)
	return layout.Dimensions{Size: overall, Baseline: overall.Y - baseline}
}

// spendPadding decides how much of a chip's padding the fragment of work[i]
// about to be placed actually spends, left and right. lineStart reports that
// the fragment begins a line.
//
// The padding is reserved space — the wrapping counts it so the words beside
// the fill clear it — and at two edges that reservation is itself the defect:
//
//   - A fragment that begins a line has no word to its left to clear, only the
//     margin. Spending the leading padding there sets the fragment's glyphs a
//     padding right of every unchipped line's first glyph, and a list whose
//     items open with a chip stair-steps down its own left edge. So a
//     line-initial fragment holds its left flush: fill and glyphs both start at
//     the margin.
//   - Closing punctuation belongs to the word before it and is set tight
//     against it. A padding between the two reads as a space nobody typed
//     (`MessageOp`; renders as "MessageOp ;"), so a chip whose next span opens
//     with such a mark holds its right flush and the mark hugs the fill.
//
// Both are decided here, at wrap time, rather than at paint time, because the
// reservation and the paint must stay the same number: what a fragment does
// not spend, the line does not count.
func spendPadding(work []resolvedSpan, i int, lineStart bool) (padL, padR int) {
	pad := work[i].chip.pad
	padL, padR = pad, pad
	if lineStart {
		padL = 0
	}
	if pad > 0 && i+1 < len(work) {
		next := work[i+1]
		// A following span on a chip of its own is left alone: two fills set
		// flush against each other read as one long chip, which is a worse
		// defect than the one being fixed here.
		if next.chip.color.A == 0 {
			if r, _ := utf8.DecodeRuneInString(next.content); closesTight(r) {
				padR = 0
			}
		}
	}
	return padL, padR
}

// closesTight reports whether r is closing punctuation: a mark that carries no
// space of its own and is set against the word it follows.
//
// The set is deliberately small — the ASCII sentence and clause marks, the
// ASCII closing brackets, the straight quotes, and the closers that typography
// substitutes for them (’ ” » › …). Openers are absent by design: '(' before a
// chip belongs to the chip's word, and the chip's leading padding is what keeps
// it off the fill. So is the dash, which reads as a separator between words and
// wants the space.
func closesTight(r rune) bool {
	switch r {
	case '.', ',', ';', ':', '!', '?', ')', ']', '}', '\'', '"',
		'’', '”', '»', '›', '…':
		return true
	}
	return false
}

// chipEdge is how wide a chip's hairline is drawn when [Chip].Border asks for
// one: one dp, the width every other line in this library is drawn at.
const chipEdge = unit.Dp(1)

// drawChip paints the rounded fill a chipped span sits on: the segment's whole
// box, padding included, at the span's own shaped height. Called with the
// segment's origin as the current transform, before the glyphs.
//
// A border with any alpha in it edges the chip the way a fence is edged: the
// border colour fills the whole rounded box and the chip colour fills a box
// one hairline smaller, concentric with it, so the rim is drawn without a
// stroke and without a seam. A stroke would be centred on the path and spill
// half its width outside the box the line's wrapping reserved, which is a
// pixel of rim under the word beside it.
func drawChip(gtx layout.Context, s segment) {
	box := image.Rectangle{Max: image.Pt(s.width, s.height)}
	chip := clip.UniformRRect(box, s.chip.radius)
	if s.chip.border.A > 0 {
		edge := max(gtx.Dp(chipEdge), 1)
		paint.FillShape(gtx.Ops, s.chip.border, chip.Op(gtx.Ops))
		chip = clip.UniformRRect(box.Inset(edge), max(s.chip.radius-edge, 0))
	}
	paint.FillShape(gtx.Ops, s.chip.color, chip.Op(gtx.Ops))
}

// drawUnderline paints the link underline for one segment, in the segment's
// text colour, one dp below the baseline. Called with the segment's origin as
// the current transform.
func drawUnderline(gtx layout.Context, s segment) {
	th := max(gtx.Dp(1), 1)
	y := s.ascent + max(gtx.Dp(1), 1)
	paint.FillShape(gtx.Ops, s.color, clip.Rect{
		Min: image.Pt(0, y),
		Max: image.Pt(s.width, y+th),
	}.Op())
}

// drawStrikethrough paints a horizontal line through one segment's glyphs, in
// the segment's text colour, at a quarter of the ascent above the baseline
// (approximately the middle of the x-height). Called with the segment's origin
// as the current transform.
func drawStrikethrough(gtx layout.Context, s segment) {
	th := max(gtx.Dp(1), 1)
	y := s.ascent * 3 / 4
	paint.FillShape(gtx.Ops, s.color, clip.Rect{
		Min: image.Pt(0, y),
		Max: image.Pt(s.width, y+th),
	}.Op())
}

// drawFocusRing paints the visible keyboard-focus ring around a focused link
// segment: a stroke of the library's ring width in style.FocusColor, padded
// the same distance clear of the glyphs.
//
// A link has neither fill nor border to promote, so its ring is drawn beside
// the ink rather than at an edge — and the pad is what keeps that the same
// idiom rather than a second one. The ring and the link ink are both primary,
// close in depth, and a ring laid straight onto the glyphs would read as a
// box around a word in one colour; the clear ground between them is what
// separates the ring from the thing it circles, and it is that ground the
// ring's contrast is measured against.
func drawFocusRing(gtx layout.Context, style Style, off image.Point, s segment) {
	w := gtx.Dp(focus.Width)
	pad := w
	r := image.Rectangle{
		Min: off.Sub(image.Pt(pad, pad)),
		Max: off.Add(image.Pt(s.width+pad, s.height+pad)),
	}
	// Corners turn at the ring's own width. Every other ring in the library
	// takes the corner of the control it marks, and a paragraph has none to
	// take — but a square-cornered box around a word does not read as one of
	// the same family; it reads as a selection or a debug outline. Rounding by
	// the stroke width is the one radius the ring can name out of itself.
	rr := clip.RRect{Rect: r, SE: w, SW: w, NE: w, NW: w}
	paint.FillShape(gtx.Ops, style.FocusColor, clip.Stroke{
		Path:  rr.Path(gtx.Ops),
		Width: float32(w),
	}.Op())
}

// registerLinkArea registers one link segment's interactive area: the hover
// pointer cursor, the click gesture, the focus/event tag (making the link
// Tab-focusable), and screen-reader semantics. A link wrapped across lines
// registers one area per segment, all sharing the same tag.
func registerLinkArea(gtx layout.Context, l *linkState, off image.Point, s segment) {
	st := op.Offset(off).Push(gtx.Ops)
	cl := clip.Rect{Max: image.Pt(s.width, s.height)}.Push(gtx.Ops)
	semantic.Button.Add(gtx.Ops)
	semantic.DescriptionOp(l.url).Add(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	l.click.Add(gtx.Ops)
	event.Op(gtx.Ops, l)
	cl.Pop()
	st.Pop()
}

// spanResult is the outcome of shaping one span against the width remaining
// on the current line.
type spanResult struct {
	call   op.CallOp
	width  int
	height int
	ascent int
	// runes is the count of the span's leading runes consumed by this
	// segment.
	runes int
	// multiLine reports that the span did not fit and continues after runes.
	multiLine bool
	// endedWithNewline reports the segment ended at a hard newline.
	endedWithNewline bool
}

// layoutSpan shapes as much of span as fits within maxWidth on one line. If
// nothing fits (a word wider than the line), the span is reshaped without
// the single-line limit so the over-long word wraps mid-word into one
// indivisible multi-line segment.
func layoutSpan(gtx layout.Context, shaper *text.Shaper, maxWidth int, span resolvedSpan) spanResult {
	call, it := shapeSpan(gtx, shaper, maxWidth, span, true)
	runes := it.runes
	total := utf8.RuneCountInString(span.content)
	multiLine := runes < total
	endedWithNewline := it.hasNewline
	if multiLine {
		next, _ := utf8.DecodeRuneInString(span.content[byteLen(span.content, runes):])
		if next == '\n' {
			// The break was a hard newline: swallow it into this segment so
			// the remainder starts after it.
			endedWithNewline = true
			runes++
			multiLine = runes < total
		} else if runes == 0 {
			// Word wider than the line: shape without the line limit.
			call, it = shapeSpan(gtx, shaper, maxWidth, span, false)
			runes = it.runes
			multiLine = runes < total
			endedWithNewline = it.hasNewline
		}
	}
	return spanResult{
		call:             call,
		width:            it.bounds.Dx(),
		height:           it.bounds.Dy(),
		ascent:           it.baseline,
		runes:            runes,
		multiLine:        multiLine,
		endedWithNewline: endedWithNewline,
	}
}

// shapeSpan shapes span.content against maxWidth and records its glyph paint
// (colour + outlines, then bitmap glyphs) into a macro whose origin is the
// segment's top-left.
// With truncate set the shaper is limited to a single line ended by a
// zero-width-space truncator, so the iterator's rune count reveals the line
// break position.
func shapeSpan(gtx layout.Context, shaper *text.Shaper, maxWidth int, span resolvedSpan, truncate bool) (op.CallOp, glyphIter) {
	maxLines := 0
	if truncate {
		maxLines = 1
	}
	macro := op.Record(gtx.Ops)
	paint.ColorOp{Color: span.color}.Add(gtx.Ops)
	shaper.LayoutString(text.Parameters{
		Font:       span.font,
		PxPerEm:    fixed.I(span.size),
		MaxLines:   maxLines,
		MaxWidth:   maxWidth,
		Truncator:  "\u200b", // zero-width space: an invisible truncator
		Locale:     gtx.Locale,
		WrapPolicy: text.WrapWords,
	}, span.content)
	it := glyphIter{maxLines: 1}
	var buf [32]text.Glyph
	line := buf[:0]
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		var cont bool
		line, cont = it.paint(gtx, shaper, g, line)
		if !cont {
			break
		}
	}
	return macro.Stop(), it
}

// byteLen returns the byte length of the first n runes of s.
func byteLen(s string, n int) int {
	i := 0
	for r := 0; r < n && i < len(s); r++ {
		_, sz := utf8.DecodeRuneInString(s[i:])
		i += sz
	}
	return i
}

// glyphIter accumulates the glyphs of one shaped span, tracking logical
// bounds, baseline, and rune count, and painting buffered glyph runs. It is
// specialised to single-line measurement: with the shaper limited to one
// line, the runes counted are exactly those that fit.
type glyphIter struct {
	// maxLines caps the counted lines (always 1 here); linesSeen tracks
	// glyphs flagged FlagLineBreak.
	maxLines  int
	linesSeen int
	// runes counts the runes represented by processed (non-truncator)
	// glyphs.
	runes int
	// hasNewline reports a hard paragraph break inside the processed run.
	hasNewline bool
	// bounds is the logical bounding box of the processed glyphs; baseline
	// is the first line's baseline (== ascent, since shaping starts at
	// y = ascent).
	bounds   image.Rectangle
	baseline int
	// started tracks bounds/baseline initialisation; painted tracks firstX
	// capture; firstX is subtracted from glyph x so the macro starts at 0.
	started bool
	painted bool
	firstX  fixed.Int26_6
	lineOff image.Point
}

// process folds one glyph into the iterator's metrics. It reports whether
// iteration should continue: false at the truncator run or once the line
// limit is reached at a paragraph break.
func (it *glyphIter) process(g text.Glyph) bool {
	lb := image.Rectangle{
		Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
		Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
	}
	if g.Flags&text.FlagTruncator != 0 {
		// A leading truncator means nothing fit on the line at all.
		if it.runes == 0 {
			it.hasNewline = true
		}
		// Keep the vertical extent so a truncator-only line still has
		// height.
		it.bounds.Min.Y = min(it.bounds.Min.Y, lb.Min.Y)
		it.bounds.Max.Y = max(it.bounds.Max.Y, lb.Max.Y)
		return false
	}
	it.runes += int(g.Runes)
	if g.Flags&text.FlagLineBreak != 0 && g.Flags&text.FlagParagraphBreak != 0 {
		it.hasNewline = true
	}
	if it.maxLines > 0 {
		if g.Flags&text.FlagLineBreak != 0 {
			it.linesSeen++
		}
		if it.linesSeen == it.maxLines && g.Flags&text.FlagParagraphBreak != 0 {
			return false
		}
	}
	if !it.started {
		it.started = true
		it.baseline = int(g.Y)
		it.bounds = lb
	} else {
		it.bounds = it.bounds.Union(lb)
	}
	return true
}

// paint buffers processed glyphs and flushes them as outline paths at each
// line break, when the buffer fills, or when processing stops. Bitmap
// glyphs (CBDT/PNG color emoji) are painted after the outline fill, the way
// widget.Label does, so the PNG is not clipped to an empty path or tinted
// by the span colour. The line slice's backing array is reused across calls
// to keep glyph buffering off the heap.
func (it *glyphIter) paint(gtx layout.Context, shaper *text.Shaper, g text.Glyph, line []text.Glyph) ([]text.Glyph, bool) {
	keep := it.process(g)
	if keep {
		if !it.painted {
			it.painted = true
			it.firstX = g.X
		}
		if len(line) == 0 {
			it.lineOff = image.Pt((g.X - it.firstX).Floor(), int(g.Y))
		}
		line = append(line, g)
	}
	if g.Flags&text.FlagLineBreak != 0 || cap(line)-len(line) == 0 || !keep {
		if len(line) > 0 {
			t := op.Offset(it.lineOff).Push(gtx.Ops)
			outline := clip.Outline{Path: shaper.Shape(line)}.Op().Push(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			outline.Pop()
			if call := shaper.Bitmaps(line); call != (op.CallOp{}) {
				call.Add(gtx.Ops)
			}
			t.Pop()
			line = line[:0]
		}
	}
	return line, keep
}
