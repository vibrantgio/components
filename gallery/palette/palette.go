// Package palette draws the platform's colour set: every name AppKit answers
// for, in the order the set carries them, with the value each name has in
// both appearances.
//
// There is no derivation to show. The set is a recording — read off the
// platform under the aqua and darkAqua appearances, with the fills the
// platform paints but gives no name to measured off stored captures — so the
// only thing a page about it can say is what each name is, what it is worth
// in each appearance, and where it carries a coverage rather than a colour.
// That last is the reason a swatch alone will not do: a label, a seam, an
// overlay and the focus ring are black or white at a coverage, and a page
// that showed them as opaque chips would be showing colours the platform
// never paints.
//
// Every function is a pure function of the [tokens.PlatformColors] handed in,
// plus the chrome colours and type roles a caller states in [Chrome] and
// [Type]; nothing reads a default set, so the same code draws either
// appearance and a test can capture it without a window.
package palette

import (
	"image"
	stdcolor "image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/gallery/inventory"
	"github.com/vibrantgio/textdraw"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// Chrome is the story's own frame: the four colours it draws with that are
// not the set it is describing. Stated by the caller rather than derived
// here, since a heading band belongs to the page the story sits in.
type Chrome struct {
	// Surface is the fill a section's heading band carries.
	Surface stdcolor.NRGBA
	// Seam is the rule under a heading band and between two rows.
	Seam stdcolor.NRGBA
	// Text is the reading foreground: section labels, their captions, and
	// every name and value on the board.
	Text stdcolor.NRGBA
	// Muted is the least pronounced of the four: the column heads over the
	// board.
	Muted stdcolor.NRGBA
}

// Type is the story's view of the theme's typography: the roles it draws
// directly, as textdraw styles, plus the shaper it draws them with. The
// shaper is a field rather than read off a [tokens.Typography] here, since
// callers may need either the theme's cached shaper or a deterministic one
// for reproducible captures.
type Type struct {
	Shaper *text.Shaper
	Head   textdraw.TextStyle // column heads over the board
	Label  textdraw.TextStyle // section labels
	Body   textdraw.TextStyle // a row's name
	Small  textdraw.TextStyle // values and captions
}

// The frame measurements the story shares with the pages it stands in.
const (
	// captionGap is what a section's caption stands off the title beside it.
	captionGap unit.Dp = 14
	// hairline is a resting outline: the rule under a heading band, the frame
	// round a swatch.
	hairline unit.Dp = 1
	// ellipsis is the mark a run of text wears when it was cut short. One
	// mark, so a reader meets one sign for one fact wherever a line stopped
	// early.
	ellipsis = "…"
)

// The section's dimensions.
//
// The heading rows are the column's own frame at the column's own sizes:
// this story stands among a page of labelled sections, and a heading a few
// points off the ones around it reads as a heading from somewhere else.
// Everything inside them is measured to what it holds.
const (
	// SectionHeadH is a section heading, at the height a page of labelled
	// sections gives its own.
	SectionHeadH unit.Dp = 32

	// SetNameW is the column the platform names stand in, SetValueW the
	// column a value and its coverage stand in, and SetSwatchW the square
	// between them. SetRowH is the pitch from one row to the next.
	SetNameW   unit.Dp = 240
	SetValueW  unit.Dp = 132
	SetSwatchW unit.Dp = 26
	SetRowH    unit.Dp = 22
	// SetColGap is the air between two columns of the board.
	SetColGap unit.Dp = 14
)

// The section's label and caption.
const (
	SetLabel = "The platform's colour set"
	SetHint  = "every name the platform answers for · in the order the set carries them · a coverage is written after the value it is a coverage of"
	// HintSep joins one caption clause to the next and is the seam [FitHint]
	// truncates at.
	HintSep = " · "
	// LightHead and DarkHead name the two appearance columns.
	LightHead = "Light"
	DarkHead  = "Dark"
	// NameHead names the column the platform names stand in.
	NameHead = "Platform name"
)

// Rows is the story as rows of a page's column: the set under its own
// heading.
func Rows(p Chrome, c tokens.PlatformColors, ty Type) []layout.Widget {
	return []layout.Widget{
		Heading(p, c, ty, SetLabel, SetHint),
		Body(c, setBoard(p, c, ty)),
	}
}

// Heading labels one section, with what it is at the leading edge and how to
// read it at the trailing one — the two-part label a page of labelled sections
// carries, on the surface that page's headings stand on.
func Heading(p Chrome, c tokens.PlatformColors, ty Type, title, hint string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(SectionHeadH))
		paint.FillShape(gtx.Ops, p.Surface, clip.Rect{Max: size}.Op())
		line := gtx.Dp(hairline)
		paint.FillShape(gtx.Ops, p.Seam,
			clip.Rect(image.Rect(0, size.Y-line, size.X, size.Y)).Op())
		pad := gtx.Dp(inventory.SectionPadX)
		box := image.Rect(pad, 0, max(pad, size.X-pad), size.Y)
		textdraw.FillText(gtx, ty.Shaper, ty.Label, box, 0, 0.5, p.Text, title)
		// The caption takes what the title leaves rather than the whole bar, so
		// a narrow window truncates it instead of running it into the title.
		if lead := box.Min.X + natural(gtx, ty.Shaper, ty.Label, title) + gtx.Dp(captionGap); lead < box.Max.X {
			if fit := FitHint(gtx, ty, hint, box.Max.X-lead); fit != "" {
				textdraw.FillText(gtx, ty.Shaper, ty.Small,
					image.Rect(lead, 0, box.Max.X, size.Y), 1, 0.5, p.Text, fit)
			}
		}
		return layout.Dimensions{Size: size}
	}
}

