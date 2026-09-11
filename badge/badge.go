package badge

import (
	"image"
	"image/color"
	"math"

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

	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Status is the status a badge indicates, or [Neutral] for none. The five
// differ in hue and in nothing else: same type, same box, same structure.
// There is no emphasis axis — see the package doc.
type Status uint8

const (
	// Neutral is the plain category label, and the zero value: a badge
	// given no status is Neutral, naming a kind rather than reporting a
	// status. It is the one value the platform has no status colour for,
	// so it wears the platform's grey instead.
	Neutral Status = iota
	// Success, Warning, Error and Info are the four statuses, each in the
	// platform's system colour for it.
	Success
	Warning
	Error
	Info
)

// systemColor is the platform's colour for the status: systemGreen,
// systemOrange, systemRed and systemBlue for the four, systemGray for
// [Neutral], which the platform gives no status colour of its own.
func (status Status) systemColor(p tokens.PlatformColors) color.NRGBA {
	switch status {
	case Success:
		return p.SystemGreen
	case Warning:
		return p.SystemOrange
	case Error:
		return p.SystemRed
	case Info:
		return p.SystemBlue
	}
	return p.SystemGray
}

// CloseHitDp is the side of the pointer target the close mark claims, in
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

// Style returns the type role a badge is set in at density d: LabelMedium at
// Comfortable, LabelSmall at Compact — one step less pronounced than the
// chip's, which is what makes a badge visibly lighter than the controls it
// stands among.
//
// The density is identified by its control height and the badge takes nothing
// else from it: a badge is off the control family, and the returned style's
// line box is the whole of its height.
func Style(t tokens.Typography, d tokens.Density) tokens.TextStyle {
	if d.ControlHeight <= tokens.CompactControlHeight {
		return t.LabelSmall
	}
	return t.LabelMedium
}

// BareForeground is the colour a BARE glyph badge draws its sign in: the
// platform's system colour for the status itself, and [tokens.PlatformColors.SecondaryLabel]
// for [Neutral], which the platform gives no status colour.
//
// A bare badge is the glyph utterance — see [Fill] for why only that one
// stands without a fill. A worded or counted badge reads in [Foreground]
// against the fill it wears instead.
//
// The platform's system colours are read on every fill a window carries,
// which is what a system colour is for, so nothing here is derived against
// standsOn: it is the surface the sign is drawn on, and the only colour that
// needs it is Neutral's secondary label, which carries a coverage.
func BareForeground(p tokens.PlatformColors, status Status, standsOn color.NRGBA) color.NRGBA {
	if status == Neutral {
		return vgcolor.Flatten(p.SecondaryLabel, standsOn)
	}
	return status.systemColor(p)
}

// Fill is the fill a worded or counted badge wears: the platform's system
// colour for the status, at full strength, which is how the platform draws
// a count badge. A [Neutral] badge wears systemGray, the platform's answer
// where no status is carried.
//
// Filled rather than tinted, because that is what the platform draws: a
// count badge on this platform is a solid capsule of the status colour with
// its digits knocked out in white, and a tint of it is not a colour the
// platform has a name for.
//
// The fill is the badge's second channel, and the reason it exists is that
// hue cannot be the only one. A reader who does not separate the four
// status hues — and a red/green pair is the commonest deficiency there is —
// has, on a bare badge, nothing else to read. A field of the status colour
// puts the difference in a second place, in a region big enough to be seen
// without being looked at.
//
// The GLYPH utterance stands bare, and it is the one exception: the
// invariant is that hue is never the badge's only channel, not that every
// badge wears a fill, and a glyph carries its meaning in its shape. A check
// and a cross differ for a reader who sees neither hue.
//
// A glyph badge may still be given this fill as a disc ([RenderState.Disc]),
// which is a request and not a second structure: the same fill and the same
// [Foreground], put behind the sign as a circle rather than behind a word as
// a rounded box. The bare sign stays the default, because standing bare is
// what lets a verdict sit inside a line of controls without reading as one
// more field.
//
// Which is an obligation on the caller and not a property the package can
// hold. [Props.Glyph] is a painter this package cannot inspect, so two glyph
// badges drawn with ONE sign under two statuses are two hues and nothing else —
// the exact channel collapse the fill exists to prevent, reintroduced above
// the component. A set of glyph badges owes distinct shapes; a set that
// cannot have them owes words instead.
func Fill(p tokens.PlatformColors, status Status) color.NRGBA {
	return status.systemColor(p)
}

// Foreground is the colour a worded or counted badge's content reads in: the
// foreground the platform pairs with a fill its accent or a system colour
// paints, alternateSelectedControlTextColor — white in both appearances.
//
// It does not vary with the status. The platform's count badge is white on
// every one of its system colours, in both appearances, and a second answer
// would make the five badges differ in two channels where the platform
// differs in one.
func Foreground(p tokens.PlatformColors) color.NRGBA {
	return p.AlternateSelectedControlText
}

// RenderState holds the explicit visual state a static badge render draws in.
// The zero value is a badge with nothing under the pointer and a sign
// standing bare, so RenderState{} is the default badge.
//
// Intended for golden-image testing and static rendering; production code
// obtains the two pointer flags from the Gio event system.
type RenderState struct {
	// DismissHovered and DismissPressed lay the platform's hover and press
	// overlay over the close mark's region — over the fill's trailing cap on
	// a badge that wears a fill, over whatever the badge stands on when it
	// stands bare. They describe the mark alone: a badge's body takes no
	// pointer state, because there is nothing on it to operate.
	DismissHovered bool
	DismissPressed bool

	// Disc asks a glyph badge to stand on the status's fill instead of bare:
	// a circle the glyph's line box across, the sign centred in it in
	// [Foreground] rather than [BareForeground] and drawn in the square
	// inscribed in that circle. It is a request and not a structure of its
	// own — a badge with a label already wears its fill and ignores this,
	// and a badge with neither label nor glyph has nothing to put a disc
	// behind.
	//
	// The bare sign is the default, so the zero value leaves every badge
	// drawn today untouched.
	Disc bool

	// Surface is the opaque fill the badge stands on. A bare badge wears no
	// fill of its own, so its sign, its close mark and the mark's hover and
	// press overlays all land on this, and the platform's secondary label and
	// its overlays carry a coverage rather than a colour. The zero value — no
	// colour — is the window's own plane.
	Surface color.NRGBA
}

// overlay is what the close mark's region lays over beneath — the fill on a
// badge that wears one, the surface it stands on when it stands bare: the
// platform's press overlay while the mark is held, its hover overlay under
// the pointer, and nothing at rest.
func (s RenderState) overlay(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	switch {
	case s.DismissPressed:
		return vgcolor.Flatten(p.PressOverlay, beneath)
	case s.DismissHovered:
		return vgcolor.Flatten(p.HoverOverlay, beneath)
	}
	return color.NRGBA{}
}

// Props configures a [Badge] instance: what it says, which status it
// indicates, what it stands on, and whether it can be dismissed.
type Props struct {
	// Label is what the badge says — a word, or the digits of a count. An
	// empty Label with a non-nil Glyph is the glyph utterance.
	Label string

	// Glyph is the sign the badge draws, in the label's own line box, leading
	// the label across the spacing scale's S1 stop. A nil Glyph draws none.
	Glyph Glyph

	// Disc asks a glyph badge — a non-nil Glyph with an empty Label — to
	// stand on the status's fill: a circle the glyph's line box across, the
	// sign centred in it and drawn smaller to fit. Copied straight into
	// [RenderState.Disc]. A badge with a label ignores it.
	Disc bool

	// Surface is the opaque fill the badge stands on, copied straight into
	// RenderState on every frame. Set it where the badge does not stand on
	// the window's own plane. See RenderState.Surface.
	Surface color.NRGBA

	// Status is the status the badge indicates. The zero value is [Neutral]:
	// a badge given no status is Neutral.
	Status Status

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

// resolvedTokens is the concrete per-emission snapshot the layout.Widget
// closure draws from: the whole theme flattened to the values one frame needs.
type resolvedTokens struct {
	platform tokens.PlatformColors
	spacing  tokens.SpacingScale
	radius   tokens.RadiusScale
	style    tokens.TextStyle // the density's badge role, per [Style]
	shaper   *text.Shaper
}

// discSign is the square a sign is handed inside a disc px across: the square
// inscribed in the circle, rounded down to the diameter's own parity so the
// sign centres on whole pixels rather than half of one.
//
// Inscribed rather than fitted to a particular sign, and that is the whole
// reason for it. A [Glyph] is a painter this package cannot inspect, and the
// contract it is written to is that a sign spans most of the box it is handed
// — so a sign handed the disc's full square lands on the rim, and one drawn
// corner to corner spills past it. The inscribed square is the largest box
// whose every point is inside the circle, so it is the only size that holds
// for a painter the badge has not seen.
func discSign(px int) int {
	in := int(float32(px) / math.Sqrt2)
	if (px-in)%2 != 0 {
		in--
	}
	return in
}

// fillPad is the space the fill holds on each side between its edge and the
// content: the spacing scale's S2 stop, twice the S1 stop the badge already
// sets its own parts across.
//
// The ratio is the whole reason for the choice. A sign and the word it stands
// for are one utterance and sit an S1 apart; if the fill's edge sat that
// close too, the word would be as near the box as it is to the sign, and the
// two would stop reading as one thing inside a box. Doubling the inner gap is
// the smallest stop that groups them.
//
// There is no vertical padding, and none is missing: the type role's line box
// carries its own leading — 16 dp of box around a 12 sp face — so a fill drawn
// at the line box already stands about 3 dp clear of the label's cap and
// descender. Padding on top of that would take the badge off its own line.
func fillPad(sp tokens.SpacingScale) float32 { return sp.S2 }

// fillRadius is the fill's corner: the radius scale's Base stop.
//
// Deliberately not Full. The pill is components/chip's shape, and a badge is
// the thing a chip must not be confused with — same rough size, same inline
// placement, opposite originator. A quarter of the badge's height reads as rounded
// without reading as a capsule, which leaves the silhouette doing the same
// work the fill does: telling a reader which of the two families this is.
func fillRadius(rad tokens.RadiusScale) float32 { return rad.Base }

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
			rx.CombineLatest5(t.Platform, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					spacing:  n.Third,
					radius:   n.Fourth,
					style:    Style(typ, n.Fifth),
					shaper:   typ.Shaper(),
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
				s := RenderState{Disc: props.Disc, Surface: props.Surface}
				if props.OnDismiss == nil {
					return draw(gtx, shaper, props.Label, props.Glyph, props.Status,
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
				return draw(gtx, shaper, props.Label, props.Glyph, props.Status,
					tok, s, desc, true, &dismiss)
			}
		})
	})
}

