package attachment

import "image"

// Side is the side of its anchor a floating surface is placed on. The
// developer states the preferred side; [TakeSide] arbitrates it against the
// available room.
type Side int

const (
	Top Side = iota
	Bottom
	Left
	Right
)

// Opposite is the side across the anchor from s.
func (s Side) Opposite() Side {
	switch s {
	case Top:
		return Bottom
	case Bottom:
		return Top
	case Left:
		return Right
	default:
		return Left
	}
}

// Vertical reports whether s is above or below the anchor: the surface is
// then placed along the vertical axis and shifted along the horizontal one.
func (s Side) Vertical() bool { return s == Top || s == Bottom }

// Beside is the rectangle of size placed on side of anchor, gap away from it
// and centred on it along the other axis.
func Beside(anchor image.Rectangle, size image.Point, side Side, gap int) image.Rectangle {
	midX := (anchor.Min.X + anchor.Max.X) / 2
	midY := (anchor.Min.Y + anchor.Max.Y) / 2
	var at image.Point
	switch side {
	case Top:
		at = image.Pt(midX-size.X/2, anchor.Min.Y-gap-size.Y)
	case Bottom:
		at = image.Pt(midX-size.X/2, anchor.Max.Y+gap)
	case Left:
		at = image.Pt(anchor.Min.X-gap-size.X, midY-size.Y/2)
	default:
		at = image.Pt(anchor.Max.X+gap, midY-size.Y/2)
	}
	return image.Rectangle{Min: at, Max: at.Add(size)}
}

// SideRoom is how long a surface placed on side of anchor, gap away, may be
// along the axis it is placed on before it crosses the edge of room: the
// pixels between the anchor's edge, less the gap, and the room's edge on that
// side. It is zero where the anchor stands at or past that edge.
func SideRoom(anchor image.Rectangle, side Side, gap int, room image.Rectangle) int {
	var n int
	switch side {
	case Top:
		n = anchor.Min.Y - gap - room.Min.Y
	case Bottom:
		n = room.Max.Y - anchor.Max.Y - gap
	case Left:
		n = anchor.Min.X - gap - room.Min.X
	default:
		n = room.Max.X - anchor.Max.X - gap
	}
	return max(n, 0)
}

// TakeSide is the side a surface of size takes beside anchor inside room.
// Placement is preference plus fit: the preferred side is kept while the
// room holds the surface whole there. Where it does not and the opposite side
// does, the surface takes the opposite side. Where neither holds it whole, it
// takes the side with more room, the preferred one when the two are equal,
// and the caller has it scroll inside itself in the length that side has.
func TakeSide(anchor image.Rectangle, size image.Point, preferred Side, gap int, room image.Rectangle) Side {
	length := size.Y
	if !preferred.Vertical() {
		length = size.X
	}
	here := SideRoom(anchor, preferred, gap, room)
	if length <= here {
		return preferred
	}
	there := SideRoom(anchor, preferred.Opposite(), gap, room)
	if length <= there || there > here {
		return preferred.Opposite()
	}
	return preferred
}

// Inward is the start of a span of length beginning at lo, moved the least
// distance that brings it inside [roomLo, roomHi]. A span already inside is
// not moved, and one longer than the room is left where it is, since moving
// it against one edge only moves what is cut off to the other.
func Inward(lo, length, roomLo, roomHi int) int {
	if length > roomHi-roomLo {
		return lo
	}
	if lo+length > roomHi {
		lo = roomHi - length
	}
	return max(lo, roomLo)
}

// ShiftInward moves r along the axis side does not place it on, by the least
// distance that brings it inside room less inset at each of that axis's two
// edges, as [Inward] does. The axis side places it on is left alone: that is
// [TakeSide]'s.
func ShiftInward(r image.Rectangle, side Side, room image.Rectangle, inset int) image.Rectangle {
	if side.Vertical() {
		x := Inward(r.Min.X, r.Dx(), room.Min.X+inset, room.Max.X-inset)
		return r.Add(image.Pt(x-r.Min.X, 0))
	}
	y := Inward(r.Min.Y, r.Dy(), room.Min.Y+inset, room.Max.Y-inset)
	return r.Add(image.Pt(0, y-r.Min.Y))
}
