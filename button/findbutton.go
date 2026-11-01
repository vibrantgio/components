package button

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/internal/controlface"
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/components/pointershape"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// FindButtonTitle is the find button's title, the platform's own.
const FindButtonTitle = "Search"

// The find button's measures, in points.
//
// MEASURED, voicememos-collection-search-light.png, a 1x capture: the button
// is 24 tall and a capsule, its corner a circular fit of 12.36 to the leading
// end. Thirteen clear columns lead the magnifier and thirteen trail the
// title, and four stand between the two. The library's font sets "Search"
// narrower than the platform's, so the button is as wide as these parts
// and its own title, and not the measured 84. The magnifier covers 13 × 13, the
// toolbar field's own magnifier, which the search field draws in an 18 pt
// square. Its first covered row is 6 below the button's top edge, so the
// middle of what it covers is the button's centre row. The title's baseline is 17 below the button's top edge.
const (
	findButtonHeightDp   unit.Dp = 24
	findButtonSideDp     unit.Dp = 13
	findButtonGapDp      unit.Dp = 4
	findButtonMarkDp     unit.Dp = 18
	findButtonMarkTopDp  unit.Dp = 6
	findButtonBaselineDp unit.Dp = 17
)

// FindButtonProps configures a FindButton instance.
type FindButtonProps struct {
	// Title is the button's title. Empty takes [FindButtonTitle].
	Title string

	// Description is the screen-reader label. Empty takes the title.
	Description string

	// Surface answers the opaque fill the button stands over: the
	// content's fill. The focus ring composites over it.
	Surface func(tokens.PlatformColors) color.NRGBA

	// OnClick is called when the button is pressed. This is the FRP path.
	OnClick func(gtx layout.Context)

	// Message, if non-nil, is emitted as mvu.MessageOp when the button is
	// pressed. This is the MVU path.
	Message any

	// Clickable, if non-nil, is used instead of an internal one, as
	// [Props.Clickable] is.
	Clickable *widget.Clickable

	// Shaper overrides the theme's shaper, as [Props.Shaper] does.
	Shaper *text.Shaper
}

// FindButton returns the find button: the bordered button with the looking
// glass and a title that opens the find bar. It stands over the content and
// casts the toolbar control's drop shadow on it. Where it stands and when is
// the caller's.
//
// MEASURED, voicememos-collection-search-light.png: the shadow under the
// button matches the toolbar control's to within 1/255 at every row below,
// beside and above it.
func FindButton(th rx.Observable[theme.Theme], props FindButtonProps) rx.Observable[layout.Widget] {
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest2(t.Platform, t.Typography),
			func(n rx.Tuple2[tokens.PlatformColors, tokens.Typography]) resolvedTokens {
				return resolvedTokens{platform: n.First, title: n.Second.SystemFont, shaper: n.Second.Shaper()}
			},
		)
	})
	return rx.Defer(func() rx.Observable[layout.Widget] {
		var own widget.Clickable
		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			title := findButtonTitle(props.Title)
			desc := props.Description
			if desc == "" {
				desc = title
			}
			return func(gtx layout.Context) layout.Dimensions {
				click := props.Clickable
				if click == nil {
					click = &own
				}
				for click.Clicked(gtx) {
					if props.OnClick != nil {
						props.OnClick(gtx)
					}
					if props.Message != nil {
						mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
					}
				}
				var standsOn color.NRGBA
				if props.Surface != nil {
					standsOn = props.Surface(tok.platform)
				}
				s := RenderState{
					Surface: standsOn,
					Hovered: click.Hovered(),
					Pressed: click.Pressed(),
					Focused: gtx.Focused(click),
				}
				body := func(gtx layout.Context) layout.Dimensions {
					return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						semantic.ClassOp(semantic.Button).Add(gtx.Ops)
						semantic.LabelOp(title).Add(gtx.Ops)
						semantic.DescriptionOp(desc).Add(gtx.Ops)
						semantic.EnabledOp(true).Add(gtx.Ops)
						return drawFindButton(gtx, shaper, title, tok, s)
					})
				}
				return controlface.Cast(gtx, controlface.Shadow(tok.platform), s.Focused, body)
			}
		})
	})
}

