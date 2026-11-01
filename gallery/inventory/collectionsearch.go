package inventory

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/button"
	"github.com/vibrantgio/components/input"
	complayout "github.com/vibrantgio/components/layout"
	"github.com/vibrantgio/markdown"
	"github.com/vibrantgio/patterns/findbar"
	"github.com/vibrantgio/patterns/shell"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// CollectionSearchQuery is the query the search specimen's toolbar field
// holds. The list column shows only the rows that hold it, and the find bar
// opens with it.
const CollectionSearchQuery = "blue"

// CollectionSearchRows are the rows of the collection the search specimen
// filters, in the list column's order.
var CollectionSearchRows = []string{
	"Morning walk",
	"Blue hour by the river",
	"Kitchen radio",
	"Bluebird notes",
	"Train platform",
}

// CollectionSearchShown is the row the content column shows.
const CollectionSearchShown = "Bluebird notes"

// CollectionSearchText is the text of the recording the content column
// shows. It holds the query three times, and it runs past the content
// column's foot.
const CollectionSearchText = `# Bluebird notes

The first bird sang from the fence at six. The light was still blue
over the field, and the second bird answered from the hedge.

At seven the sun cleared the roofs. The song changed to short calls,
one from each side of the garden, and then stopped.

The recording ends with the kettle in the kitchen. A pale blue cup
stands on the sill, and the window is open to the morning air.

Later the wind came up from the river. The birds moved to the trees at
the far end of the garden, and the recording picked up only the leaves.

The last minute holds the door, the stairs and the street outside.`

// The search specimen's measures, in dp. The window is the three-column
// specimen's width. The list column is narrower than the platform's 320, so
// the content column keeps the room to show the find bar's parts.
const (
	collectionSearchW      unit.Dp = 760
	collectionSearchH      unit.Dp = 380
	collectionSearchListW  unit.Dp = 220
	collectionSearchFieldW unit.Dp = 200
	collectionRowH         unit.Dp = 40
	// collectionRowSideDp and collectionRowCornerDp place the selected
	// row's fill. MEASURED, voicememos-collection-search-light.png: the
	// fill stands 10 clear of the window's edge and of the seam, and a
	// circular fit to its corner gives 8.35.
	collectionRowSideDp   unit.Dp = 10
	collectionRowCornerDp unit.Dp = 8
	// collectionTrackDp and collectionTrackFootDp are the band's track and
	// the clear space under it. The band stands in for a playback band.
	collectionTrackDp     unit.Dp = 32
	collectionTrackFootDp unit.Dp = 16
)

// CollectionSearchMatches answers the rows of [CollectionSearchRows] that
// hold query, case-insensitively, in order. An empty query matches every
// row: the field filters nothing while it is empty.
func CollectionSearchMatches(query string) []string {
	var out []string
	for _, r := range CollectionSearchRows {
		if strings.Contains(strings.ToLower(r), strings.ToLower(query)) {
			out = append(out, r)
		}
	}
	return out
}

