package attachment_test

import (
	"image"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/attachment"
)

// TestAvailableRoomUnsetIsUnbounded reads the value a caller reads where no
// window set the room: a rectangle wide enough to place any annotation as no
// available room does; and the room a window set, moved into a child.
func TestAvailableRoomUnsetIsUnbounded(t *testing.T) {
	gtx := layout.Context{}
	r := attachment.AvailableRoom(gtx)
	if !image.Rect(-1<<19, -1<<19, 1<<19, 1<<19).In(r) {
		t.Errorf("AvailableRoom with nothing set = %v, want an unbounded rectangle", r)
	}
	if got := attachment.AvailableRoom(attachment.AvailableRoomAt(gtx, image.Pt(10, 10))); got != r {
		t.Errorf("AvailableRoomAt with nothing set moved the room to %v", got)
	}
	set := attachment.WithAvailableRoom(gtx, image.Rect(0, 0, 100, 50))
	if got := attachment.AvailableRoom(attachment.AvailableRoomAt(set, image.Pt(10, 20))); got != image.Rect(-10, -20, 90, 30) {
		t.Errorf("AvailableRoomAt moved the room to %v, want (-10,-20)-(90,30)", got)
	}
	if got := attachment.AvailableRoom(gtx); got != r {
		t.Errorf("WithAvailableRoom wrote into the caller's context: %v", got)
	}
	if attachment.HasAvailableRoom(gtx) || !attachment.HasAvailableRoom(set) {
		t.Error("HasAvailableRoom does not say whether a window set the room")
	}
	if cgtx, _ := attachment.WithSurfaces(set); attachment.HasAvailableRoom(cgtx) {
		t.Error("WithSurfaces kept the available room on a child whose offset is not yet known")
	}
}