// Render produces a layout.Widget drawing the badge in an explicit visual
// state, without event processing: the glyph and the label on one line, at the
// line box of the style handed in and no taller.
//
// glyph may be nil, in which case the badge is its label alone; label may be
// empty, in which case it is its glyph alone. That choice is also the choice
// of structure: a badge with a label wears its status's [Fill] and reads in
// [Foreground], and a glyph-only badge stands bare and reads in
// [BareForeground] — unless s.Disc asks the glyph badge to
// stand on a disc, which puts it back on [Fill] and [Foreground]. style is the
// whole text style the badge is set in — pass [Style] of the density in play,
// which is tokens.DefaultTypography.LabelMedium at tokens.Comfortable.
//
// The badge is sized to its content and clamped to the constraints it is
// handed, and reports the baseline of its label so a row can set it on the
// same line as the text beside it. Registering the close mark's pointer target
// is the live path's job — see [Badge] — or [RenderDismissible]'s, for a
// caller that owns the clickable.
func Render(
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	status Status,
	colors tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	style tokens.TextStyle,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: colors, spacing: sp, radius: rad, style: style}
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, glyph, status, tok, s, label, false, nil)
	}
}

// RenderDismissible is [Render] for a badge that carries its close mark: the
// same utterance, widened by the mark, with the mark's pointer area registered
// against dismiss.
//
// It is the static half of [Props.OnDismiss], for golden-image testing and
// demonstrations, and it takes the clickable rather than a callback because on
// this path there is no frame loop to drain one: the caller owns the
// clickable, lays it out, and reads Clicked itself. A nil dismiss draws the
// mark and answers no pointer, which is what a still image wants.
func RenderDismissible(
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	status Status,
	dismiss *widget.Clickable,
	colors tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	style tokens.TextStyle,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: colors, spacing: sp, radius: rad, style: style}
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, glyph, status, tok, s, label, true, dismiss)
	}
}

