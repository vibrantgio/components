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
// nothing else: same type, same box, same structure. There is no emphasis axis
// — see the package doc.
type Variant uint8

const (
	// Neutral is the plain category label, and the zero value: a badge that
	// names a kind rather than reporting a status. It is the only variant
	// with no pinned base behind it, so its foreground is always a measured
	// step of the neutral ramp.
	Neutral Variant = iota
	// Success, Warning, Error and Info are the four statuses, each in its own
	// role's hue.
	Success
	Warning
	Error
	Info
)

// role is the colour role a variant reads its colours off.
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

// BareForeground returns the one colour a BARE variant's badge reads in when
// it stands on a surface: the role's pinned base while that base clears
// [tokens.TextFloor] against that surface, and otherwise the step of the
// role's own ramp nearest the mid-value 500 that does.
//
// A bare badge is the glyph utterance — see [Fill] for why only that one
// stands without a container. A worded or counted badge reads in [Foreground]
// instead, against its fill rather than against the surface beneath.
//
// One colour and one floor for the whole badge, whichever of the three
// utterances it is making. WCAG 1.4.3's 4.5:1 is what words owe, and a badge
// that says its word as a sign instead is the same utterance at the same
// weight; 1.4.11's 3:1 governs a boundary, and a badge's container draws none
// — the fill is a region and not an edge. Deriving a glyph badge at the lower
// floor would make the three utterances read at three weights, which is the
// one thing a single structure is for.
//
// It is exported because anything drawn beside a badge — a host deciding what
// its own rule should clear, a test measuring the pairing — needs the answer
// the badge drew with, and re-deriving it at the call site is how two answers
// appear.
func BareForeground(c tokens.ColorTokens, v Variant, level tokens.ElevationLevel) color.NRGBA {
	return ForegroundOver(c, v, c.SurfaceAt(level))
}

// ForegroundOver is the badge's variant applied to the shared foreground
// derivation ([tokens.ColorTokens.ForegroundOn]): the role's own hue at
// reading strength over any surface at all. [BareForeground] is it over a
// level's fill and [Foreground] over a container fill.
//
// It is exported for the third case, the one neither of those names: the
// fill walked under a pointer. A close mark whose surface has just moved two
// steps and whose colour has not is a mark derived against a surface that is
// no longer there, and 4.5:1 at rest becomes 2.3:1 pressed — measured, before
// this was the rule.
func ForegroundOver(c tokens.ColorTokens, v Variant, surface color.NRGBA) color.NRGBA {
	return c.ForegroundOn(v.role(), surface)
}

// Fill returns the container fill a worded or counted badge wears: a pale,
// tint of the role's own hue relative to the surface beneath, at the one
// chroma every tonal
// container in this system is realized at
// ([tokens.ColorTokens.StatusContainerOn]).
//
// Pale is the whole of it. The badge is a statement — the system's word about
// a thing — and a statement is read, never operated; a saturated fill under a
// knocked-out foreground is the variant interaction speaks in, and a badge
// borrowing it would claim to be a control. So the container carries the role
// at low prominence and the content carries the same hue at reading strength
// ([Foreground]): one hue, two strengths, no inversion anywhere.
//
// The fill is the badge's second channel, and the reason it exists is that hue
// cannot be the only one. A reader who does not separate the four status hues
// — and a red/green pair is the commonest deficiency there is — has, on a bare
// badge, nothing else to read: the five variants are one structure in five
// colours, so the whole distinction between "Passing" and "Failing" would sit
// in a channel some readers do not receive. A field of the same role puts the
// difference in a second place, in a region big enough to be seen without
// being looked at.
//
// Against the surface beneath rather than at a fixed depth, because a badge is
// small and goes wherever it is put. The elevation levels walk through the
// depth a fixed container is realized at, and a fill that lands on the level it
// is
// tinting is not subtle, it is absent — a dark scheme's level-2 surface and
// its own step-300 container measure 1.00:1.
//
// [Neutral] derives the same way and is not a special case: the derivation
// asks the ramp rather than a table, and the neutral ramp carries no chroma,
// so a neutral fill comes back as depth alone. That is what separates a
// Neutral badge from the prose around it — a plain category label in the
// text colour on the page's own surface is prose, whatever the type role
// says.
//
// The GLYPH utterance stands bare, and it is the one exception: the invariant
// is that hue is never the badge's only channel, not that every badge wears a
// container, and a glyph carries its meaning in its shape. A check and a cross
// differ for a reader who sees neither hue.
//
// Which is an obligation on the caller and not a property the package can
// hold. [Props.Glyph] is a painter this package cannot inspect, so two glyph
// badges drawn with ONE sign in two variants are two hues and nothing else —
// the exact channel collapse the fill exists to prevent, reintroduced above
// the component. A set of glyph badges owes distinct shapes; a set that
// cannot have them owes words instead.
func Fill(c tokens.ColorTokens, v Variant, level tokens.ElevationLevel) color.NRGBA {
	return c.StatusContainerOn(v.role(), c.SurfaceAt(level))
}

