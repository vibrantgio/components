package input

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"golang.org/x/image/math/fixed"

	"github.com/vibrantgio/theme/tokens"
)

// insertion is what a live field keeps for its insertion point from frame to
// frame: whether the field held the keyboard on the last frame, whether a
// press reached the field on the last frame, and the clock its blink runs
// on.
//
// Every live field draws its insertion point through this one type, bordered
// or borderless, so the line has one width, one height rule and one colour in
// every field. gioui.org/widget's editor paints its own caret from the text's
// material, two columns wide straddling the caret's x and cut to one column
// at the editor's leading edge, so a field lays its editor out with that
// caret suppressed and draws this one instead.
type insertion struct {
	// press is the field's pass-through pointer target: every press on the
	// field reaches it as well as the editor.
	press   *int
	pressed bool
	focused bool
	sel     [2]int
	since   time.Time
}

func newInsertion() *insertion { return &insertion{press: new(int)} }

// The blink: half a period shown and half hidden, and steady after a while
// without an edit or a move, as gioui.org/widget's own caret blinks.
const (
	blinkPeriod = time.Second
	blinkFor    = 10 * time.Second
)

// begin decides the selection a field takes when it gains the keyboard. A
// field that takes the keyboard by a key or a command, and not by a press
// on it, has its whole value selected with the insertion point at its end,
// so the text typed next replaces the value. A press puts the insertion
// point where the editor's own press put it.
//
// begin runs before the editor's events are drained, so the keys of the
// frame that first sees the field hold the keyboard act on the selected
// value. focused is whether the field holds the keyboard at that point.
// The keyboard a command or a press moves is seen here on the frame after
// it moved, so a press is remembered from the frame before: see [presses].
func (in *insertion) begin(gtx layout.Context, editor *widget.Editor, focused bool) {
	if focused && !in.focused {
		if !in.pressed {
			editor.SetCaret(editor.Len(), 0)
		}
		in.since = gtx.Now
	}
	in.pressed = false
	in.focused = focused
}

// presses reads the presses on the field's own target. It runs after the
// editor's events are drained, so a press that gave the field the keyboard
// on this frame is the one [begin] reads on the next.
func (in *insertion) presses(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: in.press, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Press {
			in.pressed = true
		}
	}
}

// target registers the field's pass-through press target over size. It is
// registered after the editor, so it lies over the editor's own area and
// lets every press through to it.
func (in *insertion) target(gtx layout.Context, size image.Point) {
	cl := clip.Rect(image.Rectangle{Max: size}).Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, in.press)
	pass.Pop()
	cl.Pop()
}

// shown reports whether the insertion point is drawn on this frame, and asks
// for the frame at which the blink next turns. focused is whether the field
// holds the keyboard as it is drawn. A move of the insertion point and an
// edit start the blink again.
func (in *insertion) shown(gtx layout.Context, editor *widget.Editor, focused bool) bool {
	start, end := editor.Selection()
	if sel := [2]int{start, end}; sel != in.sel {
		in.sel = sel
		in.since = gtx.Now
	}
	if !focused || !gtx.Enabled() || start != end {
		return false
	}
	dt := gtx.Now.Sub(in.since)
	if dt < 0 || dt >= blinkFor {
		return true
	}
	half := blinkPeriod / 2
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(half - dt%half)})
	return dt%blinkPeriod < half
}

// layoutEditor lays editor out with its own caret suppressed: the editor
// paints that caret only while it is editable, so it is laid out read-only
// for this one call. Its events are already drained by the field's own
// update loop, so no edit reaches the read-only call.
func layoutEditor(gtx layout.Context, editor *widget.Editor, lay func(gtx layout.Context) layout.Dimensions) layout.Dimensions {
	ro := editor.ReadOnly
	editor.ReadOnly = true
	dims := lay(gtx)
	editor.ReadOnly = ro
	return dims
}

// insertionWidth is the insertion point's width: one column at 1x, a point
// at any density.
const insertionWidth = unit.Dp(1)

// paint draws the insertion point in editor's own coordinates while focused
// is true, at the caret: see [drawInsertion]. ascent is how far the
// editor's baseline stands below the top of its line box. width is the
// editor's width.
func (in *insertion) paint(gtx layout.Context, editor *widget.Editor, focused bool, p tokens.PlatformColors, ascent, lineH, width int) {
	if !in.shown(gtx, editor, focused) {
		return
	}
	at := editor.CaretCoords()
	x := int(math.Floor(float64(at.X) + 0.5))
	top := int(math.Floor(float64(at.Y)+0.5)) - ascent
	drawInsertion(gtx, p, x, top, lineH, width)
}

// drawInsertion draws the insertion point that the live and the static
// field paths share: one column of [insertionWidth] starting at x, from top
// down lineH, in [insertionColor]. width is the width of the box the text
// is drawn in, and the column stays inside it.
func drawInsertion(gtx layout.Context, p tokens.PlatformColors, x, top, lineH, width int) {
	w := max(gtx.Dp(insertionWidth), 1)
	x = min(max(x, 0), max(width-w, 0))
	area := clip.Rect(image.Rect(x, top, x+w, top+lineH)).Push(gtx.Ops)
	paint.ColorOp{Color: insertionColor(p)}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	area.Pop()
}

// staticInsertionAt is where a static field draws its insertion point: after
// the last character of value, shaped with the font, the label and the size
// the paint pass draws it with, inside maxW. x is the end of that character
// and dy is how far its line stands below the first line. An empty value
// puts it at x 0 on the first line, where the text begins. The end is
// rounded as the live path rounds the caret's x.
//
// MEASURED, voicememos-multi-folder-search-2026-09-18.png: the insertion
// point stands immediately after the typed query's last character.
func staticInsertionAt(gtx layout.Context, shaper *text.Shaper, f font.Font, wl widget.Label, size unit.Sp, value string, maxW int) (x, dy int) {
	if shaper == nil || value == "" {
		return 0, 0
	}
	shaper.LayoutString(text.Parameters{
		Font:            f,
		PxPerEm:         fixed.I(gtx.Sp(size)),
		MaxLines:        wl.MaxLines,
		Truncator:       wl.Truncator,
		Alignment:       wl.Alignment,
		WrapPolicy:      wl.WrapPolicy,
		MaxWidth:        maxW,
		Locale:          gtx.Locale,
		LineHeight:      fixed.I(gtx.Sp(wl.LineHeight)),
		LineHeightScale: wl.LineHeightScale,
	}, value)
	var first, last text.Glyph
	n := 0
	for {
		g, ok := shaper.NextGlyph()
		if !ok {
			break
		}
		if n == 0 {
			first = g
		}
		last = g
		n++
	}
	if n == 0 {
		return 0, 0
	}
	return (last.X + last.Advance).Round(), int(last.Y - first.Y)
}

// insertionColor is the colour the insertion point is drawn in: the
// platform's text insertion point colour, which follows the accent.
func insertionColor(p tokens.PlatformColors) color.NRGBA {
	c := p.TextInsertionPoint
	c.A = 0xff
	return c
}
