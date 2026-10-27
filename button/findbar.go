package button

import (
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/pointershape"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// FindBarStepsProps configures the find bar's previous and next control.
type FindBarStepsProps struct {
	// PreviousDescription and NextDescription are the screen-reader labels
	// of the two segments.
	PreviousDescription string
	NextDescription     string

	// HasQuery, if non-nil, is asked on every frame whether there is a query
	// to step through. While it answers false both segments are disabled.
	// A nil HasQuery keeps both segments enabled.
	HasQuery func() bool

	// OnPrevious and OnNext are called when the matching segment is
	// pressed. This is the FRP callback path.
	OnPrevious func(gtx layout.Context)
	OnNext     func(gtx layout.Context)

	// PreviousMessage and NextMessage, if non-nil, are emitted as
	// mvu.MessageOp when the matching segment is pressed. This is the MVU
	// integration path.
	PreviousMessage any
	NextMessage     any

	// Surface is the opaque fill the control stands on: the find bar's own
	// fill. A switched-off segment fades toward it, and the focus ring
	// composites over it.
	Surface color.NRGBA
}

// FindBarSteps returns the find bar's previous and next control: the
// segmented control's [FindBar] variant, two segments that each act when
// pressed and keep no choice. Nothing it draws records which segment was
// pressed last. Both segments are disabled while
// [FindBarStepsProps.HasQuery] answers false.
//
// The chevrons are [icons.ChevronLeft] for previous and [icons.ChevronRight]
// for next.
func FindBarSteps(th rx.Observable[theme.Theme], props FindBarStepsProps) rx.Observable[layout.Widget] {
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest3(t.Platform, t.Radius, t.Density),
			func(n rx.Tuple3[tokens.PlatformColors, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				return resolvedTokens{platform: n.First, radius: n.Second, density: n.Third}
			},
		)
	})
	return rx.Defer(func() rx.Observable[layout.Widget] {
		prev, next := &widget.Clickable{}, &widget.Clickable{}
		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			return func(gtx layout.Context) layout.Dimensions {
				enabled := props.HasQuery == nil || props.HasQuery()
				step := func(c *widget.Clickable, fn func(layout.Context), msg any) {
					for c.Clicked(gtx) {
						if !enabled {
							continue
						}
						if fn != nil {
							fn(gtx)
						}
						if msg != nil {
							mvu.MessageOp{Message: msg}.Add(gtx.Ops)
						}
					}
				}
				step(prev, props.OnPrevious, props.PreviousMessage)
				step(next, props.OnNext, props.NextMessage)

				segment := func(c *widget.Clickable, mark icons.Name, desc string) BorderedSegment {
					return BorderedSegment{
						Icon: icons.Mark(mark),
						State: RenderState{
							Variant:  FindBar,
							Surface:  props.Surface,
							Hovered:  enabled && c.Hovered(),
							Pressed:  enabled && c.Pressed(),
							Focused:  enabled && gtx.Focused(c),
							Disabled: !enabled,
						},
						Target: func(gtx layout.Context) layout.Dimensions {
							if !enabled {
								gtx = gtx.Disabled()
							}
							return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								semantic.ClassOp(semantic.Button).Add(gtx.Ops)
								semantic.DescriptionOp(desc).Add(gtx.Ops)
								semantic.EnabledOp(enabled).Add(gtx.Ops)
								if enabled {
									pointershape.OverSize(gtx.Ops, gtx.Constraints.Min, pointer.CursorPointer)
								}
								return layout.Dimensions{Size: gtx.Constraints.Min}
							})
						},
					}
				}
				return BorderedSegments(gtx, nil, tok.platform, tok.radius, tokens.TextStyle{}, tok.density, []BorderedSegment{
					segment(prev, icons.ChevronLeft, props.PreviousDescription),
					segment(next, icons.ChevronRight, props.NextDescription),
				})
			}
		})
	})
}