// Foreground returns the colour a worded or counted badge's content reads in:
// the role's own hue at reading strength over the container — that is,
// [BareForeground]'s derivation run against the [Fill] the badge wears instead
// of against the surface underneath it.
//
// Same hue, floored for contrast; never an inverted on-colour. A white word on
// a saturated field is the other variant entirely.
//
// It is not [tokens.ColorTokens.OnStatusContainer], which answers a
// neighbouring question at [tokens.GraphicFloor] for a mark on a container. A
// badge's content is text — or a sign standing in for text — so it owes the
// text floor wherever it is drawn, and running the badge's own derivation
// against the new fill is what keeps the three utterances at one weight
// after the container arrives.
func Foreground(c tokens.ColorTokens, v Variant, level tokens.ElevationLevel) color.NRGBA {
	return ForegroundOver(c, v, Fill(c, v, level))
}

// RenderState holds the explicit visual state a static badge render draws in.
// The zero value is a badge on the window's own surface with nothing under the
// pointer, so RenderState{} is the default badge.
//
// Intended for golden-image testing and static rendering; production code
// obtains the two pointer flags from the Gio event system.
type RenderState struct {
	// Level is the level of the surface the badge stands on — a badge has no
	// level of its own — in the same vocabulary the host names its own fill
	// (tokens.SurfaceAt). It is the input to the one colour the badge
	// resolves, because a badge has no fill of its own to derive against. A
	// dialog at tokens.Level2 passes Level2. The zero value is tokens.Level0,
	// the window's own surface.
	Level tokens.ElevationLevel

	// DismissHovered and DismissPressed walk the close mark's region one step
	// and two toward the ramp's 900 end — the container's fill under the
	// mark on a badge that wears one, the mark's own colour on a bare one. They
	// describe the mark alone: a badge's body takes no pointer state, because
	// there is nothing on it to operate.
	DismissHovered bool
	DismissPressed bool
}

// state is the interaction state the close mark's region is walked by.
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

	// Level is the level of the surface the badge stands on — a badge has no
	// level of its own — copied straight into [RenderState.Level] on every
	// frame: the badge's colours are derived against it. A dialog at
	// tokens.Level2 passes Level2. The zero value is tokens.Level0, the
	// window's own surface.
	Level tokens.ElevationLevel

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
	radius  tokens.RadiusScale
	style   tokens.TextStyle // the density's badge role, per [Style]
	shaper  *text.Shaper
}

// containerPad is the space the container holds on each side between its edge
// and the content: the spacing scale's S2 stop, twice the S1 stop the badge
// already sets its own parts across.
//
// The ratio is the whole reason for the choice. A sign and the word it stands
// for are one utterance and sit an S1 apart; if the container's edge sat that
// close too, the word would be as near the box as it is to the sign, and the
// two would stop reading as one thing inside a box. Doubling the inner gap is
// the smallest stop that groups them.
//
// There is no vertical padding, and none is missing: the type role's line box
// carries its own leading — 16 dp of box around a 12 sp face — so a fill drawn
// at the line box already stands about 3 dp clear of the label's cap and
// descender. Padding on top of that would take the badge off its own line.
func containerPad(sp tokens.SpacingScale) float32 { return sp.S2 }

