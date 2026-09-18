package main

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/gallery/inventory"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/icons"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// The icon sheet's own measurements. Every number here belongs to the sheet
// rather than to a mark: the set says what a mark is drawn like, the sheet
// says how the set is laid out to be read.
const (
	iconSheetW            = 420 // the capture's width in px, at 1 px per dp
	iconSheetPadX unit.Dp = 16  // air the sheet holds left and right
	iconSheetPadY unit.Dp = 12  // air the sheet holds above and below
	iconNameW     unit.Dp = 120 // the column a mark's name is set in
	iconCellW     unit.Dp = 56  // the column one size is drawn in
	iconRowH      unit.Dp = 32  // the height every row is drawn at
	iconSizes             = 3   // how many sizes a row shows
	iconGOOS      string  = "darwin"
)

// iconPixels are the sizes every mark is shown at: the three a control in
// this library asks for. A mark is drawn to a grid and lands differently at
// each of them, so one specimen cannot answer for the set.
var iconPixels = []unit.Dp{16, 20, 24}

// TestIconSheetGolden stores one image per appearance of every mark the icon
// set carries, at every size a control draws one at, named.
//
// It is the image the set is reviewed against: a mark whose keyline fell off
// the grid, or whose weight reads heavier than its neighbours, shows up here
// beside the rest of the set and nowhere else. The set is resolved for macOS
// rather than for whatever system runs the test, so the stored image is the
// same everywhere and a mark with a drawing of its own for this platform is
// the drawing the sheet shows.
func TestIconSheetGolden(t *testing.T) {
	for _, sc := range schemes() {
		sheet := iconSheet(t, sc.colors)
		size := measure(sheet, iconSheetW, 1<<20)
		golden.Render(t, "icons-"+sc.name, size, onBackground(sc.colors, sheet))
	}
}

// TestTheIconSheetShowsEveryMark holds the sheet to the set: every name the
// set carries has a row. A mark added to the set and left out of the sheet is
// a mark no review is ever handed.
func TestTheIconSheetShowsEveryMark(t *testing.T) {
	set := icons.New(iconGOOS)
	names := set.Names()
	if len(names) == 0 {
		t.Fatal("the icon set carries no marks")
	}
	for _, n := range names {
		if set.Mark(n) == nil {
			t.Errorf("the set lists %q and answers no drawing for it", n)
		}
	}
}

// iconSheet builds the sheet: one row per mark, its name in a column of its
// own and the mark beside it at each size, so a reader compares one mark
// across three sizes along a row and one size across the set down a column.
func iconSheet(t *testing.T, c tokens.PlatformColors) layout.Widget {
	t.Helper()
	shaper := tokens.DefaultTypography.DeterministicShaper()
	set := icons.New(iconGOOS)
	names := set.Names()
	rows := make([]layout.Widget, 0, len(names)+1)
	rows = append(rows, inset(iconSheetPadX, iconSheetPadY, iconHeader(c, shaper)))
	for _, name := range names {
		rows = append(rows, inset(iconSheetPadX, 0, iconRow(c, shaper, set, name)))
	}
	rows = append(rows, inset(iconSheetPadX, iconSheetPadY, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(0, 0)}
	}))
	return inventory.Column(rows)
}

// iconHeader names the columns, since three marks of different sizes side by
// side do not say which size each is.
func iconHeader(c tokens.PlatformColors, shaper *text.Shaper) layout.Widget {
	muted := vgcolor.Flatten(c.SecondaryLabel, c.WindowBackground)
	return func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{layout.Rigid(iconCaption(shaper, "The mark", muted, iconNameW))}
		for _, px := range iconPixels {
			cs = append(cs, layout.Rigid(iconCaption(shaper, dpLabel(px), muted, iconCellW)))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
}

// iconRow is one mark: its name, then the mark itself at each size, each
// centred in a column of the same width so the sizes are read against each
// other rather than against the ragged edge of a name.
func iconRow(c tokens.PlatformColors, shaper *text.Shaper, set *icons.Set, name icons.Name) layout.Widget {
	label := vgcolor.Flatten(c.Label, c.WindowBackground)
	muted := vgcolor.Flatten(c.SecondaryLabel, c.WindowBackground)
	mark := set.Mark(name)
	return func(gtx layout.Context) layout.Dimensions {
		cs := []layout.FlexChild{layout.Rigid(iconCaption(shaper, string(name), muted, iconNameW))}
		for _, px := range iconPixels {
			cs = append(cs, layout.Rigid(iconCell(mark, label, px)))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, cs...)
	}
}

// iconCell draws one mark at one size, centred in a cell of the row's height.
func iconCell(mark icons.Painter, col stdcolor.NRGBA, px unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		cell := image.Pt(gtx.Dp(iconCellW), gtx.Dp(iconRowH))
		if mark != nil {
			size := gtx.Dp(px)
			off := op.Offset(image.Pt((cell.X-size)/2, (cell.Y-size)/2)).Push(gtx.Ops)
			mark(gtx, size, col)
			off.Pop()
		}
		return layout.Dimensions{Size: cell}
	}
}

// iconCaption sets one label in a column of its own width.
func iconCaption(shaper *text.Shaper, s string, col stdcolor.NRGBA, w unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Dp(w)
		gtx.Constraints.Max.X = gtx.Dp(w)
		return inventory.LabelAt(gtx, shaper, s, col, 11, font.Font{})
	}
}

// dpLabel names a size the way the library states one.
func dpLabel(px unit.Dp) string {
	return itoa(int(px)) + " dp"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