// collectionSearch draws a search over a collection in a fixed state: the
// toolbar search field holds [CollectionSearchQuery], the list column shows
// only the rows that hold it, and the find bar stands open over the content
// column with the query selected whole and every match marked. The find
// button stands at the content column's foot: in the fixed band where withBand
// is true, and over the content where it is false.
func (inv *Inventory) collectionSearch(c tokens.PlatformColors, withBand bool) layout.Widget {
	fill := c.TextBackground
	q := CollectionSearchQuery
	doc := inv.searchDoc(withBand)
	doc.Find(q, 0)
	style := markdown.FromTokens(c, inv.typography(), fill)

	field := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Dp(collectionSearchFieldW)
		gtx.Constraints.Max.X = gtx.Dp(collectionSearchFieldW)
		w := input.RenderSearch(inv.shaper, "Search", c, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.SystemFont, tokens.Comfortable,
			input.RenderState{Variant: input.Chrome, Region: input.Toolbar, Surface: fill, Text: q})
		return w(typeset.WithFill(gtx, fill))
	}
	rows := func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{}
		for _, r := range CollectionSearchMatches(q) {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(collectionRowH))
				on := fill
				if r == CollectionSearchShown {
					// The keyboard is in the find bar, so the selected
					// row takes the platform's unemphasised selection.
					on = c.UnemphasizedSelectedContentBackground
					side, rad := gtx.Dp(collectionRowSideDp), gtx.Dp(collectionRowCornerDp)
					box := image.Rect(side, 0, max(size.X-side, side), size.Y)
					paint.FillShape(gtx.Ops, on, clip.RRect{Rect: box, SE: rad, SW: rad, NE: rad, NW: rad}.Op(gtx.Ops))
				}
				lgtx := typeset.WithFill(gtx, on)
				lgtx.Constraints.Min = image.Point{}
				m := op.Record(gtx.Ops)
				d := LabelAt(lgtx, inv.shaper, r, sectionText(c), 13, font.Font{})
				call := m.Stop()
				defer op.Offset(image.Pt(gtx.Dp(threeColListInsetDp), (size.Y-d.Size.Y)/2)).Push(gtx.Ops).Pop()
				call.Add(gtx.Ops)
				return layout.Dimensions{Size: size}
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
	content := func(gtx layout.Context) layout.Dimensions {
		size := gtx.Constraints.Max
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		complayout.InsetXY(24, 16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = image.Point{}
			gtx.Constraints.Max.Y = 1 << 16
			return doc.LayoutColumn(typeset.WithFill(gtx, fill), inv.shaper, style)
		})
		return layout.Dimensions{Size: size}
	}
	var band layout.Widget
	if withBand {
		band = collectionBand(c, fill)
	}
	find := button.RenderFindButton(inv.shaper, "", c, inv.typography().SystemFont, button.RenderState{Surface: fill})
	bar := findbar.Render(inv.shaper, c, tokens.Spacing, tokens.Radius, inv.typography(), tokens.Comfortable, fill,
		input.RenderState{Focused: true, Text: q, Selection: [2]int{0, len([]rune(q))}, Count: strconv.Itoa(len(doc.Matches()))})

	noButtons := func() unit.Dp { return 0 }
	f := shell.ThreeColumn{
		PaneHidden:  true,
		ListWidth:   collectionSearchListW,
		Surface:     c.WindowBackground,
		ContentFill: fill,
		Radius:      tokens.Radius,
		Density:     tokens.Comfortable,
		ListToolbar: shell.Toolbar{
			Lead: threeColListInsetDp,
			Center: func(gtx layout.Context) layout.Dimensions {
				return LabelAt(gtx, inv.shaper, "All Recordings", sectionText(c), 13, font.Font{})
			},
		},
		List:             shell.ListColumn{Rows: rows},
		ListResize:       &shell.ListResize{Min: 160},
		Toolbar:          shell.Toolbar{Lead: 16, Trailing: []layout.Widget{field}},
		Main:             findbar.Over(bar, button.FindButtonOver(content, band, find)),
		ContentFloor:     200,
		WindowButtonsEnd: noButtons,
	}
	return func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(min(gtx.Constraints.Max.X, gtx.Dp(collectionSearchW)), gtx.Dp(collectionSearchH))
		gtx.Constraints = layout.Exact(size)
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		dims, _ := f.Layout(gtx, c)
		return dims
	}
}

// searchDoc answers the document the search specimen's content column shows,
// one for each case, so the two sections keep their own layout state.
func (inv *Inventory) searchDoc(withBand bool) *markdown.Document {
	i := 0
	if withBand {
		i = 1
	}
	if inv.searchDocs[i] == nil {
		inv.searchDocs[i] = markdown.NewDocument(markdown.Parse([]byte(CollectionSearchText)))
	}
	return inv.searchDocs[i]
}

// collectionBand is the fixed band at the foot of the search specimen's
// content column: the room the find button keeps above the track, the track,
// and the clear space under it. The band takes the column's fill, and its
// track is the separator over that fill.
func collectionBand(c tokens.PlatformColors, fill color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		room := gtx.Dp(button.FindButtonRoomDp)
		track := gtx.Dp(collectionTrackDp)
		size := image.Pt(gtx.Constraints.Max.X, room+track+gtx.Dp(collectionTrackFootDp))
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: size}.Op())
		side := gtx.Dp(24)
		r := image.Rect(side, room, max(size.X-side, side), room+track)
		rad := gtx.Dp(4)
		paint.FillShape(gtx.Ops, vgcolor.Flatten(c.Separator, fill), clip.RRect{Rect: r, SE: rad, SW: rad, NE: rad, NW: rad}.Op(gtx.Ops))
		return layout.Dimensions{Size: size}
	}
}