// containerRadius is the container's corner: the radius scale's Base stop.
//
// Deliberately not Full. The pill is components/chip's shape, and a badge is
// the thing a chip must not be confused with — same rough size, same inline
// placement, opposite voice. A quarter of the badge's height reads as rounded
// without reading as a capsule, which leaves the silhouette doing the same
// work the fill does: telling a reader which of the two families this is.
func containerRadius(rad tokens.RadiusScale) float32 { return rad.Base }

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
			rx.CombineLatest5(t.Color, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.ColorTokens, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					color:   n.First,
					spacing: n.Third,
					radius:  n.Fourth,
					style:   Style(typ, n.Fifth),
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
				s := RenderState{Level: props.Level}
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
// state, without event processing: the glyph and the label on one line, at the
// line box of the style handed in and no taller.
//
// glyph may be nil, in which case the badge is its label alone; label may be
// empty, in which case it is its glyph alone. That choice is also the choice
// of structure: a badge with a label wears its role's [Fill] and reads in
// [Foreground], and a glyph-only badge stands bare and reads in
// [BareForeground] against s.Level. style is the whole text style the badge is
// set in — pass [Style]
// of the density in play, which is tokens.DefaultTypography.LabelMedium at
// tokens.Comfortable.
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
	v Variant,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	style tokens.TextStyle,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, style: style}
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
	rad tokens.RadiusScale,
	style tokens.TextStyle,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, style: style}
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, glyph, v, tok, s, label, true, dismiss)
	}
}

// draw paints one badge: the line box tall, sized to the glyph, the label and
// the close mark it actually carries, over the container the utterance calls
// for and in the foreground the variant resolves against whatever ends up
// underneath the content.
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
	// The utterance picks the structure: anything with words in it wears the
	// container, a sign on its own stands bare. See [Fill].
	contained := label != ""
	var fill, fg color.NRGBA
	if contained {
		fill, fg = Fill(tok.color, v, s.Level), Foreground(tok.color, v, s.Level)
	} else {
		fg = BareForeground(tok.color, v, s.Level)
	}

	// The line box is the whole height: no vertical padding, no floor, no
	// control height. A badge is an annotation on a line and is as tall as
	// that line, container or not.
	lineBox := gtx.Dp(unit.Dp(tok.style.LineHeight))
	gap := gtx.Dp(unit.Dp(tok.spacing.S1))
	pad := 0
	if contained {
		pad = gtx.Dp(unit.Dp(containerPad(tok.spacing)))
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
		// reports the glyph ink instead — see theme/typeset.
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
	radius := min(gtx.Dp(unit.Dp(containerRadius(tok.radius))), h/2)
	if contained {
		paint.FillShape(gtx.Ops, fill, clip.RRect{
			Rect: image.Rectangle{Max: size},
			NW:   radius, NE: radius, SE: radius, SW: radius,
		}.Op(gtx.Ops))
	}

	x := pad
	if sign > 0 {
		so := op.Offset(image.Pt(x, (h-sign)/2)).Push(gtx.Ops)
		glyph(gtx, sign, fg)
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
	if contained {
		// The close mark's REGION is what walks: the container hands the
		// pointer a field of colour to answer with, and a field is what makes
		// an 8 dp x with a 24 dp target findable at all. The zone is the
		// fill's right cap — from the middle of the gap that separates the
		// mark from the label out to the fill's own edge and corner — so at
		// rest it is the fill, indistinguishable, and under the pointer the
		// badge grows a visibly deeper end.
		//
		// The mark then re-derives against what is actually behind it. A mark
		// that held its resting colour over a fill walked two steps would be
		// derived against a surface no longer there: measured, 4.5:1 at rest
		// and 2.3:1 pressed, the state making the affordance harder to see the
		// more the reader commits to it.
		zone := tok.color.PinnedStateColor(fill, s.state())
		if zone != fill {
			left := origin.X - markGap/2
			paint.FillShape(gtx.Ops, zone, clip.RRect{
				Rect: image.Rectangle{Min: image.Pt(left, 0), Max: size},
				NE:   radius, SE: radius,
			}.Op(gtx.Ops))
			markFg = ForegroundOver(tok.color, v, zone)
		}
	} else {
		// Bare, there is no fill to walk, so the mark's own colour does it —
		// toward the ramp's 900 end at that colour's hue and chroma, which is
		// away from a light surface and away from a dark one alike.
		markFg = tok.color.PinnedStateColor(fg, s.state())
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
