// Package icon is a name-to-icon registry that holds icons in either SVG
// (github.com/vibrantgio/svg) or IVG (github.com/vibrantgio/ivg) form behind
// one Icon type, so a call site can resolve an icon by name without knowing
// which format it was authored in.
//
// Reach for it when an application has an icon set it wants to put in a
// registry at start-up and look up from anywhere. A Registry ships empty and nothing in
// the organization populates one yet, so the icons in it are the ones you put
// there.
//
// Icon is a discriminated union: check Kind before calling SVG or IVG, because
// the accessor for the other format returns the zero value rather than
// reporting an error. Registry is a plain map with no locking, so register
// during start-up and read from the frame goroutine afterwards. And it stores
// icon data, not painters — button.Props.Icon wants a clip.Path painter with
// the signature func(gtx, sizePx, col), so rasterise through ivg/raster/gio or
// svg/driver/gio yourself.
package icon
