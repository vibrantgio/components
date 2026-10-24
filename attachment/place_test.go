package attachment_test

import (
	"image"
	"testing"

	"github.com/vibrantgio/components/attachment"
)

// TestTakeSide places a 40×30 surface beside an anchor in a 200×200 room.
func TestTakeSide(t *testing.T) {
	room := image.Rect(0, 0, 200, 200)
	size := image.Pt(40, 30)
	cases := []struct {
		name      string
		anchor    image.Rectangle
		preferred attachment.Side
		want      attachment.Side
	}{
		{"room below keeps Bottom", image.Rect(80, 20, 120, 40), attachment.Bottom, attachment.Bottom},
		{"no room below takes Top", image.Rect(80, 170, 120, 190), attachment.Bottom, attachment.Top},
		{"no room above takes Bottom", image.Rect(80, 10, 120, 30), attachment.Top, attachment.Bottom},
		{"no room after takes Left", image.Rect(170, 80, 190, 120), attachment.Right, attachment.Left},
		{"no room before takes Right", image.Rect(10, 80, 30, 120), attachment.Left, attachment.Right},
	}
	for _, c := range cases {
		if got := attachment.TakeSide(c.anchor, size, c.preferred, 4, room); got != c.want {
			t.Errorf("%s: took %v, want %v", c.name, got, c.want)
		}
	}

	// A surface taller than both sides takes the side with more room, and
	// the preferred one when the two are equal.
	tall := image.Pt(40, 150)
	if got := attachment.TakeSide(image.Rect(80, 60, 120, 80), tall, attachment.Top, 4, room); got != attachment.Bottom {
		t.Errorf("taller than both sides, more room below: took %v, want Bottom", got)
	}
	if got := attachment.TakeSide(image.Rect(80, 90, 120, 110), tall, attachment.Top, 4, room); got != attachment.Top {
		t.Errorf("taller than both equal sides: took %v, want Top", got)
	}
	if got := attachment.SideRoom(image.Rect(80, 60, 120, 80), attachment.Bottom, 4, room); got != 116 {
		t.Errorf("SideRoom below = %d, want 116", got)
	}
}

// TestShiftInward moves a surface across the axis it is not placed on, and
// leaves one longer than the room where it is.
func TestShiftInward(t *testing.T) {
	room := image.Rect(0, 0, 200, 200)
	r := attachment.Beside(image.Rect(0, 100, 20, 120), image.Pt(60, 30), attachment.Bottom, 4)
	if r.Min.X >= 0 {
		t.Fatalf("Beside centred on an anchor at the edge starts at %d, want left of 0", r.Min.X)
	}
	if got := attachment.ShiftInward(r, attachment.Bottom, room, 4); got.Min.X != 4 || got.Min.Y != r.Min.Y {
		t.Errorf("ShiftInward = %v, want Min.X 4 and Y unchanged", got)
	}
	r = attachment.Beside(image.Rect(180, 190, 200, 200), image.Pt(30, 60), attachment.Left, 4)
	if got := attachment.ShiftInward(r, attachment.Left, room, 0); got.Max.Y != 200 || got.Min.X != r.Min.X {
		t.Errorf("ShiftInward = %v, want Max.Y 200 and X unchanged", got)
	}
	if got := attachment.Inward(-10, 300, 0, 200); got != -10 {
		t.Errorf("Inward of a span longer than the room moved it to %d", got)
	}
}
