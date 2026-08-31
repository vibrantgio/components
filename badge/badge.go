package badge

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Variant is the role a badge speaks in. The five differ in hue and in
// nothing else: same type, same box, same anatomy. There is no emphasis axis
// — see the package doc.
type Variant uint8

const (
	// Neutral is the plain category label, and the zero value: a badge that
	// names a kind rather than reporting a status. It is the only variant
	// with no pinned base behind it, so its ink is always a measured rung of
	// the neutral ramp.
	Neutral Variant = iota
	// Success, Warning, Error and Info are the four statuses, each in its own
	// role's hue.
	Success
	Warning
	Error
	Info
)

// role is the colour role a variant reads its ink off.
func (v Variant) role() tokens.Role {
	switch v {
	case Success:
		return tokens.RoleSuccess
	case Warning:
		return tokens.RoleWarning
	case Error:
		return tokens.RoleError
	case Info:
		return tokens.RoleInfo
	}
	return tokens.RoleNeutral
}

// CloseHitDp is the side of the pointer target the close mark registers, in
// dp, centred on the mark and free to overhang the badge.
//
// It is WCAG 2.5.8 Target Size (Minimum), the AA criterion, and not the 44 dp
// of [tokens.MinHitTarget]: 44 is this system's floor for a standalone control
// with space around it, and a 44 dp target centred on a 16 dp badge would
// reach into whatever is set beside it.
const CloseHitDp = 24

// closeStrokeDp is the width of each of the close mark's two strokes, in dp.
// It is scaled rather than rounded to whole pixels: an axis-aligned line lands
// on whole pixels and reads at exactly its weight, while an x is two diagonals
// and is anti-aliased at any width. Measured on the label-small specimen at
// 1x, 1.25 dp lands the Medium face's stems between a whole pixel's over- and
// under-weight.
const closeStrokeDp = 1.25

// Glyph is the painter a badge draws its sign with: it fills a sizePx×sizePx
// box at the current origin in colour col. It is the same signature
// components/chip's mark, components/button's icon-only face and
// components/icon's registry all use, so a named glyph, a clip.Path drawn by
// hand and a verdict mark built for one screen are interchangeable here.
//
// A nil Glyph draws no sign; the badge is then its label alone.
type Glyph func(gtx layout.Context, sizePx int, col color.NRGBA)

// Style returns the type role a badge speaks at density d: LabelMedium at
// Comfortable, LabelSmall at Compact — one rung quieter than the chip's, which
// is what makes a badge visibly lighter than the controls it stands among.
//
// The density is identified by its control height and the badge takes nothing
// else from it: a badge is off the control ladder, and the returned style's
// line box is the whole of its height.
func Style(t tokens.Typography, d tokens.Density) tokens.TextStyle {
	if d.ControlHeight <= tokens.CompactControlHeight {
		return t.LabelSmall
	}
	return t.LabelMedium
}

// Ink returns the one colour a variant's badge reads in when it stands on
// ground: the role's pinned base while that base clears [tokens.TextFloor]
// against the storey, and otherwise the rung of the role's own ramp nearest
// the mid-value 500 that does.
//
// One colour and one floor for the whole badge, whichever of the three
// utterances it is making. WCAG 1.4.3's 4.5:1 is what words owe, and a badge
// that says its word as a sign instead is the same utterance at the same
// weight; 1.4.11's 3:1 governs a boundary, and a badge draws none. Deriving a
// glyph badge at the lower floor would make the three utterances read at three
// weights, which is the one thing a single anatomy is for.
//
// It is exported because anything drawn beside a badge — a container deciding
// what its own rule should clear, a test measuring the pairing — needs the
// answer the badge drew with, and re-deriving it at the call site is how two
// answers appear.
func Ink(c tokens.ColorTokens, v Variant, ground tokens.ElevationLevel) color.NRGBA {
	role := v.role()
	below := c.SurfaceAt(ground)
	if role == tokens.RoleNeutral {
		// InkOn refuses RoleNeutral, which has no pinned base; the walk is
		// the whole derivation for it.
		return c.MarkOn(role, below, tokens.TextFloor)
	}
	return c.InkOn(role, below, tokens.TextFloor)
}