// draw paints one badge: the line box tall, sized to the glyph, the label and
// the close mark it actually carries, over the fill the utterance calls for
// and in the foreground the platform pairs with that fill.
func draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	glyph Glyph,
	status Status,
	tok resolvedTokens,
	s RenderState,
	desc string,
	dismissible bool,
	dismiss *widget.Clickable,
) layout.Dimensions {
	// The utterance picks the structure: anything with words in it wears the
	// fill, a sign on its own stands bare unless the caller asked for the
	// disc. See [Fill].
	//
	// A label ignores [RenderState.Disc] rather than being refused it: the
	// fill the words already wear IS the fill the disc would add, so
	// honouring both would mean a badge inside a badge, and there is exactly
	// one structure branch here.
	worded := label != ""
	disc := !worded && glyph != nil && s.Disc
	standsOn := surface.Or(s.Surface, tok.platform.WindowBackground)
	var fill, fg color.NRGBA
	if worded || disc {
		fill, fg = Fill(tok.platform, status), Foreground(tok.platform)
	} else {
		fg = BareForeground(tok.platform, status, standsOn)
	}

	// The line box is the whole height: no vertical padding, no floor, no
	// control height. A badge is an annotation on a line and is as tall as
	// that line, filled or not.
	lineBox := gtx.Dp(unit.Dp(tok.style.LineHeight))
	gap := gtx.Dp(unit.Dp(tok.spacing.S1))
	// A disc takes no padding: its circle is inscribed in the glyph's own
	// square, so a disc badge's box is the same line-box square a bare sign
	// reports and a row that reserved room for one holds the other unmoved.
	pad := 0
	if worded {
		pad = gtx.Dp(unit.Dp(fillPad(tok.spacing)))
	}

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
		paint.ColorOp{Color: fg}.Add(gtx.Ops)
		material := mColor.Stop()

		labelGtx := gtx
		labelGtx.Constraints.Min = image.Point{}
		if maxLabelW := gtx.Constraints.Max.X - 2*pad - sign - signGap - markGap - mark; maxLabelW > 0 {
			labelGtx.Constraints.Max.X = maxLabelW
		}
		mLabel := op.Record(gtx.Ops)
		// typeset.Layout rather than widget.Label.Layout because the role's
		// line height has to be the height of the label box, and Gio alone
		// reports the drawn glyph extent instead — see theme/typeset.
		labelDims = typeset.Layout(labelGtx, shaper,
			typeset.Label(tok.style, 1), typeset.Font(tok.style, font.Normal),
			unit.Sp(tok.style.Size), label, material)
		labelCall = mLabel.Stop()
	}

	w := 2*pad + sign + signGap + labelDims.Size.X + markGap + mark
	h := max(lineBox, labelDims.Size.Y)
	w = min(w, gtx.Constraints.Max.X)
	h = min(h, gtx.Constraints.Max.Y)

	size := image.Pt(w, h)
	// The corner is clamped to half the height, so a badge squeezed by its
	// constraints rounds to a stadium rather than drawing a corner larger
	// than the box it belongs to.
	radius := min(gtx.Dp(unit.Dp(fillRadius(tok.radius))), h/2)
	if worded {
		paint.FillShape(gtx.Ops, fill, clip.RRect{
			Rect: image.Rectangle{Max: size},
			NW:   radius, NE: radius, SE: radius, SW: radius,
		}.Op(gtx.Ops))
	}

	x := pad
	if sign > 0 {
		so := op.Offset(image.Pt(x, (h-sign)/2)).Push(gtx.Ops)
		inner := sign
		if disc {
			// The circle inscribed in the glyph's square, so the diameter is
			// the line box; the sign is then drawn in the square inscribed in
			// THAT circle, centred on it.
			paint.FillShape(gtx.Ops, fill, clip.Ellipse{Max: image.Pt(sign, sign)}.Op(gtx.Ops))
			inner = discSign(sign)
		}
		io := op.Offset(image.Pt((sign-inner)/2, (sign-inner)/2)).Push(gtx.Ops)
		glyph(gtx, inner, fg)
		io.Pop()
		so.Pop()
		x += sign + signGap
	}
	if label != "" {
		lo := op.Offset(image.Pt(x, (h-labelDims.Size.Y)/2)).Push(gtx.Ops)
		labelCall.Add(gtx.Ops)
		lo.Pop()
		x += labelDims.Size.X
	}

	// The label's baseline, re-reported against the badge's own bottom edge.
	// layout.Dimensions.Baseline is measured up from the bottom, so what the
	// badge owes is what typeset reported plus whatever the badge left under
	// the label — which is nothing at all while the line box is the height,
	// and is the rounding of an odd remainder when a caller's constraints
	// squeeze it. A badge that reported no baseline is why a row aligned on
	// layout.Baseline had nothing to align on and fell back to the box.
	//
	// A glyph-only badge reports none: a sign has no baseline to offer, and
	// zero is what Gio reads as "align me by my box".
	baseline := 0
	if label != "" {
		below := h - labelDims.Size.Y
		baseline = labelDims.Baseline + below - below/2
	}

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
		return layout.Dimensions{Size: size, Baseline: baseline}
	}

	origin := image.Pt(x+markGap, (h-mark)/2)
	markFg := fg
	if !worded {
		// A bare sign's mark is not on the sign's own colour: it is an
		// affordance on a piece of text, so it takes the platform's
		// secondary label like every other small mark beside a word,
		// flattened onto the surface the badge stands on.
		markFg = vgcolor.Flatten(tok.platform.SecondaryLabel, standsOn)
	}

	// The close mark's REGION is what answers the pointer, not the 8 dp x
	// inside it: a field is what makes a 24 dp target findable at all. On a
	// badge that wears a fill the region is that fill's trailing cap — from
	// the middle of the gap that separates the mark from the label out to
	// the fill's own edge and corner; on a bare badge it is the mark's own
	// square. The platform's hover and press overlays carry a coverage, so
	// each is flattened onto what it actually lands on: the status fill on a
	// worded badge, the surface a bare one stands on.
	overlayOn := standsOn
	if worded {
		overlayOn = fill
	}
	if overlay := s.overlay(tok.platform, overlayOn); overlay.A != 0 {
		if worded {
			left := origin.X - markGap/2
			paint.FillShape(gtx.Ops, overlay, clip.RRect{
				Rect: image.Rectangle{Min: image.Pt(left, 0), Max: size},
				NE:   radius, SE: radius,
			}.Op(gtx.Ops))
		} else {
			markRad := min(mark/2, gtx.Dp(unit.Dp(fillRadius(tok.radius))))
			paint.FillShape(gtx.Ops, overlay, clip.RRect{
				Rect: image.Rectangle{Min: origin, Max: origin.Add(image.Pt(mark, mark))},
				NW:   markRad, NE: markRad, SE: markRad, SW: markRad,
			}.Op(gtx.Ops))
		}
	}
	drawClose(gtx, origin, mark, markFg)
	registerCloseTarget(gtx, desc, origin, mark, dismiss)
	return layout.Dimensions{Size: size, Baseline: baseline}
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