// FitHint is a section's caption cut to the room it has: whole clauses
// dropped from the tail (never a mid-word cut), and nothing marks the cut —
// clauses are written tail-droppable on purpose, in an order that keeps the
// load-bearing ones first. With room for not even the leading clause, the
// caption is dropped whole rather than shown as a fragment.
func FitHint(gtx layout.Context, ty Type, hint string, room int) string {
	if natural(gtx, ty.Shaper, ty.Small, hint) <= room {
		return hint
	}
	clauses := strings.Split(hint, HintSep)
	heads := make([]string, 0, len(clauses))
	for n := len(clauses) - 1; n > 0; n-- {
		heads = append(heads, strings.Join(clauses[:n], HintSep))
	}
	return longestHead(gtx, ty.Shaper, ty.Small, heads, "", room)
}

// FitLine is one line of the board cut to the room its column has, never at a
// mid-word boundary. It tries a clause cut first (at the comma inside a line,
// the slash between two names, or [HintSep]), which leaves a shorter true
// sentence rather than an interrupted one. Failing that it falls back to a
// word boundary with a trailing ellipsis, since a mid-sentence cut must say
// that it stopped. With room for not even the first word, the line is handed
// to the shaper whole and wears whatever truncation it gives: a dropped line
// would leave a cell with a colour and no name at all.
func FitLine(gtx layout.Context, shaper *text.Shaper, style textdraw.TextStyle, line string, room int) string {
	if room <= 0 || natural(gtx, shaper, style, line) <= room {
		return line
	}
	if cut := longestHead(gtx, shaper, style, LineHeads(line, true), "", room); cut != "" {
		return cut
	}
	if cut := longestHead(gtx, shaper, style, LineHeads(line, false), ellipsis, room); cut != "" {
		return cut
	}
	return line
}

