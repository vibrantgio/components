// The column: how the sections are stacked, banded and closed off, and the
// control that redraws the whole of it in the other scheme.
//
// A section on its own is a bare layout.Widget. What makes the inventory
// readable is the frame around it — the group banner that says which
// module the next run of families comes from, the header that names each one,
// and the bounded slot each body is laid out in.
package inventory

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	mdicons "golang.org/x/exp/shiny/materialdesign/icons"

	complayout "github.com/vibrantgio/components/layout"
	ivgraster "github.com/vibrantgio/ivg/raster/gio"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// Items returns the whole inventory as the rows of one column, in the given
// scheme: every group banded and labelled, and a closing line under the last
// of them. The rows are ordinary layout.Widget values, so a caller can put
// them in a scrolling list, print them into one tall column, or take a slice.
//
// The rows are a function of the set alone, so re-theming is calling this
// again with another set. Nothing that survives across frames is rebuilt by
// doing so.
func (inv *Inventory) Items(c tokens.PlatformColors) []layout.Widget {
	groups := inv.Groups(c)
	items := make([]layout.Widget, 0, 8*len(groups))
	total := 0
	for _, grp := range groups {
		total += len(grp.Sections)
		items = append(items, inv.GroupItems(c, grp)...)
	}
	return append(items, inv.PageEnd(c, total))
}

// ItemIndex returns the row [Items] puts the named section's heading on, or
// -1 when no section has that name. It is what a caller scrolls to: the
// column is several screens tall, and a section somebody has to find by
// dragging is a section they judge after they have lost interest.
//
// The heading rather than the body, because a body arriving with its label
// off the top of the viewport reads as a fragment of whatever was above it.
func (inv *Inventory) ItemIndex(c tokens.PlatformColors, name string) int {
	row := 0
	for _, grp := range inv.Groups(c) {
		row++ // the group's banner
		for _, s := range grp.Sections {
			if s.Name == name {
				return row
			}
			row += 2 // the section's heading and its body
		}
	}
	return -1
}

// GroupItems turns one group into the rows a column shows: a banner for the
// group, then a header and a bounded body for each section.
func (inv *Inventory) GroupItems(c tokens.PlatformColors, grp Group) []layout.Widget {
	items := make([]layout.Widget, 0, 1+2*len(grp.Sections))
	items = append(items, groupBanner(inv.shaper, c, grp.Name))
	for _, s := range grp.Sections {
		s := s
		items = append(items, sectionHeaderRow(inv.shaper, c, s.Title), sectionBody(c, s))
	}
	return items
}

// TabItems returns one named group as the rows of a surface that shows that
// group and nothing else: the group's sections, a header and a bounded body
// each, with the closing line under the last of them — and without the banner
// [GroupItems] leads with.
//
// The banner is dropped because such a surface is already labelled. The
// inventory bands its groups because in the whole column they run one after
// another and a reader has to be told where one module's families end; where
// a whole viewport is one group, the label that was clicked to reach it has
// said the word already, and a full-width band repeating it directly beneath
// says nothing new.
//
// The group is looked up by name rather than by index, so a group reordered
// upstream still lands where it is named. A name no group carries returns
// nil: that is a wiring fault, not an empty catalogue, and it is meant to be
// caught by a caller's test rather than smoothed over — a surface quietly
// showing a blank column is exactly what a fallback would hide.
func (inv *Inventory) TabItems(c tokens.PlatformColors, group string) []layout.Widget {
	for _, grp := range inv.Groups(c) {
		if grp.Name != group {
			continue
		}
		rows := inv.GroupItems(c, grp)
		return append(rows[1:], inv.PageEnd(c, len(grp.Sections)))
	}
	return nil
}

// PageEnd closes the column. A column this tall that simply stops reads as a
// render that gave out; a line saying how much of the surface has just gone
// past says it ended on purpose.
func (inv *Inventory) PageEnd(c tokens.PlatformColors, sections int) layout.Widget {
	return pageEnd(inv.shaper, c, sections)
}

