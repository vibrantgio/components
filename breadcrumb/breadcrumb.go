// Package breadcrumb provides the breadcrumb control: going back up the
// hierarchy — each earlier place a link to its place, the last where you
// are. It draws a horizontal row of titles parted by the icon set's chevron.
//
// It is drawn as the platform's path bar: every place in the label colour,
// the last of them plain text, and the chevron between two places in the
// secondary label colour. No capture in the reference holds a path bar, so
// those two are the nearest measured readings — Finder's own window at 1x,
// where the sidebar's row titles plateau at #191919 over the pane's #fafaf9
// and its section headings at #7b7b7b, which is the secondary label over
// that fill to within a byte.
//
// The pointer over it stays the arrow. A trail stands on chrome, where the
// platform's path bar leaves the arrow over every place it holds; the
// pointing hand belongs to links in the content.
//
// Breadcrumb is a callable Go function consuming a components theme
// observable, returning a stream of layout.Widget. Source is intentionally
// short and free of opaque configuration — copy it into your own app and
// modify as needed.
//
// Colour says nothing about which segments answer a press: the path bar
// sets every place in the one colour, and which place you are at is told by
// position. A segment is clickable exactly when its own OnClick is non-nil.
// The conventional trail — every segment but the last carrying an OnClick —
// is the caller's to build, and nothing here enforces it.
//
// There is no overflow behaviour. Every Item renders, in one horizontal
// row, each title clamped to a single line; a trail deeper than its
// constraint is clipped rather than collapsed to a leading ellipsis. An
// empty Items renders to zero Dimensions.
//
// A trail comes in two shapes. Breadcrumb and Render take their Items when
// they are built, which is what a trail known up front wants and what fixes
// each segment's clickable to a position. Trail and NewTrail take their
// Segments per frame instead, for a trail that is decided again on every
// frame as the user navigates; those route a click by the segment's own Key
// rather than by where it stood. Both shapes draw the same row.
package breadcrumb

import (
	"image"
	"image/color"

	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/internal/surface"
	complayout "github.com/vibrantgio/components/layout"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// Item is one segment in the breadcrumb trail. OnClick may be nil, in which
// case the segment is treated as a non-interactive current location.
// Conventionally the last item carries OnClick == nil; the package does not
// enforce this — interactivity follows the OnClick field per item.
type Item struct {
	Title   string
	OnClick func(gtx layout.Context)
}

// Props configures a Breadcrumb. Items must contain at least one entry;
// an empty slice renders to zero-sized Dimensions.
type Props struct {
	Items []Item

	// Chevron is the square each separator is drawn in. Zero takes
	// DefaultChevron, the square a mark standing beside text takes.
	//
	// It is stated rather than derived because the separator is drawn and
	// not typeset: nothing in the title's text style says how large a mark
	// beside it should be, and a trail set in a smaller style than the
	// desktop SystemFont — or one whose separator should read as a hairline
	// between titles rather than a mark of its own — wants a smaller square
	// than the default. The chevron is drawn into the square and covers 8 of
	// its 24 units across and 14 down, so this is the square the mark stands
	// in and not the drawn figure's own extent.
	Chevron unit.Dp

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the breadcrumb then shapes its titles with the
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

	// Surface is the opaque fill the trail stands on. It draws no fill of
	// its own, so its titles and separators are drawn straight on this, and
	// the platform's label and secondary label carry a coverage rather than
	// a colour. The zero value — no colour — is the window's own plane.
	Surface color.NRGBA
}

// Breadcrumb returns an rx.Observable[layout.Widget] that emits a new one
// whenever any consumed theme token changes. Click handlers fire
// for any item whose OnClick is non-nil; mirror the components/button
// interaction model (widget.Clickable + semantic ops) per segment.
//
// Props.Items is read when the stream is built, so this is the trail that is
// known up front. A trail that changes as the user navigates wants Trail
// instead, which takes its segments per frame.
func Breadcrumb(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	resolved := resolveTokens(th)

	return rx.Defer(func() rx.Observable[layout.Widget] {
		clicks := make([]*widget.Clickable, len(props.Items))
		for i := range clicks {
			clicks[i] = new(widget.Clickable)
		}

		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			// Props.Shaper is an explicit override; the theme's shaper is
			// the default.
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			return func(gtx layout.Context) layout.Dimensions {
				for i := range props.Items {
					if props.Items[i].OnClick != nil && clicks[i].Clicked(gtx) {
						props.Items[i].OnClick(gtx)
					}
				}
				return drawBreadcrumb(gtx, shaper, props.Items, clicks, tok.platform, props.Surface, tok.spacing, tok.title, props.Chevron)
			}
		})
	})
}