// LineHeads is every head this line can be cut down to, longest first: the
// head at each of its clause boundaries when clauses is set, and the head at
// each of its word boundaries when it is not. A boundary is a space; a clause
// boundary is a space preceded by the line's own punctuation, which is
// trimmed off the head so it never ends in a dangling separator.
func LineHeads(line string, clauses bool) []string {
	var heads []string
	for i := len(line) - 1; i > 0; i-- {
		if line[i] != ' ' {
			continue
		}
		if clauses && !strings.HasSuffix(line[:i], ",") &&
			!strings.HasSuffix(line[:i], " ·") && !strings.HasSuffix(line[:i], " /") {
			continue
		}
		if head := strings.TrimRight(line[:i], " ,·/"); head != "" {
			heads = append(heads, head)
		}
	}
	return heads
}

// longestHead is the first of these heads that fits the room with the tail on
// the end of it, and "" when none of them does. The heads arrive longest first,
// so the first that fits is the most of the line that could be kept.
func longestHead(gtx layout.Context, shaper *text.Shaper, style textdraw.TextStyle, heads []string, tail string, room int) string {
	for _, head := range heads {
		if natural(gtx, shaper, style, head+tail) <= room {
			return head + tail
		}
	}
	return ""
}

// Body lays one section's content out on the content plane, inside the margin
// every other body in that column keeps.
//
// The content is drawn before the fill is painted and replayed over it: the
// height is the content's own, so the fill cannot be painted until the
// content is measured.
func Body(c tokens.PlatformColors, body func(gtx layout.Context, width int) int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		padX, padY := gtx.Dp(inventory.SectionPadX), gtx.Dp(inventory.SectionPadY)
		width := max(0, gtx.Constraints.Max.X-2*padX)
		macro := op.Record(gtx.Ops)
		h := 0
		at(gtx, image.Pt(padX, padY), func(gtx layout.Context) { h = body(gtx, width) })
		content := macro.Stop()
		size := image.Pt(gtx.Constraints.Max.X, h+2*padY)
		paint.FillShape(gtx.Ops, inventory.SectionSurface(c), clip.Rect{Max: size}.Op())
		content.Add(gtx.Ops)
		return layout.Dimensions{Size: size}
	}
}

// setBoard draws the set as a table: the column heads once, then a row per
// field — its platform name at the leading edge, and its light and dark
// values beside it, each as a swatch and the value written out.
//
// A coverage is composited onto ITS OWN appearance's content plane, not onto
// the surface the board happens to stand on: white at 0.85 is what a dark
// label is, and over the light page it would read as very nearly nothing. So
// the two columns carry the same two swatches in either appearance, and the
// value beside each is the recorded one, coverage and all — which is the only
// place a coverage and an opaque colour can be told apart.
func setBoard(p Chrome, c tokens.PlatformColors, ty Type) func(gtx layout.Context, width int) int {
	rows := inventory.PlatformRows()
	return func(gtx layout.Context, width int) int {
		rowH := gtx.Dp(SetRowH)
		gap := gtx.Dp(SetColGap)
		nameW, valueW, swatchW := gtx.Dp(SetNameW), gtx.Dp(SetValueW), gtx.Dp(SetSwatchW)
		line := gtx.Dp(hairline)

		// The column origins, once: the name, then each appearance's swatch
		// and the value beside it.
		nameX := 0
		lightSwatchX := nameX + nameW + gap
		lightValueX := lightSwatchX + swatchW + gap
		darkSwatchX := lightValueX + valueW + gap
		darkValueX := darkSwatchX + swatchW + gap
		end := min(width, darkValueX+valueW)

		write := func(x, y, room int, style textdraw.TextStyle, col stdcolor.NRGBA, s string) {
			box := image.Rect(x, y, x+room, y+rowH)
			textdraw.FillText(gtx, ty.Shaper, style, box, 0, 0.5,
				col, FitLine(gtx, ty.Shaper, style, s, room))
		}
		swatch := func(x, y int, v, plane stdcolor.NRGBA) {
			box := image.Rect(x, y+2, x+swatchW, y+rowH-2)
			paint.FillShape(gtx.Ops, vgcolor.Flatten(v, plane), clip.Rect(box).Op())
			strokeRect(gtx, box, line, p.Seam)
		}

		y := 0
		write(nameX, y, nameW, ty.Head, p.Muted, NameHead)
		write(lightValueX, y, valueW, ty.Head, p.Muted, LightHead)
		write(darkValueX, y, valueW, ty.Head, p.Muted, DarkHead)
		y += rowH
		paint.FillShape(gtx.Ops, p.Seam, clip.Rect(image.Rect(0, y, end, y+line)).Op())
		y += line

		for _, r := range rows {
			write(nameX, y, nameW, ty.Body, p.Text, r.Name)
			swatch(lightSwatchX, y, r.Light, tokens.PlatformLight.ControlBackground)
			write(lightValueX, y, valueW, ty.Small, p.Text, inventory.Hex(r.Light))
			swatch(darkSwatchX, y, r.Dark, tokens.PlatformDark.ControlBackground)
			write(darkValueX, y, valueW, ty.Small, p.Text, inventory.Hex(r.Dark))
			y += rowH
		}
		return y
	}
}