func pageEnd(shaper *text.Shaper, c tokens.PlatformColors, sections int) layout.Widget {
	fill := ChromeSurface(c)
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(64)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		// The closing strip wears the chrome material, under the specimens
		// rather than raised over them, and the platform's seam closes the
		// column off above it.
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, vgcolor.Flatten(c.Separator, fill),
			clip.Rect(image.Rect(0, 0, sz.X, 1)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper,
				fmt.Sprintf("End of the inventory — %d sections in the current set.", sections),
				vgcolor.Flatten(c.SecondaryLabel, fill), 12, font.Font{})
		})
	}
}

// groupBanner separates one module's families from the next. It wears the
// platform's box fill — a small step off the plane, darker in the light
// appearance and lighter in the dark one — with the seam under it, which is
// how the platform bands a run of groups. It is the one row on the column
// carrying a fill of its own, so a reader can find where one module's
// families end without a colour the platform does not paint.
func groupBanner(shaper *text.Shaper, c tokens.PlatformColors, name string) layout.Widget {
	fill := c.CardFill
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(44)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, vgcolor.Flatten(c.Separator, fill),
			clip.Rect(image.Rect(0, h-1, sz.X, h)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 13).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper, name, vgcolor.Flatten(c.Label, fill), 15, font.Font{Weight: font.Bold})
		})
	}
}

// sectionHeaderRow labels one family. Every section carries one: an
// unlabelled swatch is a puzzle, not an inventory.
func sectionHeaderRow(shaper *text.Shaper, c tokens.PlatformColors, title string) layout.Widget {
	fill := ChromeSurface(c)
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(32)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		// A section header bands the inventory, so it wears the chrome
		// material. In the light appearance that is the content's own fill,
		// so the seam is what the row is read by; one above it as well as
		// below is what closes it off from the family over it.
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: sz}.Op())
		seam := vgcolor.Flatten(c.Separator, fill)
		paint.FillShape(gtx.Ops, seam, clip.Rect(image.Rect(0, 0, sz.X, 1)).Op())
		paint.FillShape(gtx.Ops, seam, clip.Rect(image.Rect(0, h-1, sz.X, h)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper, title, vgcolor.Flatten(c.Label, fill), 13, font.Font{Weight: font.Bold})
		})
	}
}

// SectionPadX and SectionPadY are the margin a section's body is laid out
// inside: the distance from the row's own edges to the family drawn in it.
// They are stated rather than buried because a caller that puts something of
// its own alongside a body — in the row, beside what the row shows — has to
// land on the same margin, and a number copied would drift the first time this
// one moved.
const (
	SectionPadX unit.Dp = 24
	SectionPadY unit.Dp = 20
)

// sectionBody lays a family out in a slot of its own. The height is the
// section's, not the content's: several patterns expand into whatever
// constraints they are handed, and one of those left unbounded would swallow
// the rest of the column.
func sectionBody(c tokens.PlatformColors, s Section) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(s.Height) + gtx.Dp(2*SectionPadY)
		full := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, SectionSurface(c), clip.Rect{Max: full}.Op())
		gtx.Constraints = layout.Exact(full)
		return complayout.InsetXY(float32(SectionPadX), float32(SectionPadY)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = gtx.Dp(s.Height)
			gtx.Constraints.Min.Y = 0
			gtx.Constraints.Min.X = 0
			s.Body(gtx)
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(s.Height))}
		})
	}
}

// Column stacks items top to bottom at the width it is given. It is the
// inventory with no viewport in front of it — what the column would be if it
// were printed rather than scrolled.
func Column(items []layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cs := make([]layout.FlexChild, len(items))
		for i, w := range items {
			cs[i] = layout.Rigid(w)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, cs...)
	}
}

