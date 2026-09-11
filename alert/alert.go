// Package alert provides the alert: the status signal for a situation — a
// rounded box with a leading status icon, a Title, and an arbitrary Body
// layout.Widget, standing in the page flow until the situation resolves. It
// holds words about the situation, never a control: an action on the
// situation stands beside the alert. The statuses are Info, Success, Warning
// and Error; an alert given no status is Info.
//
// Alert is a callable Go function consuming a components theme observable,
// returning a stream of layout.Widget. Source is intentionally short and
// free of opaque configuration — copy it into your own app and modify as
// needed.
//
// All four statuses draw the same right-pointing chevron glyph, differing
// only in colour; the per-status icon set arrives with components/icon.
//
// Colour: every one of the platform's own names. The box stands on the
// content's fill — controlBackgroundColor — inside a separatorColor
// hairline, its title in labelColor, and the status is carried by the
// glyph alone, in the platform's system colour for it: systemGreen,
// systemOrange, systemRed, systemBlue. There is no tinted box and no
// knocked-out foreground: an in-flow box on this platform is the fill it
// stands on with a hairline around it, and the colour that says which
// status this is belongs to the mark, not to the field behind the words.
//
// Info is systemBlue and not the accent: an informational box that wore the
// accent would say whatever colour the user had chosen in System Settings,
// and under a red accent it would say "error" more loudly than an alert
// indicating Error does.
//
// The banner takes the width it is given and the height its content
// needs, clamped into the height its slot allows: an alert standing in a
// page flow is as deep as the words in it, and an alert given an exact box
// still fills that box. A title is one line; a Body of arbitrary depth
// carries the rest.
package alert

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	complayout "github.com/vibrantgio/components/layout"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Status is the status the alert indicates. The zero value is Info: an
// alert given no status is Info.
type Status int

const (
	Info Status = iota
	Success
	Warning
	Error
)

// Props configures an Alert. Title may be empty (the title row is omitted);
// Body may be nil (only the icon and title render).
type Props struct {
	Status Status
	Title  string
	Body   layout.Widget

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the alert then shapes its title with the theme's
	// shaper (Typography.Shaper()), which is built once for the process and
	// shared by every component reading that typography — the cache lives
	// behind the Typography value, so it survives the copy this component's
	// map function makes of it. Set it only when this instance must shape
	// with a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays every
	// layout.Widget out on the one goroutine that runs the event loop,
	// which is what makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// Alert returns an rx.Observable[layout.Widget] that emits a new one
// whenever any consumed theme token changes.
func Alert(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the TitleMedium text style and the
	// theme's cached shaper: the theme owns the typeface.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest4(t.Platform, t.Spacing, t.Radius, t.Typography),
			func(n rx.Tuple4[tokens.PlatformColors, tokens.SpacingScale, tokens.RadiusScale, tokens.Typography]) resolvedTokens {
				typ := n.Fourth
				return resolvedTokens{
					platform: n.First,
					spacing:  n.Second,
					radius:   n.Third,
					title:    typ.TitleMedium,
					shaper:   typ.Shaper(),
				}
			},
		)
	})
	return rx.Defer(func() rx.Observable[layout.Widget] {
		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			// Props.Shaper is an explicit override; the theme's shaper is
			// the default.
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			return func(gtx layout.Context) layout.Dimensions {
				return drawAlert(gtx, shaper, props, tok.platform, tok.spacing, tok.radius, tok.title)
			}
		})
	})
}

// Render produces a layout.Widget for an alert with pre-resolved tokens.
// Intended for golden-image testing and static demonstrations; production
// code should use Alert, which reads the shaper and the same text style
// off the theme.
//
// title is the TitleMedium role's whole text style — typeface, weight,
// size and line height all reach the shaper, exactly as they do on the
// live path. Pass tokens.DefaultTypography.TitleMedium for the default
// desktop look. There is no density parameter: an alert is a surface
// sized by its content, not a control.
func Render(
	shaper *text.Shaper,
	props Props,
	colors tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	title tokens.TextStyle,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return drawAlert(gtx, shaper, props, colors, sp, rad, title)
	}
}

type resolvedTokens struct {
	platform tokens.PlatformColors
	spacing  tokens.SpacingScale
	radius   tokens.RadiusScale
	title    tokens.TextStyle // the TitleMedium role: typeface, weight, size, line height
	shaper   *text.Shaper     // the theme's shaper; nil in the Render path
}

const iconDp = 20