// typeSection is the inventory section a page borrows to close the story: the
// whole type stack, every role a surface reads in.
const typeSection = "foundations-type"

// sectionTitleSep is the seam an inventory section's title is written with:
// what the section is, then how to read it. The story's own bands are built
// from exactly that pair — a label at the leading edge and a caption at the
// trailing one — so a borrowed title splits into a band with nothing reworded.
const sectionTitleSep = " — "

// TypeScaleRows is the inventory's type stack as two rows in the story's own
// bands: the heading band the story's own sections wear, over the inventory
// section's own body. The inventory's own title words are kept, split at the
// em dash its titles are already written with, so nothing is reworded here. A
// title with no separator lands as the whole label with no caption.
func TypeScaleRows(inv *inventory.Inventory, p Chrome, c tokens.PlatformColors, ty Type) []layout.Widget {
	for _, s := range inv.Foundations(c) {
		if s.Name != typeSection {
			continue
		}
		label, hint, _ := strings.Cut(s.Title, sectionTitleSep)
		return []layout.Widget{
			Heading(p, c, ty, label, hint),
			Body(c, scaleBody(s)),
		}
	}
	return nil
}

// scaleBody adapts an inventory section's body to the story body's shape: the
// story measures its content and reports the height, while a section body is
// laid out in a slot of the height the section states — bounded, since the
// type stack measures nothing of its own.
func scaleBody(s inventory.Section) func(gtx layout.Context, width int) int {
	return func(gtx layout.Context, width int) int {
		h := gtx.Dp(s.Height)
		gtx.Constraints = layout.Constraints{Max: image.Pt(width, h)}
		s.Body(gtx)
		return h
	}
}

// natural is how wide a string wants to be, unconstrained by the room it is
// about to be given.
func natural(gtx layout.Context, shaper *text.Shaper, style textdraw.TextStyle, str string) int {
	gtx.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	return textdraw.MeasureText(gtx, shaper, style, str).X
}

// strokeRect outlines a rectangle inside its own bounds, which is what a flat
// swatch needs to be visible against a surface of nearly its own colour.
func strokeRect(gtx layout.Context, r image.Rectangle, width int, c stdcolor.NRGBA) {
	if width <= 0 {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.Rect(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+width)).Op())
	paint.FillShape(gtx.Ops, c, clip.Rect(image.Rect(r.Min.X, r.Max.Y-width, r.Max.X, r.Max.Y)).Op())
	paint.FillShape(gtx.Ops, c, clip.Rect(image.Rect(r.Min.X, r.Min.Y, r.Min.X+width, r.Max.Y)).Op())
	paint.FillShape(gtx.Ops, c, clip.Rect(image.Rect(r.Max.X-width, r.Min.Y, r.Max.X, r.Max.Y)).Op())
}

// at offsets the operations w records to origin, leaving the caller's
// coordinate system untouched.
func at(gtx layout.Context, origin image.Point, w func(gtx layout.Context)) {
	defer op.Offset(origin).Push(gtx.Ops).Pop()
	w(gtx)
}
