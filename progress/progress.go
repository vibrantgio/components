package progress

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
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
)

// The indicator's measures. MEASURED in the reference's "What the
// indeterminate indicator measures", off System Settings checking for
// updates at 1x, where one pixel is one point.
const (
	// Size is the indicator's extent, square.
	Size unit.Dp = 16
	// Strokes is the number of strokes, one every 45°.
	Strokes = 8
	// Step is how long the darkest stroke stays at one place before it
	// moves one stroke clockwise. DERIVED: two steps in 235 ms and two more
	// in 224 ms.
	Step = 115 * time.Millisecond
)

// Gap is the space between the indicator and the text under it in the
// region shape. DERIVED in the reference's "What the indeterminate indicator
// measures": no captured platform window centres the two as one group, so the
// gap is half the indicator's measured extent, the wheel's radius.
const Gap unit.Dp = Size / 2

// The strokes' geometry in the 16 px box at 1x: each stroke runs from 3 px to
// 8 px off the centre and is 2 px wide. MEASURED with Size.
const (
	strokeInner = 3.0
	strokeOuter = 8.0
	strokeWidth = 2.0
)

// The darkest stroke is the label colour's own hue at this coverage, per
// scheme. MEASURED: the light darkest stroke reads #424242 over #f7f7f7,
// black at about 187/255; the dark lightest stroke reads #9e9e9e over
// #252525, white at about 140/255. The lightest stroke is QuaternaryLabel,
// within 5 levels in both schemes. The strokes between the two step evenly
// in colour; that spacing is DERIVED, not read.
const (
	headAlphaLight = 187
	headAlphaDark  = 140
)

// Props states one indicator and the text that may stand with it.
type Props struct {
	// Platform is the platform's colours: the strokes are the label hue and
	// QuaternaryLabel, and the text is Label.
	Platform tokens.PlatformColors
	// Fill is the fill the indicator and the text stand on. The strokes and
	// the text are flattened over it, so the pixels are the platform's
	// pixels on that fill.
	Fill color.NRGBA
	// Text is the application's own phrase naming the task, ending with an
	// ellipsis. [Indicator] ignores it; [Region] and [Row] draw it. An empty
	// Text draws the indicator alone.
	Text string
	// Shaper shapes Text. [Region] and [Row] need it when Text is set.
	Shaper *text.Shaper
	// Typography is the type set Text is drawn from: the System font role.
	// The zero value is tokens.DefaultTypography.
	Typography tokens.Typography
	// Phase is the stroke drawn darkest, counted clockwise from the top,
	// when the frame has no clock or Still is set. It exists so that a test
	// and a stored image can fix the frame.
	Phase int
	// Still stops the wheel at Phase even when the frame has a clock. It
	// exists for a screen that lays out a live frame and must not change
	// between two captures.
	Still bool
}

// TextStyle is the style the text beside the indicator is set in: the Body
// Large role. MEASURED: the platform sets "Checking for updates…" at 13 pt
// in the system font's regular weight. The reference's body row, which Body
// Large takes, has that size and weight. The line box is the text's own, so
// the group's height centres the glyphs and not a role's leading.
func TextStyle(typ tokens.Typography) tokens.TextStyle {
	if typ.SystemFont.Size == 0 {
		typ = tokens.DefaultTypography
	}
	s := typ.SystemFont
	s.LineHeight = 0
	return s
}

// phase answers the stroke drawn darkest in this frame. With a clock it is
// the step the clock is at, and the frame at the next step is requested.
func (p Props) phase(gtx layout.Context) int {
	if p.Still || gtx.Now.IsZero() {
		return ((p.Phase % Strokes) + Strokes) % Strokes
	}
	n := gtx.Now.UnixNano() / int64(Step)
	gtx.Execute(op.InvalidateCmd{At: time.Unix(0, (n+1)*int64(Step))})
	return int(n % Strokes)
}

// dark answers whether the colours are a dark scheme's: the label is light.
func dark(c tokens.PlatformColors) bool {
	return int(c.Label.R)+int(c.Label.G)+int(c.Label.B) > 3*127
}

// StrokeColor is the colour of the stroke k steps behind the darkest, for k
// from 0 (the darkest) to Strokes-1 (the lightest), flattened over fill.
func StrokeColor(c tokens.PlatformColors, fill color.NRGBA, k int) color.NRGBA {
	head := c.Label
	head.A = headAlphaLight
	if dark(c) {
		head.A = headAlphaDark
	}
	tail := c.QuaternaryLabel
	t := float64(k) / float64(Strokes-1)
	mix := func(a, b uint8) uint8 {
		return uint8(math.Round(float64(a) + t*(float64(b)-float64(a))))
	}
	col := color.NRGBA{R: mix(head.R, tail.R), G: mix(head.G, tail.G), B: mix(head.B, tail.B), A: mix(head.A, tail.A)}
	return vgcolor.Flatten(col, fill)
}

