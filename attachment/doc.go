// Package attachment is the recipe every floating surface places itself
// with: a tooltip, a menu, a popover. It is not a component and draws
// nothing; each component that floats a surface keeps its own shape and
// calls it.
//
// It holds the two halves of placement against fit. The first is the
// available room: the rectangle a floating surface may be placed in, set by
// the window with [WithAvailableRoom] and moved, together with the
// [Surfaces] a layout collects, by every layout that places a child at an
// offset ([AvailableRoomAt], [PlaceSurfacesAt], [PlacingFlex]). The second
// is the arbitration inside that room: the developer states the preferred
// [Side], and [TakeSide] keeps it while the room holds the surface whole
// there and otherwise gives the surface the opposite side when that one
// does; [ShiftInward] and [Inward] keep the surface inside the room along
// the other axis; and [SideRoom] is the length the side it took leaves, in
// which a surface longer than that scrolls inside itself.
//
// The anchor in its signatures is the rectangle the surface is placed
// against, in the coordinates the room is in.
package attachment
