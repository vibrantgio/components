// Package composite paints an overlay over what the frame has already drawn,
// compositing it in encoded sRGB the way the platform does.
//
// # Why an overlay needs this at all
//
// Gio renders into an sRGB render target and blends there, so a translucent
// fill it is handed is mixed in linear light: the hardware decodes the
// surface byte, mixes, and encodes the result again. The platform mixes the
// byte itself. Black at 0x33 over white lands on #e7e7e7 that way where
// macOS and the browser both land on #cccccc.
//
// Where the surface beneath an overlay is one known fill, the fix is
// [github.com/vibrantgio/theme/color.Flatten]: flatten before painting and
// hand Gio an opaque fill. An overlay over mixed content — a scrim over a
// whole page — has no one fill to flatten onto, so this package reads the
// pixels instead: it renders the frame recorded so far into an offscreen
// window, flattens the overlay over every pixel of it, and paints the
// result opaque. Painting an opaque image round-trips exactly, so the byte
// is the platform's.
//
// # What it costs, and when it is available
//
// One offscreen render of the frame so far, one readback of it, one flatten
// per pixel and one texture upload, each frame the overlay is up: 2 ms at
// 320x240 and 27 ms at 1440x900 on an Apple M1 Max, 23 of those 27 the
// readback alone, which is the only part no public Gio API can make cheaper.
// It is paid per frame, not per second — a window with a modal open draws
// only when something in it changes — but it is why an overlay declares
// itself here rather than the whole frame going through it, and why the
// overlays that use it are the ones a window shows while it waits: a scrim
// over an interrupted page.
//
// Reading the pixels beneath an overlay needs to know where the overlay
// stands in the frame, and a [layout.Context] does not carry that: Gio
// offers no way to ask for the transform in force. So the frame's owner —
// the application's event loop, or a test's renderer — declares its plane
// with [Frame], and [Flatten] composites only for an overlay that covers
// exactly that plane, which is what a full-window scrim is. Every other
// overlay, and every frame that declared no plane, falls back to a coverage
// fitted to Gio's blend ([github.com/vibrantgio/theme/color.LinearCoverage]),
// which misses the platform by at most three 255ths instead of twenty-seven.
package composite

import (
	"image"
	"image/color"
	"sync"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	vgcolor "github.com/vibrantgio/theme/color"
)

// Frame declares gtx's constraints as the plane this frame draws, so that an
// overlay covering it can be composited against what is beneath. Whatever
// lays out the root of a window calls it as its first act, where the
// constraints are the window's own plane and nothing has been offset yet; so
// does a renderer that drives Gio itself, such as a golden harness.
//
// Declaring nothing is safe: every overlay then falls back to the fitted
// coverage.
func Frame(gtx layout.Context) {
	plane.mu.Lock()
	defer plane.mu.Unlock()
	plane.ops, plane.size = gtx.Ops, gtx.Constraints.Max
}

// Flatten paints overlay over rect, in encoded sRGB.
//
// When rect is the plane [Frame] declared for gtx.Ops, it reads what the
// frame has drawn there and lands the platform's byte on every pixel. When
// it is not — a specimen tile holding a scrim of its own, a frame that
// declared no plane, an offscreen renderer the platform cannot give — it
// fills rect at the coverage fitted to Gio's blend instead. Either way the
// rect is painted.
//
// One frame is composited at a time: the offscreen window and the plane it
// was declared for are one per process, which is what a renderer is.
func Flatten(gtx layout.Context, rect image.Rectangle, overlay color.NRGBA) {
	if overlay.A == 0 || overlay.A == 0xff || rect.Empty() {
		paint.FillShape(gtx.Ops, overlay, clip.Rect(rect).Op())
		return
	}
	if flattened := flattenedFrame(gtx.Ops, rect, overlay); flattened != nil {
		iop := paint.NewImageOp(flattened)
		iop.Filter = paint.FilterNearest
		iop.Add(gtx.Ops)
		defer clip.Rect(rect).Push(gtx.Ops).Pop()
		paint.PaintOp{}.Add(gtx.Ops)
		return
	}
	fitted := overlay
	fitted.A = vgcolor.LinearCoverage(overlay.A)
	paint.FillShape(gtx.Ops, fitted, clip.Rect(rect).Op())
}

