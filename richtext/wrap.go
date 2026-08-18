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
// then there is no padding to reserve either.
type resolvedChip struct {
	color  color.NRGBA
	pad    int
	radius int
}

// segment is one laid-out fragment of a span: at most one line's worth of
// text (or, for a single word wider than the wrap width, one indivisible
// multi-line block).
type segment struct {
	call   op.CallOp
	x      int // offset within the line
	width  int // the glyphs plus the chip's padding on both sides
	height int
	ascent int
	color  color.NRGBA
	link   int
	strike bool
	chip   resolvedChip
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
		lineTop := overall.Y
		for _, s := range segs {
			// Baseline-align: shift each segment down so all baselines on
			// the line coincide at lineTop+maxAscent.
			off := image.Pt(s.x, lineTop+maxAscent-s.ascent)

			st := op.Offset(off).Push(gtx.Ops)
			if s.chip.color.A > 0 {
				drawChip(gtx, s)
			}
			// The glyphs sit inside the chip's padding; without a chip the
			// offset is zero and the two coincide.
			gt := op.Offset(image.Pt(s.chip.pad, 0)).Push(gtx.Ops)
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
		overall.Y = lineTop + maxAscent + maxDescent
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
		// both paddings are taken out of it.
		pad := 2 * span.chip.pad
		res := layoutSpan(gtx, shaper, max(remaining-pad, 0), span)
		width := res.width + pad

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

// drawChip paints the rounded fill a chipped span sits on: the segment's whole
// box, padding included, at the span's own shaped height. Called with the
// segment's origin as the current transform, before the glyphs.
func drawChip(gtx layout.Context, s segment) {
	r := image.Rectangle{Max: image.Pt(s.width, s.height)}
	paint.FillShape(gtx.Ops, s.chip.color, clip.UniformRRect(r, s.chip.radius).Op(gtx.Ops))
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
// segment: a 2 dp stroke in style.FocusColor, padded 2 dp clear of the
// glyphs, matching components/button's ring treatment.
func drawFocusRing(gtx layout.Context, style Style, off image.Point, s segment) {
	pad := gtx.Dp(2)
	r := image.Rectangle{
		Min: off.Sub(image.Pt(pad, pad)),
		Max: off.Add(image.Pt(s.width+pad, s.height+pad)),
	}
	paint.FillShape(gtx.Ops, style.FocusColor, clip.Stroke{
		Path:  clip.Rect(r).Path(),
		Width: float32(gtx.Dp(2)),
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
// (colour + outlines) into a macro whose origin is the segment's top-left.
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
// line break, when the buffer fills, or when processing stops. The line
// slice's backing array is reused across calls to keep glyph buffering off
// the heap.
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
			t.Pop()
			line = line[:0]
		}
	}
	return line, keep
}
