// Package toast provides the toast: the status signal presenting a
// notification — a small surface floating at level 2 that appears when the
// notification is raised and leaves by itself after a set time. It draws one
// toast and nothing else; the column that receives notifications, places,
// stacks and times them is patterns/notifications.
//
// Toast is a callable Go function consuming a components theme observable,
// returning a stream of layout.Widget. Source is intentionally short and
// free of opaque configuration — copy it into your own app and modify as
// needed.
//
// Colour: level 2 is where a toast is placed, not what it is filled with.
// Nothing stands on a toast, so it is filled inverse — the token set's
// InverseSurface under its OnInverseSurface, the pair built from the
// counterpart scheme — which reads as speech and is found over any content
// in either scheme by construction rather than by out-raising it. The
// status role speaks through the leading edge, never through the fill.
//
// The cast shadow that says a toast floats and can leave is not drawn here:
// it belongs to whatever places the toast, because only the placement knows
// where the surface lands. patterns/notifications draws it under every
// toast it stands in its column.
package toast

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

	"github.com/reactivego/rx"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Role is the status role a toast speaks: the colour identity its leading
// edge carries. It is not a variant — a toast in Warning is a toast
// speaking Warning — and it is not a level, which is where the toast is
// placed.
type Role int

const (
	Info Role = iota
	Success
	Warning
	Error
)

// Toast surface metrics. A toast is a transient surface, not a control: its
// height hugs the message plus spacing-scale padding, and MinHeightDp is a
// legibility floor, so neither follows density.
const (
	// WidthDp is the width a toast takes when its constraints allow it.
	WidthDp = 240
	// MinHeightDp is the height no toast goes under however short its
	// message is.
	MinHeightDp = 36
)

// Props configures a Toast.
type Props struct {
	Role Role
	Text string

	// Alpha is the opacity every colour the toast paints is scaled by,
	// which is how a placement fades one out. A non-positive Alpha is
	// fully opaque, so the zero Props paints solid; a placement that wants
	// the toast invisible does not draw it.
	Alpha float64

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the toast then shapes its message with the
	// theme's shaper (Typography.Shaper()), which is built once for the
	// process and shared by every component reading that typography — the
	// cache lives behind the Typography value, so it survives the copy this
	// component's map function makes of it. Set it only when this instance
	// must shape with a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays every
	// layout.Widget out on the one goroutine that runs the event loop,
	// which is what makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

type resolvedTokens struct {
	color   tokens.ColorTokens
	spacing tokens.SpacingScale
	radius  tokens.RadiusScale
	style   tokens.TextStyle // the LabelMedium role: typeface, weight, size, line height
	shaper  *text.Shaper     // the theme's shaper; nil in the Render path
}

// Toast returns an rx.Observable[layout.Widget] that emits a new one
// whenever any consumed theme token changes.
func Toast(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the LabelMedium text style and the
	// theme's cached shaper: the theme owns the typeface.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Color, t.Spacing, t.Radius, t.Typography),
			func(n rx.Tuple4[tokens.ColorTokens, tokens.SpacingScale, tokens.RadiusScale, tokens.Typography]) resolvedTokens {
				typ := n.Fourth
				return resolvedTokens{
					color:   n.First,
					spacing: n.Second,
					radius:  n.Third,
					style:   typ.LabelMedium,
					shaper:  typ.Shaper(),
				}
			},
		)
	})
	return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
		// Props.Shaper is an explicit override; the theme's shaper is
		// the default.
		shaper := props.Shaper
		if shaper == nil {
			shaper = tok.shaper
		}
		return func(gtx layout.Context) layout.Dimensions {
			return draw(gtx, shaper, props, tok)
		}
	})
}

// Render produces a layout.Widget for a toast with pre-resolved tokens.
// Intended for golden-image testing, static demonstrations, and callers
// that already hold the theme's tokens — patterns/notifications among them.
// Production code that has a theme observable should use Toast, which takes
// the shaper and the text style off it.
//
// label is the LabelMedium role's whole text style — typeface, weight, size
// and line height all reach the shaper, exactly as they do on the live path.
// Pass tokens.DefaultTypography.LabelMedium for the default desktop look.
// There is no density parameter: a toast's height is a legibility floor
// around its message, not a control height.
func Render(
	shaper *text.Shaper,
	props Props,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	label tokens.TextStyle,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, style: label}
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, props, tok)
	}
}