// RenderFindButton draws the find button in an explicit visual state,
// without event processing or rx machinery, with its shadow. style is the
// SystemFont role. It is for golden-image tests and static renders;
// production code uses [FindButton].
func RenderFindButton(shaper *text.Shaper, title string, p tokens.PlatformColors, style tokens.TextStyle, s RenderState) layout.Widget {
	tok := resolvedTokens{platform: p, title: style}
	title = findButtonTitle(title)
	return func(gtx layout.Context) layout.Dimensions {
		return controlface.Cast(gtx, controlface.Shadow(p), s.Focused, func(gtx layout.Context) layout.Dimensions {
			return drawFindButton(gtx, shaper, title, tok, s)
		})
	}
}

func findButtonTitle(t string) string {
	if t == "" {
		return FindButtonTitle
	}
	return t
}

// findButtonFill is the find button's fill in an interaction state: its own
// fill at rest, and the platform's hover and press coverages over it, as the
// toolbar control takes them. The reference holds no capture of the find
// button under the pointer or held down.
func findButtonFill(p tokens.PlatformColors, s RenderState) color.NRGBA {
	switch controlState(s) {
	case tokens.StatePressed:
		return vgcolor.Flatten(p.PressOverlay, p.FindButtonFill)
	case tokens.StateHover:
		return vgcolor.Flatten(p.HoverOverlay, p.FindButtonFill)
	}
	return p.FindButtonFill
}

// drawFindButton draws the find button's face: the capsule with its edge,
// the looking glass and the title. It is as wide as its parts and the
// measured clear space around them. It performs no event queries.
func drawFindButton(gtx layout.Context, shaper *text.Shaper, title string, tok resolvedTokens, s RenderState) layout.Dimensions {
	p := tok.platform
	fill := findButtonFill(p, s)
	fg := p.FindButtonForeground

	titleGtx := typeset.WithFill(gtx, fill)
	titleGtx.Constraints.Min = image.Point{}
	titleGtx.Constraints.Max.X = titleRoom
	mTitle := op.Record(gtx.Ops)
	titleDims := typeset.Text(titleGtx, shaper, title, tok.title, fg, 1)
	titleCall := mTitle.Stop()

	slot := gtx.Dp(findButtonMarkDp)
	side := gtx.Dp(findButtonSideDp)
	markW := int(icons.MagnifyingGlassDrawingSizePx(gtx, slot) + 0.5)
	titleX := side + markW + gtx.Dp(findButtonGapDp)
	h := gtx.Dp(findButtonHeightDp)
	size := image.Pt(titleX+titleDims.Size.X+side, h)

	controlface.EdgedCapsule(gtx, image.Rectangle{Max: size}, p, fill, p.FindButtonRim, controlface.State{
		Focused:  s.Focused,
		StandsOn: surface.Or(s.Surface, p.WindowBackground),
	})

	if g := icons.Mark(icons.MagnifyingGlass); g != nil {
		// The square is placed off the drawing inside it, at a fraction of
		// a pixel, so that its first pixel stands 13 in and 6 down.
		origin := icons.MagnifyingGlassDrawingOriginPx(gtx, slot)
		x := float32(side) - origin
		y := float32(gtx.Dp(findButtonMarkTopDp)) - origin
		st := op.Affine(f32.Affine2D{}.Offset(f32.Pt(x, y))).Push(gtx.Ops)
		g(gtx, slot, fg)
		st.Pop()
	}

	y := gtx.Dp(findButtonBaselineDp) - (titleDims.Size.Y - titleDims.Baseline)
	off := op.Offset(image.Pt(titleX, y)).Push(gtx.Ops)
	titleCall.Add(gtx.Ops)
	off.Pop()

	pointershape.OverSize(gtx.Ops, size, pointer.CursorPointer)
	return layout.Dimensions{Size: size}
}