// The light/dark control's measurements.
const (
	// SchemeSegmentW is one segment's width and SchemeSwitchH the control's
	// height. A segment is what somebody has to hit while looking at the page
	// rather than at the control, so its width is a target's and not the
	// glyph's.
	//
	// The height is not, and that is deliberate. This control is as much at
	// home in a window's top strip as on a page, and a strip is dressed at a
	// scale of its own: the platform's own title bar is a 32 px band whose
	// controls are 14 px across, and its floating toolbar draws a control row
	// 18 px tall. A track at the height of a control on a page — the desktop
	// control height, 36 dp — is taller than the whole of that band, which is
	// what makes it read as an object dropped into a strip rather than part of
	// one. So the track is cut to the smallest height that still dresses the
	// glyph — the fill inside it clears [schemeIconSize] by a point at each
	// edge — and it leaves room for a band around itself at the scale the
	// platform gives one.
	//
	// What does not shrink with it is the target — see [SchemeTargetH].
	SchemeSegmentW unit.Dp = 44
	SchemeSwitchH  unit.Dp = 28
	// SchemeSwitchW is the whole control, both segments.
	SchemeSwitchW = 2 * SchemeSegmentW

	// SchemeTargetH is how tall the press area over one segment is, which is
	// taller than the segment draws. A control cut to the scale of the strip
	// it stands in must not take the pointer target down with it, so the
	// height the track gives up is handed back as slop: [SchemeTarget] is the
	// wrapper that spends it, above and below the drawn control, leaving the
	// layout at chrome scale.
	//
	// It is tokens.MinHitTarget, the standalone-control floor — which this
	// control may take because it is standalone, with air above and below it,
	// rather than a row in a stack whose neighbour the slop would be stolen
	// from. The width needs none: a segment already draws at the floor.
	SchemeTargetH = unit.Dp(tokens.MinHitTarget)

	// schemeIconSize is the glyph in a segment. It does not follow the track
	// down: at chrome scale the same mark is dressed in less, rather than a
	// smaller mark being dressed the same.
	schemeIconSize unit.Dp = 20
	// schemeThumbInset is how far the current segment's fill sits inside its
	// half. The track showing all round it is what makes the pair read as one
	// control with a marker on it rather than as two buttons that touch.
	//
	// With the glyph fixed it is also the floor under the track: the fill is
	// the track less twice this, and a fill down to the glyph's own size would
	// have the mark bursting out of the thing that marks it.
	schemeThumbInset unit.Dp = 3
)

// SchemeSwitch draws the light/dark control: a sun and a moon side by side,
// with the scheme on screen filled. The selected segment is the state — the
// control says which side you are on rather than which side a press would take
// you to, which is what every other segmented control on a desktop says.
//
// It paints and measures only; the press belongs to whatever surface puts it
// on screen, and a surface that wants one belongs on each half rather than on
// the pair — see [SchemeSegment]. Two targets are what make the control mean
// what it looks like: pointing at the moon asks for dark, whatever is on
// screen, and pointing at the segment already filled asks for nothing.
func SchemeSwitch(c tokens.PlatformColors, dark bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(SchemeSegment(c, false, !dark)),
			layout.Rigid(SchemeSegment(c, true, dark)),
		)
	}
}

// SchemeSegment draws one half of the light/dark control on its own: the moon
// half when dark is set and the sun half otherwise, filled when selected says
// the scheme it names is the one on screen. Two of them laid out side by side
// are [SchemeSwitch] — each half rounds off only its outer corners, so the
// pair closes into one track with no seam.
//
// It is exported because the press belongs to the segment and not to the
// control: a caller puts its own click area around each half and gets a
// control that names a scheme on either side, rather than a toggle that
// happens to be drawn as two. What it puts that area over is the target
// [SchemeTarget] hands it and not the track, which is smaller.
func SchemeSegment(c tokens.PlatformColors, dark, selected bool) layout.Widget {
	foreground, fill := schemeSegmentColors(c, selected)
	glyph := schemeGlyph(dark, foreground)
	return func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(SchemeSegmentW), gtx.Dp(SchemeSwitchH)
		r := h / 2
		track := clip.RRect{Rect: image.Rect(0, 0, w, h), NW: r, SW: r}
		if dark {
			track = clip.RRect{Rect: image.Rect(0, 0, w, h), NE: r, SE: r}
		}
		paint.FillShape(gtx.Ops, schemeTrack(c), track.Op(gtx.Ops))
		if selected {
			in := gtx.Dp(schemeThumbInset)
			thumb := clip.RRect{
				Rect: image.Rect(in, in, w-in, h-in),
				NW:   (h - 2*in) / 2, SW: (h - 2*in) / 2,
				NE: (h - 2*in) / 2, SE: (h - 2*in) / 2,
			}
			paint.FillShape(gtx.Ops, fill, thumb.Op(gtx.Ops))
		}
		size := gtx.Dp(schemeIconSize)
		off := op.Offset(image.Pt((w-size)/2, (h-size)/2)).Push(gtx.Ops)
		gtx.Constraints = layout.Exact(image.Pt(size, size))
		glyph(gtx)
		off.Pop()
		return layout.Dimensions{Size: image.Pt(w, h)}
	}
}