// draw paints one toast sized to its message: a flat InverseSurface fill,
// the message in OnInverseSurface, and a leading edge two spacing stops
// wide in the status role's own ramp. Props.Alpha is applied to the fill,
// the edge and the text colour alike.
//
// The leading edge is the only place on the toast that identifies the
// status role, so its width has to clear the desktop's hairline band (the
// one to three px reserved for hairlines, separators and insets that are
// not meant to be looked at) while staying inside the horizontal air that
// holds the message off from it, or it stops reading as an edge and starts
// reading as a panel. See TestLeadingEdgeIsWiderThanTheHairlineBandAndNarrowerThanItsOwnAir.
func draw(gtx layout.Context, shaper *text.Shaper, props Props, tok resolvedTokens) layout.Dimensions {
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.spacing.S2))
	// The edge stays on the spacing scale rather than becoming a number of
	// its own so that the two measures the judgement above rests on — the
	// air above the message and the air beside it — keep their ratio to it
	// under any scale a theme emits.
	edgeW := gtx.Dp(unit.Dp(tok.spacing.S2))
	r := gtx.Dp(unit.Dp(tok.radius.Md))
	alpha := props.Alpha
	if alpha <= 0 {
		alpha = 1
	}

	w := gtx.Dp(unit.Dp(WidthDp))
	if w > gtx.Constraints.Max.X {
		w = gtx.Constraints.Max.X
	}
	if w < 0 {
		w = 0
	}

	fill := withAlpha(Fill(tok.color), alpha)
	edge := withAlpha(Edge(tok.color, props.Role), alpha)
	fg := withAlpha(Foreground(tok.color), alpha)

	// Pre-record the label so we can size the surface around its dims. The
	// leading edge takes its width off the label's, so the trailing margin
	// stays one padH and the text stands one padH clear of the edge.
	mColor := op.Record(gtx.Ops)
	paint.ColorOp{Color: fg}.Add(gtx.Ops)
	material := mColor.Stop()
	mLabel := op.Record(gtx.Ops)
	labelGtx := gtx
	labelGtx.Constraints = layout.Constraints{
		Max: image.Pt(w-edgeW-2*padH, gtx.Constraints.Max.Y),
	}
	// Shape with the LabelMedium role's typeface, weight, size and line
	// height. Zero fields fall back to the shaper's defaults.
	style := tok.style
	f := typeset.Font(style, font.Normal)
	wl := typeset.Label(style, 1)
	labelDims := typeset.Layout(labelGtx, shaper, wl, f, unit.Sp(style.Size), props.Text, material)
	labelCall := mLabel.Stop()

	h := labelDims.Size.Y + 2*padV
	minH := gtx.Dp(unit.Dp(MinHeightDp))
	if minH < gtx.Constraints.Min.Y {
		minH = gtx.Constraints.Min.Y
	}
	if h < minH {
		h = minH
	}

	rect := clip.RRect{Rect: image.Rectangle{Max: image.Pt(w, h)}, SE: r, SW: r, NE: r, NW: r}
	paint.FillShape(gtx.Ops, fill, rect.Op(gtx.Ops))
	// The role's edge, clipped to the surface so it wears the same rounded
	// corners the fill does rather than squaring off the leading side.
	clipped := rect.Op(gtx.Ops).Push(gtx.Ops)
	paint.FillShape(gtx.Ops, edge, clip.Rect{Max: image.Pt(edgeW, h)}.Op())
	clipped.Pop()

	labelY := padV
	if labelDims.Size.Y < h-2*padV {
		labelY = (h - labelDims.Size.Y) / 2
	}
	labelOff := op.Offset(image.Pt(edgeW+padH, labelY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	labelOff.Pop()

	return layout.Dimensions{Size: image.Pt(w, h)}
}

// Fill is the colour a toast's surface is filled with, at every status
// role: the token set's InverseSurface. Level 2 is where a toast is placed,
// not what it is filled with, so the fill tells no two toasts apart.
func Fill(c tokens.ColorTokens) color.NRGBA { return c.InverseSurface }

// Foreground is the colour a toast's message reads in: the token set's
// OnInverseSurface, the counterpart of Fill.
func Foreground(c tokens.ColorTokens) color.NRGBA { return c.OnInverseSurface }

// edgeFloor is the contrast the leading edge owes the inverse surface it
// sits on. The edge is a graphic and not text, so 3:1 would satisfy WCAG,
// but it is also the only thing on a toast that says which status role this
// is, so it is held to the body-text floor instead.
//
// The number does not bind, which is worth knowing before anyone tunes it.
// Over the whole seed sweep, both derivations, all four roles and both
// schemes, asking for 3.0 picks exactly the steps asking for 4.5 does: step
// 500 in a light scheme, never worse than 5.52:1 over the dark surface, and
// step 400 in a dark scheme, never worse than 7.58:1 over the light one.
// What chooses the step is the shape of the role's ramp against a surface
// built out of the counterpart scheme — see edgeColor — and not this floor.
const edgeFloor = 4.5

// Edge maps Role to the colour of the toast's leading edge: the step of that
// role's own ramp nearest the ramp's mid-value step that still clears
// edgeFloor over the inverse surface (tokens.MarkOn), so the edge flips
// with light/dark and follows whatever seed, palette or high-contrast
// variant the theme is emitting.
//
// It reads a ramp rather than a pinned base: the pins are tuned to be
// filled and written on, chosen against the scheme's own surfaces, and are
// on the wrong side of an inverse surface — a dark scheme's pins sit at
// L* 82, most of the way to that scheme's own light surface.
//
// It asks for a step near the ramp's middle rather than naming a fixed
// one, because a single step cannot read over both schemes' surfaces at
// every hue without losing chroma: a light scheme lands on step 500 at
// all four roles — the step where each role holds its anchor's full
// chroma — and a dark scheme is forced to step 400, the nearest-to-middle
// step that still reads over its own light surface at all.
//
// Info reads the info ramp rather than the accent one, so an informational
// toast's colour says "info" regardless of the brand's own hue; the info
// role is anchored on a blue of its own, so the four stay four whatever the
// seed.
func Edge(c tokens.ColorTokens, r Role) color.NRGBA {
	var role tokens.Role
	switch r {
	case Error:
		role = tokens.RoleError
	case Success:
		role = tokens.RoleSuccess
	case Warning:
		role = tokens.RoleWarning
	default:
		role = tokens.RoleInfo
	}
	return c.MarkOn(role, Fill(c), edgeFloor)
}

func withAlpha(c color.NRGBA, a float64) color.NRGBA {
	if a >= 1 {
		return c
	}
	out := c
	out.A = uint8(float64(c.A)*a + 0.5)
	return out
}
