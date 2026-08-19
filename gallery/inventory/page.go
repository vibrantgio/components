// The column: how the sections are stacked, banded and closed off, and the
// control that redraws the whole of it in the other scheme.
//
// A section on its own is a widget. What makes the inventory readable is the
// furniture around it — the group banner that says which module the next run
// of families comes from, the header that names each one, and the bounded
// slot each body is laid out in.
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
	"github.com/vibrantgio/theme/tokens"
)

// Items returns the whole inventory as the rows of one column, in the given
// scheme: every group banded and labelled, and a closing line under the last
// of them. The rows are ordinary widgets, so a caller can put them in a
// scrolling list, print them into one tall column, or take a slice.
//
// The rows are a function of the palette alone, so re-theming is calling this
// again with other tokens. Nothing that survives across frames is rebuilt by
// doing so.
func (inv *Inventory) Items(c tokens.ColorTokens) []layout.Widget {
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
func (inv *Inventory) ItemIndex(c tokens.ColorTokens, name string) int {
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
func (inv *Inventory) GroupItems(c tokens.ColorTokens, grp Group) []layout.Widget {
	items := make([]layout.Widget, 0, 1+2*len(grp.Sections))
	items = append(items, groupBanner(inv.shaper, c, grp.Name))
	for _, s := range grp.Sections {
		s := s
		items = append(items, sectionHeaderRow(inv.shaper, c, s.Title), sectionBody(c, s))
	}
	return items
}

// PageEnd closes the column. A column this tall that simply stops reads as a
// render that gave out; a line saying how much of the surface has just gone
// past says it ended on purpose.
func (inv *Inventory) PageEnd(c tokens.ColorTokens, sections int) layout.Widget {
	return pageEnd(inv.shaper, c, sections)
}

func pageEnd(shaper *text.Shaper, c tokens.ColorTokens, sections int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(64)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Ramps.Neutral.Step(200), clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, c.Divider, clip.Rect(image.Rect(0, 0, sz.X, 1)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 24).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper,
				fmt.Sprintf("End of the inventory — %d sections in the current theme.", sections),
				c.Ramps.Neutral.Step(600), 12, font.Font{})
		})
	}
}

// groupBanner separates one module's families from the next.
func groupBanner(shaper *text.Shaper, c tokens.ColorTokens, name string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(44)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Primary, clip.Rect{Max: sz}.Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 13).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper, name, c.OnPrimary, 15, font.Font{Weight: font.Bold})
		})
	}
}

// sectionHeaderRow labels one family. Every section carries one: an
// unlabelled swatch is a puzzle, not an inventory.
func sectionHeaderRow(shaper *text.Shaper, c tokens.ColorTokens, title string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(32)
		sz := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Ramps.Neutral.Step(200), clip.Rect{Max: sz}.Op())
		paint.FillShape(gtx.Ops, c.Divider,
			clip.Rect(image.Rect(0, h-1, sz.X, h)).Op())
		gtx.Constraints = layout.Exact(sz)
		return complayout.InsetXY(24, 8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LabelAt(gtx, shaper, title, c.Text, 13, font.Font{Weight: font.Bold})
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
func sectionBody(c tokens.ColorTokens, s Section) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := gtx.Dp(s.Height) + gtx.Dp(2*SectionPadY)
		full := image.Pt(gtx.Constraints.Max.X, h)
		paint.FillShape(gtx.Ops, c.Background, clip.Rect{Max: full}.Op())
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
	// rather than at the control, so it is sized as a target and not shrunk to
	// the glyph inside it.
	SchemeSegmentW unit.Dp = 44
	SchemeSwitchH  unit.Dp = 36
	// SchemeSwitchW is the whole control, both segments.
	SchemeSwitchW = 2 * SchemeSegmentW

	// schemeIconSize is the glyph in a segment.
	schemeIconSize unit.Dp = 20
	// schemeThumbInset is how far the current segment's fill sits inside its
	// half. The track showing all round it is what makes the pair read as one
	// control with a marker on it rather than as two buttons that touch.
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
func SchemeSwitch(c tokens.ColorTokens, dark bool) layout.Widget {
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
// happens to be drawn as two.
func SchemeSegment(c tokens.ColorTokens, dark, selected bool) layout.Widget {
	ink, ground := schemeSegmentInks(c, selected)
	glyph := schemeGlyph(dark, ink)
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
			paint.FillShape(gtx.Ops, ground, thumb.Op(gtx.Ops))
		}
		size := gtx.Dp(schemeIconSize)
		off := op.Offset(image.Pt((w-size)/2, (h-size)/2)).Push(gtx.Ops)
		gtx.Constraints = layout.Exact(image.Pt(size, size))
		glyph(gtx)
		off.Pop()
		return layout.Dimensions{Size: image.Pt(w, h)}
	}
}

// schemeTrack is the ground both segments sit on: three steps up the neutral
// ramp from the page, which is enough to read as a control against a page that
// is otherwise flat, and not so much that a control changed twice an hour
// competes with what it changes.
func schemeTrack(c tokens.ColorTokens) color.NRGBA { return c.Ramps.Neutral.Step(300) }

// schemeSegmentInks returns the glyph's colour and the ground it is read
// against, for a segment that is or is not the current one. Both come out of
// here rather than being written at the point they are painted, so what a
// contrast measurement reads is what the control draws.
//
// The current segment carries the theme's own primary pair, which is the one
// pairing in a palette guaranteed legible; the other is a quiet neutral on the
// track, dark enough to be read as a glyph and light enough not to be mistaken
// for the choice that is in force.
func schemeSegmentInks(c tokens.ColorTokens, selected bool) (ink, ground color.NRGBA) {
	if selected {
		return c.OnPrimary, c.Primary
	}
	return c.Ramps.Neutral.Step(700), schemeTrack(c)
}

// schemeGlyph returns the sun or the moon drawn in ink, from the Material set.
// The vector carries its own colours, which on the wrong ground would be a
// dark disc on a dark segment, so the ink is substituted on the way in.
//
// It is built where it is drawn rather than kept. Deciding a glyph this small
// costs a few microseconds against a frame budget of several thousand, and a
// cache of them would be shared mutable state in a package whose whole point
// is that a surface is a function of the tokens it was handed.
func schemeGlyph(dark bool, ink color.NRGBA) layout.Widget {
	data := mdicons.ImageWBSunny
	if dark {
		data = mdicons.ImageBrightness2
	}
	w, err := ivgraster.Widget(data, schemeIconSize, schemeIconSize, ivgraster.WithColors(ink))
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
// ground of nearly its own colour.
func swatchBorder(gtx layout.Context, col color.NRGBA, size image.Point, width unit.Dp) {
	w := gtx.Dp(width)
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, 0, size.X, w)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, size.Y-w, size.X, size.Y)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(0, 0, w, size.Y)).Op())
	paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(size.X-w, 0, size.X, size.Y)).Op())
}