// SchemeTarget lays seg — one segment — inside the press area it is owed,
// through lay: the caller's own event wrapper, which is anything shaped like a
// [gioui.org/widget.Clickable]'s Layout, a gesture area of the caller's own
// making included.
//
// The wrapper is handed a block [SchemeTargetH] tall with the segment drawn
// centred in it, and what comes back out is the segment's own size — so the
// row the control stands in is laid out at the track's height and the slop
// overhangs the air above and below it. That split is the whole point: the
// track is chrome scale (see [SchemeSwitchH]) and the thing a pointer has to
// land on is not.
//
// The slop is vertical only. The two halves of the control tile side by side,
// so width taken here would be taken off the other half rather than added to
// anything, and a segment is already as wide as the floor asks.
func SchemeTarget(gtx layout.Context, lay func(layout.Context, layout.Widget) layout.Dimensions, seg layout.Widget) layout.Dimensions {
	var drawn layout.Dimensions
	macro := op.Record(gtx.Ops)
	target := lay(gtx, func(gtx layout.Context) layout.Dimensions {
		inner := op.Record(gtx.Ops)
		drawn = seg(gtx)
		call := inner.Stop()
		size := drawn.Size
		size.Y = max(size.Y, gtx.Dp(SchemeTargetH))
		off := op.Offset(image.Pt(0, (size.Y-drawn.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
		return layout.Dimensions{Size: size}
	})
	call := macro.Stop()
	// Shift the whole thing back up, so the segment draws where a caller that
	// asked for no target at all would have drawn it.
	off := op.Offset(image.Pt(0, -(target.Size.Y-drawn.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return drawn
}

// schemeTrack is the fill both segments sit on: the platform's push button,
// which is what a segmented control is made of. It reads eight percent off
// the white behind it in the light appearance and lighter than the plane in
// the dark one — the platform's own separation and no more, which is all a
// control changed twice an hour is owed.
func schemeTrack(c tokens.PlatformColors) color.NRGBA { return c.PushButtonFill }

// schemeSegmentColors returns the glyph's colour and the fill it is read
// against, for a segment that is or is not the current one. Both come out of
// here rather than being written at the point they are painted, so what a
// contrast measurement reads is what the control draws.
//
// The current segment is the platform's selection: the emphasized selection
// fill under the foreground the platform pairs with it. The other is the
// platform's control text over the track — the same foreground a push
// button's own label wears, which is not mistakable for the choice in force.
func schemeSegmentColors(c tokens.PlatformColors, selected bool) (foreground, fill color.NRGBA) {
	if selected {
		return c.AlternateSelectedControlText, c.SelectedContentBackground
	}
	track := schemeTrack(c)
	return vgcolor.Flatten(c.ControlText, track), track
}

// schemeGlyph returns the sun or the moon drawn in `foreground`. The vector
// carries its own colours, which on the wrong fill would be a dark disc on a
// dark segment, so the colour is substituted on the way in.
//
// It is built where it is drawn rather than kept. Deciding a glyph this small
// costs a few microseconds against a frame budget of several thousand, and a
// cache of them would be shared mutable state in a package whose whole point
// is that a surface is a function of the tokens it was handed.
func schemeGlyph(dark bool, foreground color.NRGBA) layout.Widget {
	data := mdicons.ImageWBSunny
	if dark {
		data = mdicons.ImageBrightness2
	}
	w, err := ivgraster.Widget(data, schemeIconSize, schemeIconSize, ivgraster.WithColors(foreground))
	if err != nil {
		// A glyph that will not decode leaves a blank of the right size: the
		// control keeps its shape and its targets, which is more than a panic
		// on a paint path would leave.
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}
	}
	return w
}

// swatchBorder is the hairline a flat swatch needs to be visible against a
// surface of nearly its own colour.
func swatchBorder(gtx layout.Context, col color.NRGBA, size image.Point, width unit.Dp) {
	w := gtx.Dp(width)
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, 0, size.X, w)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, size.Y-w, size.X, size.Y)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, 0, w, size.Y)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(size.X-w, 0, size.X, size.Y)).Op())
}