// Indicator draws the indeterminate indicator at its measured size, at the
// origin, and reports that size.
func Indicator(gtx layout.Context, p Props) layout.Dimensions {
	px := gtx.Dp(Size)
	s := float32(px) / float32(Size)
	head := p.phase(gtx)
	ctr := f32.Pt(float32(px)/2, float32(px)/2)
	for i := range Strokes {
		th := float64(i) * 2 * math.Pi / Strokes
		d := f32.Pt(float32(math.Sin(th)), float32(-math.Cos(th)))
		n := f32.Pt(-d.Y, d.X)
		in, out, hw := strokeInner*s, strokeOuter*s, strokeWidth*s/2
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(ctr.Add(d.Mul(in)).Add(n.Mul(hw)))
		path.LineTo(ctr.Add(d.Mul(out)).Add(n.Mul(hw)))
		path.LineTo(ctr.Add(d.Mul(out)).Sub(n.Mul(hw)))
		path.LineTo(ctr.Add(d.Mul(in)).Sub(n.Mul(hw)))
		path.Close()
		behind := ((head-i)%Strokes + Strokes) % Strokes
		paint.FillShape(gtx.Ops, StrokeColor(p.Platform, p.Fill, behind), clip.Outline{Path: path.End()}.Op())
	}
	return layout.Dimensions{Size: image.Pt(px, px)}
}

// drawText draws the phrase in the label colour over the fill across the
// width gtx gives, aligned as align says, and reports its size.
func drawText(gtx layout.Context, p Props, align text.Alignment) layout.Dimensions {
	return typeset.TextAligned(gtx, p.Shaper, p.Text, TextStyle(p.Typography),
		vgcolor.Flatten(p.Platform.Label, p.Fill), 0, font.Normal, align)
}

// Region draws the region shape in the frame gtx's maximum gives: the
// indicator above the text, [Gap] between them, centred across and down in
// the frame as one group. It is the shape for a region whose content is
// still arriving, and it stands where that content will stand. The text
// wraps across the frame when it is wider. It reports the frame.
func Region(gtx layout.Context, p Props) layout.Dimensions {
	frame := gtx.Constraints.Max
	if frame.X <= 0 || frame.Y <= 0 {
		return layout.Dimensions{Size: frame}
	}
	size := gtx.Dp(Size)
	height := size
	var call op.CallOp
	if p.Text != "" && p.Shaper != nil {
		tgtx := gtx
		tgtx.Constraints = layout.Constraints{Min: image.Pt(frame.X, 0), Max: frame}
		rec := op.Record(gtx.Ops)
		dims := drawText(tgtx, p, text.Middle)
		call = rec.Stop()
		height += gtx.Dp(Gap) + dims.Size.Y
	}
	top := max(frame.Y-height, 0) / 2
	off := op.Offset(image.Pt((frame.X-size)/2, top)).Push(gtx.Ops)
	Indicator(gtx, p)
	off.Pop()
	if p.Text != "" && p.Shaper != nil {
		off = op.Offset(image.Pt(0, top+size+gtx.Dp(Gap))).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}
	return layout.Dimensions{Size: frame}
}

// Row draws the row shape across the width gtx's maximum gives: the text at
// the leading end and the indicator at the trailing end, both centred on the
// row's middle, as System Settings draws "Checking for updates…". The row is
// as tall as its minimum or as the taller of the two. The text is kept
// [Gap] short of the indicator and wraps when it is wider. Row draws no fill
// and no padding: the row it stands in owns both. It reports the row's size.
func Row(gtx layout.Context, p Props) layout.Dimensions {
	width := gtx.Constraints.Max.X
	size := gtx.Dp(Size)
	height := max(gtx.Constraints.Min.Y, size)
	var call op.CallOp
	var tdims layout.Dimensions
	if p.Text != "" && p.Shaper != nil {
		tgtx := gtx
		tgtx.Constraints = layout.Constraints{Max: image.Pt(max(width-size-gtx.Dp(Gap), 0), gtx.Constraints.Max.Y)}
		rec := op.Record(gtx.Ops)
		tdims = drawText(tgtx, p, text.Start)
		call = rec.Stop()
		height = max(height, tdims.Size.Y)
		off := op.Offset(image.Pt(0, (height-tdims.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}
	off := op.Offset(image.Pt(width-size, (height-size)/2)).Push(gtx.Ops)
	Indicator(gtx, p)
	off.Pop()
	return layout.Dimensions{Size: image.Pt(width, height)}
}
