package button

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/toolbarface"
)

// SegmentSeamClearDp is the space a segmented toolbar control leaves clear
// above and below the hairline parting two of its segments.
//
// MEASURED at 1x, the back/forward pair in finder-window-light.png and
// finder-window-untinted-dark.png: the control runs y 8–43, 36 px tall, and
// its seam runs y 16–35 in both appearances — twenty rows, eight clear at
// each end. The pixel itself is one column wide, #f2f2f2 over the light
// control's #ffffff and #3a3a3a over the dark one's #262626.
const SegmentSeamClearDp unit.Dp = 8

// segmentSeamDp is the width of that hairline: one column in both stored
// Finder captures, which is the width every seam in this library is drawn at.
const segmentSeamDp unit.Dp = 1

// segmentLabelRoom is the width a segment's word is measured in: room enough
// that no single-line label this library draws is broken or elided by it, so
// the measured width is the label's own.
const segmentLabelRoom = 1 << 20

// ChromeSegment is one segment of the platform's segmented toolbar control:
// what it carries, the state it is drawn in, and the target that makes it
// pressable.
type ChromeSegment struct {
	// Icon draws the segment's symbol into the square it is handed. A nil
	// Icon draws an empty segment. It is not reached while Label carries a
	// word: a segment carries one or the other.
	Icon func(gtx layout.Context, sizePx int, col color.NRGBA)

	// Label is the word this segment carries instead of a symbol.
	//
	// NOT MEASURED, and the one thing here that is not: not one control in
	// the five stored toolbar bands carries a word (controls.md, "What a
	// toolbar control's symbol measures"), so no capture says what a worded
	// segment insets its word by. A worded segment is drawn at its label
	// plus the clearance the symbol path spends at either side
	// ([control.ChromeMarkSideDp]), which is the nearest measured number
	// this control holds.
	Label string

	// State is this segment's own interaction state. Hovered and Pressed
	// tint this segment and not the control, which is what the platform
	// draws: the pointer is on one segment at a time. Disabled fades this
	// segment's symbol and takes its cursor away. Checked draws this segment
	// as the chosen one. Focused is read off the whole control, since the
	// ring the platform draws is the control's.
	State RenderState

	// Target, if non-nil, is laid out over this segment's own box at the
	// segment's exact size — where the caller's widget.Clickable, the
	// segment's semantics and its cursor go. It is laid out after the face
	// is painted, so it takes the pointer; nothing it draws is expected to
	// show. The cursor belongs to it and not here: a cursor is set for the
	// clip area in force, and the area that ends at the segment's own edges
	// is the caller's clickable.
	Target layout.Widget
}