// RenderState holds the explicit visual state a static badge render draws in.
// The zero value is a badge on the window ground with nothing under the
// pointer, so RenderState{} is the default badge.
//
// Intended for golden-image testing and static rendering; production code
// obtains the two pointer flags from the Gio event system.
type RenderState struct {
	// Ground is the elevation storey of the surface hosting the badge, in the
	// same vocabulary the host names its own fill (tokens.SurfaceAt). It is
	// the input to the one colour the badge resolves, because a badge has no
	// fill of its own to derive against. A dialog at tokens.Level2 passes
	// Level2. The zero value is tokens.Level0, the window ground.
	Ground tokens.ElevationLevel

	// DismissHovered and DismissPressed walk the close mark's ink one rung
	// and two toward the ramp's 900 end. They describe the mark alone: a
	// badge's body takes no pointer state, because there is nothing on it to
	// operate.
	DismissHovered bool
	DismissPressed bool
}

// state is the interaction state the close mark's ink is walked by.
func (s RenderState) state() tokens.State {
	switch {
	case s.DismissPressed:
		return tokens.StatePressed
	case s.DismissHovered:
		return tokens.StateHover
	}
	return tokens.StateNormal
}

// Props configures a [Badge] instance: what it says, which role it says it in,
// what it stands on, and whether it can be dismissed.
type Props struct {
	// Label is what the badge says — a word, or the digits of a count. An
	// empty Label with a non-nil Glyph is the glyph utterance.
	Label string

	// Glyph is the sign the badge draws, in the label's own line box, leading
	// the label across the spacing scale's S1 stop. A nil Glyph draws none.
	Glyph Glyph

	// Variant is the role the badge speaks in. The zero value is [Neutral].
	Variant Variant

	// Ground is the elevation storey of the surface hosting the badge, copied
	// straight into [RenderState.Ground] on every frame: the badge's ink is
	// derived against it. A dialog at tokens.Level2 passes Level2. The zero
	// value is tokens.Level0, the window ground.
	Ground tokens.ElevationLevel

	// Description is the screen-reader label. Falls back to Label when empty,
	// which is what a glyph badge needs — a sign with no words has nothing
	// for a reader to say unless the caller says it.
	Description string

	// OnDismiss makes the badge dismissible. When it is non-nil the badge
	// draws its close mark and calls this on a click; when it is nil the
	// badge draws no mark and registers no pointer area.
	//
	// It reports that the reader asked for this label to go away, and nothing
	// more. Dismissing a badge removes the label, never behaviour: the badge
	// does not hide itself on the next frame, and whatever it was about is
	// untouched.
	OnDismiss func(gtx layout.Context)

	// DismissMessage, if non-nil, is emitted as mvu.MessageOp into gtx.Ops on
	// dismissal — the MVU path, where OnDismiss is the FRP one. Both fire
	// when both are set, and they fire from the one place the click is
	// noticed.
	DismissMessage any

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the badge then shapes with the theme's shaper
	// (tokens.Typography.Shaper()), built once for the process and shared by
	// every component reading that typography. Set it only when this badge
	// must shape with a different one — a golden test pinning its faces.
	Shaper *text.Shaper
}

// resolvedTokens is the concrete per-emission snapshot the widget closure
// draws from: the whole theme flattened to the values one frame needs.
type resolvedTokens struct {
	color   tokens.ColorTokens
	spacing tokens.SpacingScale
	style   tokens.TextStyle // the density's badge role, per [Style]
	shaper  *text.Shaper
}

// Badge returns an rx.Observable[layout.Widget] emitting a new widget whenever
// the theme changes. It is the live face of [Render]: the same inline
// utterance, drawn from the theme rather than from tokens handed in, with the
// two things the pure path cannot carry — the close mark's pointer target and
// the dismissal dispatch.
//
// There is no keyboard path and no focus ring. A badge is read; the close mark
// is an affordance on a piece of text rather than a control in the tab order,
// and a badge that took focus would be one more stop between the reader and
// the controls that do something.
//
// A badge with no OnDismiss holds no interaction state at all. A dismissible
// one holds exactly one clickable, which the deferred scope keeps across every
// emission: a click lands on the frame after the one that drew the mark, so a
// clickable rebuilt per emission would drop whichever click a theme change
// happened to straddle.
func Badge(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	// Flatten the nested theme observables into one concrete snapshot. The
	// typography emission carries both the role [Style] picks at the emitted
	// density and the theme's cached shaper (ADR-003: the theme owns the
	// typeface).
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Color, t.Typography, t.Spacing, t.Density),
			func(n rx.Tuple4[tokens.ColorTokens, tokens.Typography, tokens.SpacingScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					color:   n.First,
					spacing: n.Third,
					style:   Style(typ, n.Fourth),
					shaper:  typ.Shaper(),
				}
			},
		)
	})

	return rx.Defer(func() rx.Observable[layout.Widget] {
		// Allocated once per subscription, so the close mark's hover and
		// press survive every theme emission.
		var dismiss widget.Clickable

		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			desc := props.Description
			if desc == "" {
				desc = props.Label
			}

			return func(gtx layout.Context) layout.Dimensions {
				s := RenderState{Ground: props.Ground}
				if props.OnDismiss == nil {
					return draw(gtx, shaper, props.Label, props.Glyph, props.Variant,
						tok, s, desc, false, nil)
				}
				// Drained to empty and reported once: a double click on a
				// close mark is one dismissal, not two, and the second click
				// left queued would fire on the next frame against a label
				// the caller has already taken away.
				dismissed := false
				for dismiss.Clicked(gtx) {
					dismissed = true
				}
				if dismissed {
					if props.OnDismiss != nil {
						props.OnDismiss(gtx)
					}
					if props.DismissMessage != nil {
						mvu.MessageOp{Message: props.DismissMessage}.Add(gtx.Ops)
					}
				}
				s.DismissHovered = dismiss.Hovered()
				s.DismissPressed = dismiss.Pressed()
				return draw(gtx, shaper, props.Label, props.Glyph, props.Variant,
					tok, s, desc, true, &dismiss)
			}
		})
	})
}

