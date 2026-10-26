package list

import "gioui.org/layout"

// rowHeights holds the height in pixels of each row the list has laid out,
// by index. The list reads the content's length and the scrolled distance
// from it, so the bar's thumb keeps one length and moves in proportion to
// the scroll however much the rows' heights differ.
type rowHeights struct {
	// h is the height of row i, or -1 for a row not yet laid out. Its
	// length is the row count it was recorded for.
	h []int
	// count and sum are the number and the total of the heights recorded.
	count, sum int
}

// reset clears the record when the row count is not n: an index then names
// a different row.
func (r *rowHeights) reset(n int) {
	if len(r.h) == n {
		return
	}
	if cap(r.h) >= n {
		r.h = r.h[:n]
	} else {
		r.h = make([]int, n)
	}
	for i := range r.h {
		r.h[i] = -1
	}
	r.count, r.sum = 0, 0
}

// set records row i's height.
func (r *rowHeights) set(i, h int) {
	if i < 0 || i >= len(r.h) {
		return
	}
	if old := r.h[i]; old >= 0 {
		r.sum -= old
	} else {
		r.count++
	}
	r.h[i] = h
	r.sum += h
}

// recorded wraps rowFn so that the height of every row it lays out is
// recorded in r, which is first reset for n rows.
func (r *rowHeights) recorded(n int, rowFn layout.ListElement) layout.ListElement {
	r.reset(n)
	return func(gtx layout.Context, i int) layout.Dimensions {
		d := rowFn(gtx, i)
		r.set(i, d.Size.Y)
		return d
	}
}

// span returns the visible part of the content as fractions of the
// content's length: start is the scrolled distance ÷ the content's length,
// end is (the scrolled distance + visible) ÷ the content's length, both
// clamped to [0, 1]. visible is the list's height in pixels.
//
// The content's length is the sum of the rows' heights and the gaps between
// them. The scrolled distance is the heights and gaps before the first row in
// view plus the list's offset into that row.
//
// When heightOf is not nil, it gives every row's height, laid out or not.
// When it is nil, a row laid out counts at its recorded height and a row not
// yet laid out counts at the mean height of the rows laid out. With no row
// laid out the whole content then counts as visible: (0, 1).
func (r *rowHeights) span(pos layout.Position, gap, visible int, heightOf func(i int) int) (start, end float32) {
	n := len(r.h)
	if n == 0 {
		return 0, 1
	}
	var height func(i int) float64
	var content float64
	if heightOf != nil {
		height = func(i int) float64 { return float64(heightOf(i)) }
		for i := range n {
			content += height(i)
		}
	} else {
		if r.count == 0 {
			return 0, 1
		}
		mean := float64(r.sum) / float64(r.count)
		height = func(i int) float64 {
			if h := r.h[i]; h >= 0 {
				return float64(h)
			}
			return mean
		}
		content = float64(r.sum) + float64(n-r.count)*mean
	}
	content += float64(gap * (n - 1))
	if content <= 0 {
		return 0, 1
	}
	scrolled := float64(pos.Offset)
	for i := range min(pos.First, n) {
		scrolled += height(i) + float64(gap)
	}
	return clamp1(scrolled / content), clamp1((scrolled + float64(visible)) / content)
}

// clamp1 limits v to [0, 1].
func clamp1(v float64) float32 {
	return float32(min(max(v, 0), 1))
}