// ChromeSegments draws the platform's segmented bordered toolbar control: one
// capsule carrying several symbols or words, each in a segment of the
// control's own width, parted by the hairline the platform draws between
// them, with the chosen segment wearing the patch the platform fills it with.
//
// MEASURED at 1x, Finder's back/forward pair: x 326–398 in
// finder-window-untinted-dark.png, where the control's rim reads at both ends
// and the seam at x=362 — two segments of 36 px with one column between
// them, at the toolbar control's own 36 px height. The light window draws the
// same pair at the same columns. A symbol segment is therefore the width the
// chrome variant already draws around one symbol ([control.ChromeMarkDp] with
// [control.ChromeMarkSideDp] clear on each side), and the control is that
// width per segment plus the seams.
//
// EVERY SEGMENT OF ONE CONTROL IS THE SAME WIDTH. MEASURED,
// finder-window-untinted-dark.png: the four-segment view control spans x
// 796–943, 148 px over four segments, 37 each; Mail's three-segment groups
// divide to the same 37.3. So the widest segment sets the width of all of
// them, and a caller asking for a wider control through gtx.Constraints.Min.X
// — a segmented control laid across a form's own column — gets that width
// divided evenly rather than a ragged row.
//
// The seam is a MEASURED value of the toolbar and not the Language's
// separator flattened: the two Finder captures read #f2f2f2 light over the
// control's #ffffff and #3a3a3a dark over its #262626, and the separator over
// those fills gives #e6e6e6 and #3b3b3b — twelve of 255 short in the light
// appearance and one over in the dark. See
// [tokens.PlatformColors.ToolbarControlSeam], which carries both readings.
//
// THE PLACE SETTLES THE HEIGHT. A control standing in a chrome band is the
// toolbar control's own 36; one standing in a sheet's or a pane's body is the
// dialog control's 24, which is what the first segment's
// [RenderState.Place] says. See [Place].
//
// The shadow the control casts on its band is not drawn here — it falls
// outside the box this reports, the way [ChromeFace]'s does. Callers in a
// band wrap this in [ChromeShadow]; a control in a body casts none.
func ChromeSegments(gtx layout.Context, shaper *text.Shaper, p tokens.PlatformColors, labelStyle tokens.TextStyle, d tokens.Density, segs []ChromeSegment) layout.Dimensions {
	if len(segs) == 0 {
		return layout.Dimensions{}
	}
	// The control's height is the one its PLACE names, and the place is the
	// control's rather than a segment's: every segment of one control stands
	// in one place, so the first segment's answer is the control's, the way
	// its focus and the band it stands on already are.
	h := min(gtx.Dp(unit.Dp(segs[0].State.Place.Height(d))), gtx.Constraints.Max.Y)
	mark := gtx.Dp(control.ChromeMarkDp)
	side := gtx.Dp(control.ChromeMarkSideDp)
	rule := max(gtx.Dp(segmentSeamDp), 1)

	// The fill each segment is drawn over, and the word it carries, both
	// worked out before the control's width is: the widest segment sets the
	// width of every segment, so nothing can be placed until all of them
	// have been measured.
	fills := make([]color.NRGBA, len(segs))
	labels := make([]op.CallOp, len(segs))
	labelSize := make([]image.Point, len(segs))
	segW := mark + 2*side
	place := segs[0].State.Place
	rest := placeRestingFill(p, place)
	for i, s := range segs {
		fills[i] = rest
		if !s.State.Disabled {
			if st := chromeState(s.State); st == tokens.StateHover || st == tokens.StatePressed {
				fills[i] = placeFill(p, place, st)
			}
		}
		if s.Label == "" {
			continue
		}
		fg := placeForeground(p, place, fills[i])
		if s.State.Disabled {
			fg = vgcolor.Flatten(p.DisabledControlText, fills[i])
		}
		labels[i], labelSize[i] = shapeSegmentLabel(gtx, shaper, labelStyle, s.Label, fg)
		if w := labelSize[i].X + 2*side; w > segW {
			segW = w
		}
	}

	total := len(segs)*segW + (len(segs)-1)*rule
	if total < gtx.Constraints.Min.X {
		total = gtx.Constraints.Min.X
	}
	if total > gtx.Constraints.Max.X {
		total = gtx.Constraints.Max.X
	}
	size := image.Pt(total, h)
	box := image.Rectangle{Max: size}

	// One capsule for the whole control, at the resting fill and wearing the
	// control's own rim: the segments are divisions of one box and not boxes
	// of their own, which is why the rim runs round the pair and not round
	// each half.
	focused := false
	for _, s := range segs {
		focused = focused || (s.State.Focused && !s.State.Disabled)
	}
	outer := toolbarface.Capsule(gtx, box, h/2, p, rest, toolbarface.State{
		Focused: focused,
		Form:    place == Body,
		// The band the pair stands on. Every segment of one control stands
		// on one band, so the first segment's answer is the control's.
		StandsOn: segs[0].State.Surface,
	})

	// The segments divide what the control ended up at, so a control asked
	// for a width its natural segments do not add up to spreads the
	// difference rather than leaving it at one end.
	inner := total - (len(segs)-1)*rule
	edgeAt := func(i int) int { return i*inner/len(segs) + i*rule }

	clear := min(gtx.Dp(SegmentSeamClearDp), h/2)
	for i, s := range segs {
		x, xEnd := edgeAt(i), edgeAt(i)+inner*(i+1)/len(segs)-inner*i/len(segs)
		seg := image.Rect(x, 0, xEnd, h)
		fill := fills[i]
		if fill != rest {
			// Clipped to the capsule, so a tint on an end segment
			// follows the corner rather than squaring it off.
			area := outer.Push(gtx.Ops)
			paint.FillShape(gtx.Ops, fill, clip.Rect(seg).Op())
			area.Pop()
		}
		if s.State.Checked {
			// The chosen segment's own patch, inset inside the SEGMENT and
			// cornered at half its height — the 32 by 26 measured inside
			// Finder's 37 by 36 view segment. Clipped to the capsule for the
			// same reason the tint is.
			area := outer.Push(gtx.Ops)
			toolbarface.CheckedPatch(gtx, seg, p, fill)
			area.Pop()
		}
		if i > 0 {
			// The seam is the CONTROL'S own line and a measured value, not
			// the Language's separator flattened: the platform's separator
			// over the fill lands #e6e6e6 light against the measured #f2f2f2,
			// twelve of 255 short. See
			// tokens.PlatformColors.ToolbarControlSeam.
			line := image.Rect(x-rule, clear, x, h-clear)
			paint.FillShape(gtx.Ops, p.ToolbarControlSeam, clip.Rect(line).Op())
		}
		switch {
		case s.Label != "":
			area := outer.Push(gtx.Ops)
			off := op.Offset(image.Pt(
				x+(seg.Dx()-labelSize[i].X)/2,
				(h-labelSize[i].Y)/2,
			)).Push(gtx.Ops)
			labels[i].Add(gtx.Ops)
			off.Pop()
			area.Pop()
		case s.Icon != nil && mark > 0:
			fg := placeForeground(p, place, fill)
			if s.State.Disabled {
				fg = vgcolor.Flatten(p.DisabledControlText, fill)
			}
			area := outer.Push(gtx.Ops)
			off := op.Offset(image.Pt(x+(seg.Dx()-mark)/2, (h-mark)/2)).Push(gtx.Ops)
			s.Icon(gtx, mark, fg)
			off.Pop()
			area.Pop()
		}
		if s.Target != nil {
			off := op.Offset(seg.Min).Push(gtx.Ops)
			tgtx := gtx
			tgtx.Constraints = layout.Exact(seg.Size())
			s.Target(tgtx)
			off.Pop()
		}
	}
	return layout.Dimensions{Size: size}
}

// shapeSegmentLabel measures a segment's word at its own width rather than
// at the width a caller left it, and returns the recorded drawing with
// the size it measured to. typeset.Layout, not widget.Label.Layout, because
// the role's line height has to be the height of the label box.
func shapeSegmentLabel(gtx layout.Context, shaper *text.Shaper, style tokens.TextStyle, label string, fg color.NRGBA) (op.CallOp, image.Point) {
	mColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: fg}.Add(gtx.Ops)
	material := mColor.Stop()

	lg := gtx
	lg.Constraints.Min = image.Point{}
	lg.Constraints.Max.X = segmentLabelRoom

	m := op.Record(gtx.Ops)
	dims := typeset.Layout(lg, shaper, typeset.Label(style, 1), typeset.Font(style, font.Normal), unit.Sp(style.Size), label, material)
	return m.Stop(), dims.Size
}