// Render produces a layout.Widget drawing the badge in an explicit visual
// state, without event processing: the glyph and the label on one line, in the
// one ink the variant resolves to against s.Ground, at the line box of the
// style handed in and no taller.
//
// glyph may be nil, in which case the badge is its label alone; label may be
// empty, in which case it is its glyph alone. style is the whole text style
// the badge is set in — pass [Style] of the density in play, which is
// tokens.DefaultTypography.LabelMedium at tokens.Comfortable.
//
// The badge is sized to its content and clamped to the constraints it is
// handed. Registering the close mark's pointer target is the live path's job —
// see [Badge] — or [RenderDismissible]'s, for a caller that owns the
// clickable.
func Render(
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	v Variant,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	style tokens.TextStyle,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, style: style}
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, glyph, v, tok, s, label, false, nil)
	}
}

// RenderDismissible is [Render] for a badge that carries its close mark: the
// same utterance, widened by the mark, with the mark's pointer area registered
// against dismiss.
//
// It is the static half of [Props.OnDismiss], for golden-image testing and
// demonstrations, and it takes the clickable rather than a callback because on
// this path there is no frame loop to drain one: the caller owns the
// clickable, lays the widget out, and reads Clicked itself. A nil dismiss
// draws the mark and registers nothing, which is what a still image wants.
func RenderDismissible(
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	v Variant,
	dismiss *widget.Clickable,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	style tokens.TextStyle,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, style: style}
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, glyph, v, tok, s, label, true, dismiss)
	}
}

