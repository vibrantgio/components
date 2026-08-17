package list

import "gioui.org/layout"

// Programmatic viewport movement, in pixels.
//
// [State.Reveal] moves the viewport to a row, which is the right unit for a
// list of rows: a selection lands on one. A reader moving through a document
// wants the other unit — a page is a viewport less an overlap, and a viewport
// is a pixel height, not a row count. Rows of wildly different heights make the
// two units disagree badly: layout.List.ScrollBy divides the estimated content
// length by the number of rows, so on a list whose rows are a one-line heading
// and a forty-line code fence "one page" resolves to whatever the average row
// happens to be. Pixels are exact where that average is not, so the movement
// below speaks pixels and leaves rows to Reveal.

// endOffset is the leading-edge offset [State.ScrollToEnd] asks for: further
// into the last row than any content can be. The next layout clamps a leading
// offset that would show empty space past the end back to the exact trailing
// edge, so asking for an unreachable offset is how a caller says "as far as
// this goes" without having to know how tall the content is.
const endOffset = 1 << 30

// record stores the viewport the current frame lays out in, so a later
// [State.Viewport] can answer in pixels. Every entry point calls it before
// laying out.
func (s *State) record(gtx layout.Context) {
	s.viewport = s.l.Axis.Convert(gtx.Constraints.Max).X
}

// Viewport returns the main-axis size in pixels of the viewport the most
// recent layout ran in, and zero before the first layout. It is what a caller
// paging the list by a viewport measures its page against.
func (s *State) Viewport() int { return s.viewport }

// Position returns the scroll position the most recent layout resolved: which
// row leads the viewport, how far into it the leading edge sits, how many rows
// are laid out, and the estimated total length. It is a snapshot — writing to
// the returned value moves nothing.
func (s *State) Position() layout.Position { return s.l.Position }

// ScrollPixels moves the viewport dy pixels along the list: positive toward the
// end, negative toward the start. The move takes effect on the next layout,
// which is also what bounds it — the list stops at the first row's leading edge
// and at the last row's trailing edge, so a caller may ask for more than there
// is and get the end rather than a viewport of nothing.
//
// A list shorter than its viewport therefore does not move at all, in either
// direction.
func (s *State) ScrollPixels(dy int) {
	if dy == 0 {
		return
	}
	s.l.Position.Offset += dy
	// Without this the list ignores Offset while it is pinned at the end, and
	// a caller cannot page back up from the last screen.
	s.l.Position.BeforeEnd = true
}

// ScrollToStart puts the leading edge of the first row at the leading edge of
// the viewport.
func (s *State) ScrollToStart() { s.l.ScrollTo(0) }

// ScrollTo puts the leading edge of row i at the leading edge of the viewport,
// and a negative i at the start. It is not [State.Reveal]: Reveal moves the
// short way and leaves a row that is already visible alone, which is what a
// selection wants; this one puts the row at the top whether it was on screen or
// not, which is what following a table of contents wants.
//
// The move takes effect on the next layout, which is also what bounds it: an i
// past the last row lands on the content's end rather than on a viewport of
// nothing. The selection is untouched.
func (s *State) ScrollTo(i int) {
	if i < 0 {
		i = 0
	}
	s.l.ScrollTo(i)
}

// ScrollToEnd puts the trailing edge of the last of n rows at the trailing edge
// of the viewport, so the viewport shows the content's end and no empty space
// past it. n is the caller's item count, the same number the next layout is
// given.
//
// It is not layout.List.ScrollToEnd, which pins the list to the end for as long
// as it is set. This moves the viewport once, and the reader can page back.
func (s *State) ScrollToEnd(n int) {
	if n <= 0 {
		s.ScrollToStart()
		return
	}
	s.l.Position.First = n - 1
	s.l.Position.Offset = endOffset
	s.l.Position.BeforeEnd = true
}