// Render produces a layout.Widget for a breadcrumb with pre-resolved
// tokens. Intended for golden-image testing and static demonstrations;
// production code should use Breadcrumb, which reads the shaper and the
// same text style off the theme.
//
// title is the SystemFont role's whole text style — typeface, weight,
// size and line height all reach the shaper, exactly as they do on the
// live path. Pass tokens.DefaultTypography.SystemFont for the default
// desktop look. There is no density parameter: a breadcrumb trail is a
// text row, not a control.
//
// sp is unread: the row's one gap is [separatorGapDp], stated rather than
// taken off the scale. It stays in the signature, which is the shape the
// library's static render paths share.
func Render(
	shaper *text.Shaper,
	props Props,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	title tokens.TextStyle,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return drawBreadcrumb(gtx, shaper, props.Items, nil, p, props.Surface, sp, title, props.Chevron)
	}
}

type resolvedTokens struct {
	platform tokens.PlatformColors
	spacing  tokens.SpacingScale
	title    tokens.TextStyle // the SystemFont role: typeface, weight, size, line height
	shaper   *text.Shaper     // the theme's shaper; nil in the Render path
}

// resolveTokens flattens the nested theme observables into a stream of
// concrete snapshots, one per token change. The typography emission supplies
// both the SystemFont text style and the theme's cached shaper, since the
// theme owns the typeface. Both live paths read the same snapshot.
func resolveTokens(th rx.Observable[theme.Theme]) rx.Observable[resolvedTokens] {
	return rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest3(t.Platform, t.Spacing, t.Typography),
			func(n rx.Tuple3[tokens.PlatformColors, tokens.SpacingScale, tokens.Typography]) resolvedTokens {
				typ := n.Third
				return resolvedTokens{
					platform: n.First,
					spacing:  n.Second,
					title:    typ.SystemFont,
					shaper:   typ.Shaper(),
				}
			},
		)
	})
}

// DefaultChevron is the square a separator is drawn in when the caller
// states no size of its own: the size a mark standing beside text is drawn
// at everywhere else in this library — components/icon's own size at the
// comfortable density, and the box patterns/sidebar gives this same chevron.
// No capture in the reference holds the platform's path bar, so the square
// its separator stands in is unmeasured and this is the nearest reading the
// reference carries. The figure covers 6.7 by 11.7 px of it.
const DefaultChevron unit.Dp = 20

// chevronSize answers which square this row's separators take: the caller's
// where it stated one, and the default otherwise. A zero or negative size is
// no size at all, not an invisible separator.
func chevronSize(d unit.Dp) unit.Dp {
	if d <= 0 {
		return DefaultChevron
	}
	return d
}

// separatorGapDp is the clear room the row leaves between a place's title
// and the separator's drawn figure, on each side of it.
//
// It is a stated number: no capture in the reference holds the platform's
// path bar, so the room that bar leaves around its separator is unmeasured,
// and this is the eight the platform leaves beside a mark in its sidebar —
// the nearest reading the reference carries — standing in until a path bar
// capture exists. It is stated rather than taken as a step of the spacing
// scale because what it measures from is the drawn figure, which the square
// the mark stands in says nothing about.
const separatorGapDp unit.Dp = 8

