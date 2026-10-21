// Package elide shortens a line of text to the width it has to stand in.
//
// It exists because a line that does not fit is shortened differently
// depending on what the line is. Prose is cut at the end, which is what
// [gioui.org/widget.Label] does for every control in this library. An
// address is not prose: its ends carry the host and the leaf, which is what
// a reader is looking for, and the middle is the part that can go. So the
// platform elides an address in the middle, leaving the start and the end
// standing on either side of one ellipsis, and that is what this package
// does.
//
// The width is measured with theme/typeset, so what fits here is what the
// matching draw puts on the screen: the same role, the same shaper, the same
// metric.
package elide

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"

	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Ellipsis is the one character a middle elision leaves in place of what it
// took out.
const Ellipsis = "…"

// Middle returns txt with as much of its middle replaced by [Ellipsis] as it
// takes for one line of it to fit within width, in the role style names at
// the weight [typeset.Text] shapes it in. Text that already fits is returned
// unchanged, and so is text at a width that holds nothing at all: a caller
// given no room is better served by the start of the line than by an
// ellipsis alone.
//
// What is kept is split evenly between the start and the end, the odd
// character going to the start, because the start of an address is what says
// which thing it is at all.
func Middle(gtx layout.Context, sh *text.Shaper, txt string, style tokens.TextStyle, width int) string {
	if txt == "" || width <= 0 {
		return txt
	}
	if typeset.LineWidth(gtx, sh, txt, style, font.Normal) <= width {
		return txt
	}
	r := []rune(txt)
	// Keep is monotone in the width it wants — a character added to either
	// end only widens the line — so the most that fits is found by halving
	// the range rather than by stepping down it.
	lo, hi := 0, len(r)-1
	best := ""
	for lo <= hi {
		mid := (lo + hi) / 2
		cut := keep(r, mid)
		if typeset.LineWidth(gtx, sh, cut, style, font.Normal) <= width {
			best, lo = cut, mid+1
		} else {
			hi = mid - 1
		}
	}
	if best == "" {
		// Not even the ellipsis fits. The line is drawn clipped either way,
		// and the start of it is the half that says something.
		return txt
	}
	return best
}

// keep returns txt's first and last n characters around one ellipsis, the odd
// one going to the start.
func keep(r []rune, n int) string {
	head := (n + 1) / 2
	tail := n - head
	return string(r[:head]) + Ellipsis + string(r[len(r)-tail:])
}