// flattenedFrame renders ops into the offscreen window, lays overlay over
// every pixel of what it drew, and returns the result — or nil when rect is
// not the declared plane, or the platform has no offscreen renderer to give.
func flattenedFrame(ops *op.Ops, rect image.Rectangle, overlay color.NRGBA) *image.RGBA {
	plane.mu.Lock()
	defer plane.mu.Unlock()
	if plane.denied || ops == nil || ops != plane.ops || rect != (image.Rectangle{Max: plane.size}) {
		return nil
	}
	if plane.offscreen == nil || plane.offscreenSize != plane.size {
		plane.release()
		w, err := headless.NewWindow(plane.size.X, plane.size.Y)
		if err != nil {
			// No offscreen renderer here, and none next frame either.
			plane.denied = true
			return nil
		}
		plane.offscreen, plane.offscreenSize = w, plane.size
		plane.shot = image.NewRGBA(image.Rectangle{Max: plane.size})
	}
	if err := plane.offscreen.Frame(ops); err != nil {
		plane.release()
		return nil
	}
	if err := plane.offscreen.Screenshot(plane.shot); err != nil {
		plane.release()
		return nil
	}
	return over(overlay, plane.shot)
}

// plane holds the declared frame and the offscreen window that renders it.
// It is package state because the window is a GPU context: one per size,
// kept between frames, rebuilt only when the plane changes shape.
var plane framePlane

type framePlane struct {
	mu            sync.Mutex
	denied        bool
	ops           *op.Ops
	size          image.Point
	offscreen     *headless.Window
	offscreenSize image.Point
	shot          *image.RGBA
}

// release drops the offscreen window, so that the next frame builds a fresh
// one. The caller holds the lock.
func (p *framePlane) release() {
	if p.offscreen != nil {
		p.offscreen.Release()
	}
	p.offscreen, p.offscreenSize, p.shot = nil, image.Point{}, nil
}

// over lays overlay over every pixel of beneath in encoded sRGB and returns
// the result premultiplied, which is how Gio reads an [image.RGBA] handed to
// [paint.NewImageOp]. beneath carries straight alpha, which is what an
// offscreen screenshot holds.
//
// On an opaque pixel this is exactly
// [github.com/vibrantgio/theme/color.Flatten]. Where the frame drew nothing
// the overlay keeps its own coverage, so a plane nothing filled comes back
// as it went in rather than turning into a black sheet.
//
// The result is a fresh image every frame: [paint.NewImageOp] hands Gio the
// pixels by reference and they are read when the frame renders, which is
// after this returns.
func over(overlay color.NRGBA, beneath *image.RGBA) *image.RGBA {
	size := beneath.Bounds().Size()
	out := image.NewRGBA(image.Rectangle{Max: size})
	// Integer arithmetic on the eight-bit channels, in 1/65025ths: a·src +
	// (1−a)·dst with a = overlay.A/255 and dst premultiplied by its own
	// alpha, rounded the way [vgcolor.Flatten] rounds. The loop runs once
	// per pixel of the plane every frame the overlay is up, so it carries no
	// floating point and no bounds check it can avoid.
	a := int(overlay.A)
	keep := 255 - a
	sr, sg, sb := a*int(overlay.R)*255, a*int(overlay.G)*255, a*int(overlay.B)*255
	const half, unit = 65025 / 2, 65025
	for y := 0; y < size.Y; y++ {
		src := beneath.Pix[y*beneath.Stride : y*beneath.Stride+size.X*4]
		dst := out.Pix[y*out.Stride : y*out.Stride+size.X*4]
		for x := 0; x+3 < len(src); x += 4 {
			k := keep * int(src[x+3])
			dst[x+0] = uint8((sr + k*int(src[x+0]) + half) / unit)
			dst[x+1] = uint8((sg + k*int(src[x+1]) + half) / unit)
			dst[x+2] = uint8((sb + k*int(src[x+2]) + half) / unit)
			dst[x+3] = uint8((a*255 + k + 127) / 255)
		}
	}
	return out
}