// separatorSpacer is the gap the row actually lays out beside the
// separator's square: the clear room above, less the room that square
// already leaves around the figure drawn into it. The chevron covers 8 of
// the mark box's 24 units across, so a square of c leaves c/3 clear on each
// side of the figure and the row owes only the rest. A square wide enough to
// leave the whole clearance by itself takes no gap at all.
func separatorSpacer(chevron unit.Dp) unit.Dp {
	if gap := separatorGapDp - chevron/3; gap > 0 {
		return gap
	}
	return 0
}

// drawBreadcrumb lays the row out: the places parted by the separator and
// its two gaps. It spends no step of the spacing scale — the row's one gap
// is [separatorGapDp], measured from the figure the separator draws rather
// than from the square it stands in — so the scale reaches it and is unread.
func drawBreadcrumb(
	gtx layout.Context,
	shaper *text.Shaper,
	items []Item,
	clicks []*widget.Clickable,
	p tokens.PlatformColors,
	standsOn color.NRGBA,
	_ tokens.SpacingScale,
	style tokens.TextStyle,
	chevron unit.Dp,
) layout.Dimensions {
	if len(items) == 0 {
		return layout.Dimensions{}
	}

	// The trail draws no fill, so every foreground it draws is flattened
	// onto the surface it stands on.
	standsOn = surface.Or(standsOn, p.WindowBackground)
	chevronCol := vgcolor.Flatten(p.SecondaryLabel, standsOn)

	fg := titleColor(p, standsOn)

	square := chevronSize(chevron)
	gap := float32(separatorSpacer(square))

	children := make([]layout.FlexChild, 0, 2*len(items)-1)
	for i, item := range items {
		if i > 0 {
			children = append(children,
				layout.Rigid(complayout.HSpacer(gap)),
				layout.Rigid(chevronWidget(square, chevronCol)),
				layout.Rigid(complayout.HSpacer(gap)),
			)
		}
		children = append(children, layout.Rigid(segmentWidget(shaper, item, clickFor(clicks, i), fg, style)))
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
}

// titleColor returns the foreground every segment's title is drawn in: the
// platform's label, flattened onto the surface the row stands on. It takes
// no index because the path bar sets every place in the one colour — where
// you are is told by standing last and by answering nothing, never by a
// second colour. The separator between two places is the secondary label,
// the row's one mark that is neither a place nor set in type.
func titleColor(p tokens.PlatformColors, standsOn color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.Label, standsOn)
}

// clickFor returns the clickable drawing segment i, or nil when the caller
// laid out no clickable for it — the static Render path passes none at all,
// and a frame-time trail passes none for an inert segment.
func clickFor(clicks []*widget.Clickable, i int) *widget.Clickable {
	if i >= len(clicks) {
		return nil
	}
	return clicks[i]
}

func segmentWidget(shaper *text.Shaper, item Item, click *widget.Clickable, fg color.NRGBA, style tokens.TextStyle) layout.Widget {
	title := titleWidget(shaper, item.Title, fg, style)
	if click == nil || item.OnClick == nil {
		return title
	}
	return func(gtx layout.Context) layout.Dimensions {
		return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			semantic.LabelOp(item.Title).Add(gtx.Ops)
			semantic.EnabledOp(true).Add(gtx.Ops)
			// No shape is declared over a place: the trail stands on
			// chrome, and the platform's path bar keeps the arrow there.
			return title(gtx)
		})
	}
}

func titleWidget(shaper *text.Shaper, title string, fg color.NRGBA, style tokens.TextStyle) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return typeset.Text(gtx, shaper, title, style, fg, 1)
	}
}

// chevronWidget draws one separator into a square of sizeDp: the set's
// chevron pointing the way the row reads, parent to child, in col. The
// drawing is the set's chevron-right mark — the one right-pointing chevron
// the set carries, measured off Finder's own toolbar chevrons at 8 by 14 px
// in the 24-unit mark box. A set that cannot resolve the name leaves the
// square empty rather than drawing a figure of this package's own, so the
// row's measure does not move with the set.
func chevronWidget(sizeDp unit.Dp, col color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(sizeDp)
		if mark := icons.Mark(icons.ChevronRight); mark != nil {
			mark(gtx, sz, col)
		}
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	}
}
