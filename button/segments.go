package button

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/toolbarface"
)

// SegmentSeamClearDp is the room a segmented toolbar control leaves clear
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

// ChromeSegment is one segment of the platform's segmented toolbar control:
// the symbol it carries, the state it is drawn in, and the target that makes
// it pressable.
type ChromeSegment struct {
	// Icon draws the segment's symbol into the square it is handed. A nil
	// Icon draws an empty segment.
	Icon func(gtx layout.Context, sizePx int, col color.NRGBA)

	// State is this segment's own interaction state. Hovered and Pressed
	// tint this segment and not the control, which is what the platform
	// draws: the pointer is on one segment at a time. Disabled fades this
	// segment's symbol and takes its cursor away. Focused is read off the
	// whole control, since the ring the platform draws is the control's.
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
// capsule carrying several symbols, each in a segment of the control's own
// width, parted by the hairline the platform draws between them.
//
// MEASURED at 1x, Finder's back/forward pair: x 326–398 in
// finder-window-untinted-dark.png, where the control's rim reads at both ends
// and the seam at x=362 — two segments of 36 px with one column between
// them, at the toolbar control's own 36 px height. The light window draws the
// same pair at the same columns. A segment is therefore the width the chrome
// variant already draws around one symbol ([control.ChromeMarkDp] with
// [control.ChromeMarkSideDp] clear on each side), and the control is that
// width per segment plus the seams.
//
// The seam is a MEASURED value of the toolbar and not the Language's
// separator flattened: the two Finder captures read #f2f2f2 light over the
// control's #ffffff and #3a3a3a dark over its #262626, and the separator over
// those fills gives #e6e6e6 and #3b3b3b — twelve of 255 short in the light
// appearance and one over in the dark. See
// [tokens.PlatformColors.ToolbarControlSeam], which carries both readings.
//
// The shadow the control casts on its band is not drawn here — it falls
// outside the box this reports, the way [ChromeFace]'s does. Callers wrap
// this in [ChromeShadow].
func ChromeSegments(gtx layout.Context, p tokens.PlatformColors, d tokens.Density, segs []ChromeSegment) layout.Dimensions {
	if len(segs) == 0 {
		return layout.Dimensions{}
	}
	h := min(gtx.Dp(unit.Dp(d.ToolbarControlHeight)), gtx.Constraints.Max.Y)
	mark := gtx.Dp(control.ChromeMarkDp)
	segW := mark + 2*gtx.Dp(control.ChromeMarkSideDp)
	rule := max(gtx.Dp(segmentSeamDp), 1)
	total := len(segs)*segW + (len(segs)-1)*rule
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
	rest := toolbarface.Fill(p, tokens.StateNormal)
	outer := toolbarface.Capsule(gtx, box, h/2, p, rest, toolbarface.State{
		Focused: focused,
		// The band the pair stands on. Every segment of one control stands
		// on one band, so the first segment's answer is the control's.
		StandsOn: segs[0].State.Surface,
	})

	clear := min(gtx.Dp(SegmentSeamClearDp), h/2)
	for i, s := range segs {
		x := i * (segW + rule)
		seg := image.Rect(x, 0, x+segW, h)
		fill := rest
		if !s.State.Disabled {
			if st := chromeState(s.State); st == tokens.StateHover || st == tokens.StatePressed {
				fill = toolbarface.Fill(p, st)
				// Clipped to the capsule, so a tint on an end segment
				// follows the corner rather than squaring it off.
				area := outer.Push(gtx.Ops)
				paint.FillShape(gtx.Ops, fill, clip.Rect(seg).Op())
				area.Pop()
			}
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
		if s.Icon != nil && mark > 0 {
			fg := toolbarface.Mark(p, fill)
			if s.State.Disabled {
				fg = vgcolor.Flatten(p.DisabledControlText, fill)
			}
			area := outer.Push(gtx.Ops)
			off := op.Offset(image.Pt(x+(segW-mark)/2, (h-mark)/2)).Push(gtx.Ops)
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
