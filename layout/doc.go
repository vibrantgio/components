// Package layout holds the small layout helpers components' components and the
// applications above them share: Inset and InsetXY, HSpacer and VSpacer, Row
// and Col over Gio's Flex, Pill for a rounded-rectangle clip, and FocusGroup.
//
// Reach for it to keep spacing written as plain dp numbers instead of a
// unit.Dp conversion at every call site, and for the common case where every
// child of a row or column is rigid. Drop to gioui.org/layout directly as soon
// as you need mixed Rigid and Flexed children: Row and Col do not express
// them.
//
// FocusGroup tracks keyboard focus across a fixed set of interactive items.
// Allocate one per logical group, Grow it to the item count, call Update once
// per frame before the items lay out, and register each item's tag inside its
// own clip area; Focused reports -1 when nothing in the group holds focus.
//
// One thing to know before importing: this package is named layout and so
// collides with gioui.org/layout, whose Context, Widget and Dimensions its own
// signatures use. Applications conventionally import one of the two under an
// alias; the workbench launcher example imports this one as pllayout.
package layout