// draw paints one badge: the line box tall, sized to the glyph, the label and
// the close mark it actually carries, every one of them in the ink the variant
// resolves against the storey underneath.
func draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	v Variant,
	tok resolvedTokens,
	s RenderState,
	desc string,
	dismissible bool,
	dismiss *widget.Clickable,
) layout.Dimensions {
	ink := Ink(tok.color, v, s.Ground)

	// The line box is the whole height: no padding, no floor, no control
	// height. A badge is an annotation on a line and is as tall as that line.
	lineBox := gtx.Dp(unit.Dp(tok.style.LineHeight))
	gap := gtx.Dp(unit.Dp(tok.spacing.S1))

	// The glyph's square is the label's own line box, the rule every inline
	// mark in this library follows.
	sign := 0
	if glyph != nil {
		sign = lineBox
	}
	mark := 0
	if dismissible {
		// Half the line box. Every type role's line height is a multiple of
		// four, so the mark is an even number of dp and centres in the box
		// without a half-pixel offset.
		mark = gtx.Dp(unit.Dp(tok.style.LineHeight / 2))
	}

	signGap, markGap := 0, 0
	if sign > 0 && label != "" {
		signGap = gap
	}
	if mark > 0 {
		markGap = gap
	}

	labelDims := layout.Dimensions{}
	var labelCall op.CallOp
	if label != "" {
		mColor := op.Record(gtx.Ops)
		paint.ColorOp{Color: ink}.Add(gtx.Ops)
		material := mColor.Stop()

		labelGtx := gtx
		labelGtx.Constraints.Min = image.Point{}
		if maxLabelW := gtx.Constraints.Max.X - sign - signGap - markGap - mark; maxLabelW > 0 {
			labelGtx.Constraints.Max.X = maxLabelW
		}
		mLabel := op.Record(gtx.Ops)
		// typeset.Layout rather than widget.Label.Layout because the role's
		// line height has to be the height of the label box, and Gio alone
		// reports the glyph ink instead — see theme/typeset.
		labelDims = typeset.Layout(labelGtx, shaper,
			typeset.Label(tok.style, 1), typeset.Font(tok.style, font.Normal),
			unit.Sp(tok.style.Size), label, material)
		labelCall = mLabel.Stop()
	}

	w := sign + signGap + labelDims.Size.X + markGap + mark
	h := max(lineBox, labelDims.Size.Y)
	w = min(w, gtx.Constraints.Max.X)
	h = min(h, gtx.Constraints.Max.Y)

	x := 0
	if sign > 0 {
		so := op.Offset(image.Pt(x, (h-sign)/2)).Push(gtx.Ops)
		glyph(gtx, sign, ink)
		so.Pop()
		x += sign + signGap
	}
	if label != "" {
		lo := op.Offset(image.Pt(x, (h-labelDims.Size.Y)/2)).Push(gtx.Ops)
		labelCall.Add(gtx.Ops)
		lo.Pop()
		x += labelDims.Size.X
	}

	size := image.Pt(w, h)

	// The badge's own semantic node, scoped to the box it drew. A semantic op
	// attaches to the innermost clip area around it, so a badge that emitted
	// its label without an area of its own would write that label onto
	// whatever area encloses it — and a page of badges would leave one
	// surviving name between them.
	//
	// The close mark stays outside this area on purpose: a pointer area is
	// clipped by the areas above it, and the mark's target is deliberately
	// larger than the badge.
	sem := clip.Rect{Max: size}.Push(gtx.Ops)
	semantic.LabelOp(label).Add(gtx.Ops)
	semantic.DescriptionOp(desc).Add(gtx.Ops)
	sem.Pop()

	if !dismissible {
		return layout.Dimensions{Size: size}
	}

	origin := image.Pt(x+markGap, (h-mark)/2)
	// The mark rides the badge's own ink and walks under the pointer at that
	// ink's hue and chroma, toward the ramp's 900 end — away from a light
	// ground and away from a dark one alike.
	drawClose(gtx, origin, mark, tok.color.PinnedStateColor(ink, s.state()))
	registerCloseTarget(gtx, desc, origin, mark, dismiss)
	return layout.Dimensions{Size: size}
}

// drawClose strokes the x in the mark-sized square at origin.
func drawClose(gtx layout.Context, origin image.Point, mark int, c color.NRGBA) {
	stroke := closeStrokeDp * gtx.Metric.PxPerDp
	if stroke < 1 {
		// A zero or unset metric would erase the mark; a sub-pixel width
		// would leave it a smear. Neither is better than the thinnest stroke
		// that draws.
		stroke = 1
	}
	// Inset by the stroke's half-width so the arms end inside the square
	// rather than bleeding a half-stroke past it on the diagonal.
	in := stroke / 2
	x0, y0 := float32(origin.X)+in, float32(origin.Y)+in
	x1, y1 := float32(origin.X+mark)-in, float32(origin.Y+mark)-in

	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(x0, y0))
	p.LineTo(f32.Pt(x1, y1))
	p.MoveTo(f32.Pt(x1, y0))
	p.LineTo(f32.Pt(x0, y1))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

// registerCloseTarget puts the clickable's pointer area over the mark, grown
// to [CloseHitDp] on each axis and centred on it — the drawn mark is 8 dp and
// the target it answers to is 24.
//
// The badge's own reported size is unaffected: a caller laying badges out
// spaces the words it can see, not the slop behind them, so where the slop of
// two targets overlaps the one laid out later wins it, exactly as Gio delivers
// to the topmost area.
func registerCloseTarget(gtx layout.Context, desc string, origin image.Point, mark int, dismiss *widget.Clickable) {
	if dismiss == nil {
		return
	}
	target := gtx.Dp(unit.Dp(CloseHitDp))
	if target < mark {
		target = mark
	}
	off := op.Offset(image.Pt(origin.X-(target-mark)/2, origin.Y-(target-mark)/2)).Push(gtx.Ops)
	dismiss.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.ClassOp(semantic.Button).Add(gtx.Ops)
		// The badge's own words name the target: what the mark removes is
		// this label, and a reader reaching the mark should be told which
		// label rather than a word this package invented for it.
		semantic.LabelOp(desc).Add(gtx.Ops)
		semantic.EnabledOp(true).Add(gtx.Ops)
		pointer.CursorPointer.Add(gtx.Ops)
		return layout.Dimensions{Size: image.Pt(target, target)}
	})
	off.Pop()
}