func drawAlert(gtx layout.Context, shaper *text.Shaper, props Props, colors tokens.PlatformColors, sp tokens.SpacingScale, rad tokens.RadiusScale, title tokens.TextStyle) layout.Dimensions {
	r := gtx.Dp(unit.Dp(rad.Lg))

	mark := Mark(colors, props.Status)
	bg := colors.ControlBackground

	// The content is measured before the box is filled, so the fill can
	// be laid under the depth the words actually took. Recording it keeps
	// the paint order the reader sees: fill first, words over it.
	rec := op.Record(gtx.Ops)
	inner := layout.UniformInset(unit.Dp(sp.S4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
			layout.Rigid(iconWidget(iconDp, mark)),
			layout.Rigid(complayout.HSpacer(sp.S3)),
			layout.Flexed(1, contentColumn(shaper, props, colors, sp, title)),
		)
	})
	content := rec.Stop()

	size := image.Pt(gtx.Constraints.Max.X, min(max(inner.Size.Y, gtx.Constraints.Min.Y), gtx.Constraints.Max.Y))
	// The hairline is what separates the box from the page: the box stands on
	// the same fill the content does, which is the platform's answer for
	// something set in the flow rather than floating over it.
	//
	// Drawn as two fills rather than as a stroke, which is the idiom
	// components/input draws a field's edge by. A stroke is centred on its
	// path, so a one-dp line laid on the box's own edge spends half its
	// coverage outside the box: the separator, black at a tenth, reached a
	// white page as two rows near #f9f9f9 where one row of #e6e6e6 was owed,
	// and the box read as having no edge at all. Filling the box in the
	// separator and the inset box in the fill lands the hairline on whole
	// pixels at the coverage the platform recorded.
	edgePx := max(gtx.Dp(1), 1)
	rrect := clip.RRect{Rect: image.Rectangle{Max: size}, SE: r, SW: r, NE: r, NW: r}
	innerR := max(r-edgePx, 0)
	innerBox := clip.RRect{
		Rect: image.Rectangle{Min: image.Pt(edgePx, edgePx), Max: size.Sub(image.Pt(edgePx, edgePx))},
		SE:   innerR, SW: innerR, NE: innerR, NW: innerR,
	}
	paint.FillShape(gtx.Ops, colors.Separator, rrect.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, bg, innerBox.Op(gtx.Ops))
	content.Add(gtx.Ops)

	return layout.Dimensions{Size: size}
}

// iconWidget renders the status icon — a right-pointing filled chevron —
// into a fixed sizeDp square. All four statuses share this chevron shape
// and differentiate by colour only.
func iconWidget(sizeDp float32, col color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(unit.Dp(sizeDp))
		drawChevron(gtx, sz/2, sz/2, sz, col)
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	}
}

func contentColumn(shaper *text.Shaper, props Props, colors tokens.PlatformColors, sp tokens.SpacingScale, title tokens.TextStyle) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		var ws []layout.Widget
		if props.Title != "" {
			ws = append(ws, titleWidget(shaper, props.Title, colors.Label, title))
		}
		if props.Body != nil {
			if len(ws) > 0 {
				ws = append(ws, complayout.VSpacer(sp.S1))
			}
			ws = append(ws, props.Body)
		}
		if len(ws) == 0 {
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
		}
		return complayout.Col(gtx, ws...)
	}
}

func titleWidget(shaper *text.Shaper, label string, fg color.NRGBA, style tokens.TextStyle) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		mColor := op.Record(gtx.Ops)
		paint.ColorOp{Color: fg}.Add(gtx.Ops)
		material := mColor.Stop()
		// Shape with the TitleMedium role's typeface, weight, size and
		// line height. A zero weight in style falls back to SemiBold, so
		// the title keeps its emphasis against the body even when style
		// carries only a size.
		f := typeset.Font(style, font.SemiBold)
		wl := typeset.Label(style, 1)
		return typeset.Layout(gtx, shaper, wl, f, unit.Sp(style.Size), label, material)
	}
}

func drawChevron(gtx layout.Context, cx, cy, sz int, col color.NRGBA) {
	half := float32(sz) / 2
	quarter := float32(sz) / 4
	fcx := float32(cx)
	fcy := float32(cy)

	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(fcx-quarter, fcy-half))
	p.LineTo(f32.Pt(fcx+quarter, fcy))
	p.LineTo(f32.Pt(fcx-quarter, fcy+half))
	p.Close()
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}

// Mark is the colour of the alert's leading glyph: the platform's system
// colour for the status, which is the only place on an alert the status is
// carried. All four are system colours — Info included — so all four flip
// with the appearance and none of them follows the accent.
//
// It is exported because what is set beside an alert — a test measuring the
// pairing, a host drawing its own mark to match — needs the answer the alert
// drew with, and re-deriving it at the call site is how two answers appear.
func Mark(p tokens.PlatformColors, s Status) color.NRGBA {
	switch s {
	case Error:
		return p.SystemRed
	case Success:
		return p.SystemGreen
	case Warning:
		return p.SystemOrange
	default:
		return p.SystemBlue
	}
}
