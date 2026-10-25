package picker_test

import (
	"image"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/layout"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/attachment"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/theme/tokens"
)

// listRowFrame, listRowH, listRowFirst and listRowScroll are the window, the
// list rows' height, the first row in view and the pixels the list is
// scrolled past that row's top: row 10 is at 145..185, and its picker's
// trigger has its top edge at 145.
var listRowFrame = image.Pt(300, 200)

const (
	listRowH      = 40
	listRowFirst  = 6
	listRowScroll = 15
	listRowField  = 10
)

// TestAPickerInAScrolledListRowOpensItsMenuInsideTheWindow holds the picker
// to the attachment in a list row. The list places each row only after it
// measures it, so the field records its open menu and the list places it in
// the window's available room. The menu that stands over the trigger runs
// past the window's bottom edge, so the attachment pushes it up until its
// foot is the window's bottom edge. A press just above that edge picks the
// last option. The test holds this where the window sets the available room
// and where a layout above collects the surfaces.
func TestAPickerInAScrolledListRowOpensItsMenuInsideTheWindow(t *testing.T) {
	for _, collected := range []bool{false, true} {
		var picked []int
		field := materialize(t, picker.Field(rx.Of(liveTheme()), picker.FieldProps{
			Description:   "choose",
			Options:       options,
			AvailableRoom: attachment.AvailableRoom,
			Shaper:        defaultShaper(t),
			OnSelect:      func(_ layout.Context, i int) { picked = append(picked, i) },
		}))
		state := list.NewStateAt(listRowFirst)
		state.ScrollPixels(listRowScroll)
		items := make([]int, 20)
		for i := range items {
			items[i] = i
		}
		window := func(gtx layout.Context) layout.Dimensions {
			outer := gtx
			var s *attachment.Surfaces
			if collected {
				gtx, s = attachment.WithSurfaces(gtx)
			} else {
				gtx = attachment.WithAvailableRoom(gtx, image.Rectangle{Max: listRowFrame})
			}
			list.Layout(gtx, state, items, func(gtx layout.Context, item int) layout.Dimensions {
				size := image.Pt(gtx.Constraints.Max.X, listRowH)
				if item == listRowField {
					fgtx := gtx
					fgtx.Constraints = layout.Constraints{Max: size}
					field(fgtx)
				}
				return layout.Dimensions{Size: size}
			})
			if collected {
				attachment.PlaceSurfaces(outer, s, image.Rectangle{Max: listRowFrame})
			}
			return layout.Dimensions{Size: listRowFrame}
		}

		r := new(gioinput.Router)
		drive := driver(window, r, listRowFrame)
		drive()

		row, trig := rowHeight(tokens.Comfortable), triggerHeight(tokens.Comfortable)
		top := (listRowField-listRowFirst)*listRowH - listRowScroll
		click(r, drive, f32.Pt(100, float32(top)+float32(trig)/2))

		// The menu over the trigger would run to below the window's bottom
		// edge, so the press just above that edge is on the last option
		// only where the menu was pushed inside the window.
		if want := top + (trig-row)/2 + len(options)*row; want <= listRowFrame.Y {
			t.Fatalf("collected %v: the menu over the trigger ends at %d, inside the window's %d; the test does not reach the edge", collected, want, listRowFrame.Y)
		}
		click(r, drive, f32.Pt(100, float32(listRowFrame.Y)-float32(row)/2))
		if last := len(options) - 1; len(picked) != 1 || picked[0] != last {
			t.Errorf("collected %v: a press just above the window's bottom edge picked %v, want exactly option %d: the menu is not inside the window", collected, picked, last)
		}
	}
}
