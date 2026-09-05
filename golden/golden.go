// Package golden provides a golden-image test harness for Gio components.
//
// It is exported rather than internal on purpose: it owns the one headless-Gio
// capture path, so a golden test anywhere imports it rather than inlining a
// copy.
//
// # Usage
//
//	func TestMyWidget(t *testing.T) {
//	    golden.Render(t, "my-widget", image.Pt(200, 200), func(gtx layout.Context) layout.Dimensions {
//	        paint.FillShape(gtx.Ops, color.NRGBA{R: 255, A: 255},
//	            clip.Rect{Max: gtx.Constraints.Max}.Op())
//	        return layout.Dimensions{Size: gtx.Constraints.Max}
//	    })
//	}
//
// [Render] is [Capture] followed by [Compare]. Split them when the image comes
// from somewhere other than a headless window ([CompareNRGBA] takes a CPU-drawn
// one), or when a test diffs two live captures against each other rather than
// against a stored file ([PixelDiff]).
//
// # File layout
//
// Golden images live in testdata/golden/<name>.png relative to the calling
// test's package directory (the directory go test uses as the working directory).
//
// That directory is shared by every test file in the package, so names must be
// unique across the whole package and not merely within one test. Prefix them
// with the component: a directory holding several components will otherwise
// have two of them collide on a state name.
//
// # Updating goldens
//
// Name the packages explicitly and put the flag AFTER them:
//
//	go test ./button ./golden ./icon ./input ./layout ./list ./richtext ./scrollbar -golden.update
//
// Both halves of that line matter. go test cannot tell that an unfamiliar flag
// is boolean, so -golden.update placed before the packages swallows them and
// only the package in the current directory is tested. And ./... cannot stand
// in for the list: a module has test packages that store no goldens, and a
// test binary rejects a flag it never declared.
//
// The flag is declared here, exactly once, and reaches every importing package
// through the linked test binary. A package that imports this one must not also
// declare a -golden.update of its own: two registrations of one flag name in a
// single binary is a panic in flag.Bool at init, before any test runs.
//
// # CI gate
//
// If a golden file does not exist and -golden.update is NOT set, the test
// fails with a message directing the developer to run -golden.update. This
// prevents silently passing tests with no stored baseline.
package golden

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

var update = flag.Bool("golden.update", false, "overwrite golden images with current output")

// Render renders draw into a headless window of size and diffs the result
// against testdata/golden/<name>.png.
//
// If -golden.update is set, the stored image is written (or overwritten) and
// the test passes. Otherwise the stored golden must exist; if it is absent the
// test fails with instructions to run -golden.update.
//
// On a mismatch — a changed size or a changed pixel — the test fails and saves
// the actual output alongside the golden as testdata/golden/<name>.actual.png
// for side-by-side inspection.
func Render(t *testing.T, name string, size image.Point, draw layout.Widget) {
	t.Helper()
	Compare(t, name, Capture(t, size, draw))
}

// Compare diffs an already-rendered image against testdata/golden/<name>.png,
// with the same update, missing-file and mismatch behaviour as [Render].
//
// It is the half of [Render] that does not need a GPU: a test that composes its
// image some other way — several captures blended, a CPU rasterisation, a frame
// pulled out of an interaction sequence — calls this directly.
func Compare(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name+".png")

	if *update {
		if err := saveImage(path, img); err != nil {
			t.Fatalf("golden: save %s: %v", path, err)
		}
		return
	}

	stored, err := loadImage(path)
	if os.IsNotExist(err) {
		t.Fatalf("golden: %s not found; run go test -golden.update to create", path)
		return
	}
	if err != nil {
		t.Fatalf("golden: load %s: %v", path, err)
		return
	}

	if err := compare(stored, img); err != nil {
		actualPath := strings.TrimSuffix(path, ".png") + ".actual.png"
		_ = saveImage(actualPath, img)
		t.Fatalf("golden: %q: %v (actual saved to %s)", name, err, actualPath)
	}
}

// CompareNRGBA is [Compare] for an image drawn on the CPU, where the natural
// type is *image.NRGBA rather than the *image.RGBA headless.Screenshot fills.
//
// The two are the same bytes here and the conversion is free: Screenshot writes
// straight-alpha (non-premultiplied) samples into its *image.RGBA, which is
// exactly what an *image.NRGBA holds, and saveImage re-labels them the other
// way round before encoding. A golden stored by either path is the same file.
func CompareNRGBA(t *testing.T, name string, img *image.NRGBA) {
	t.Helper()
	Compare(t, name, &image.RGBA{Pix: img.Pix, Stride: img.Stride, Rect: img.Rect})
}

// compare reports how img fails to match stored, or nil if it matches.
//
// The size check comes first and is a failure in its own right: once the
// bounds differ there is no pixel count to report. It is a separate function
// from Render so that both failure conditions can be tested without a
// *testing.T that has to actually fail.
func compare(stored, img *image.RGBA) error {
	if sb, ib := stored.Bounds(), img.Bounds(); sb != ib {
		return fmt.Errorf("size changed: golden is %dx%d, render is %dx%d",
			sb.Dx(), sb.Dy(), ib.Dx(), ib.Dy())
	}
	if n := PixelDiff(stored, img); n > 0 {
		return fmt.Errorf("%d pixel(s) differ", n)
	}
	return nil
}

// Capture renders draw into a headless window of size and returns the RGBA
// pixel data. The test is skipped if headless rendering is not available on
// the current platform, so it never returns nil.
//
// The metric is pinned at one pixel per dp and per sp. Every conversion in
// gioui.org/unit reads the zero unit.Metric as exactly that, but code that
// reads PxPerDp straight out of the struct does not: gio's widget.Image
// multiplies its transform by PxPerDp with no zero guard, so an image laid out
// under the zero metric collapses to nothing.
func Capture(t *testing.T, size image.Point, draw layout.Widget) *image.RGBA {
	t.Helper()
	w, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		t.Skipf("golden: headless rendering not supported: %v", err)
	}
	defer w.Release()

	var ops op.Ops
	gtx := layout.Context{
		Constraints: layout.Exact(size),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Ops:         &ops,
	}
	draw(gtx)

	if err := w.Frame(&ops); err != nil {
		t.Fatalf("golden: Frame: %v", err)
	}

	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := w.Screenshot(img); err != nil {
		t.Fatalf("golden: Screenshot: %v", err)
	}
	return img
}

// PixelDiff counts the number of pixels that differ between a and b, which
// must have equal bounds. It panics if they do not.
//
// The panic is deliberate rather than a sentinel count. "How many pixels
// differ" has no answer for two images of different shapes, and callers test
// the count with some variant of n > 0 or n == 0, which a sentinel answers
// wrongly in one direction or the other with no way to tell it apart from a
// real count.
//
// So the one caller for which a size change is a real outcome rather than a
// bug — the stored-golden comparison, where an image is allowed to have
// changed shape since it was recorded — compares Bounds itself first and
// reports the change on its own terms. Every other caller diffs two images it
// captured in the same test at the same requested size, where differing bounds
// is a defect in the test.
func PixelDiff(a, b *image.RGBA) int {
	if a.Bounds() != b.Bounds() {
		panic(fmt.Sprintf("golden: PixelDiff: images must have equal bounds, got %v and %v",
			a.Bounds(), b.Bounds()))
	}
	bounds := a.Bounds()
	n := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			aOff := (y-bounds.Min.Y)*a.Stride + (x-bounds.Min.X)*4
			bOff := (y-bounds.Min.Y)*b.Stride + (x-bounds.Min.X)*4
			if a.Pix[aOff] != b.Pix[bOff] ||
				a.Pix[aOff+1] != b.Pix[bOff+1] ||
				a.Pix[aOff+2] != b.Pix[bOff+2] ||
				a.Pix[aOff+3] != b.Pix[bOff+3] {
				n++
			}
		}
	}
	return n
}

// Save writes img to path as a PNG, creating the containing directory if it is
// missing. It is the same encoder [Compare] stores goldens with, exported for
// tests that write an image without diffing it, so that a diagnostic PNG and a
// stored golden are byte-for-byte the same file.
func Save(path string, img *image.RGBA) error { return saveImage(path, img) }

func saveImage(path string, img *image.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// headless.Screenshot fills *image.RGBA with straight-alpha (non-premultiplied)
	// pixel values. Wrapping as *image.NRGBA before encoding tells png.Encode to
	// store the bytes as-is, avoiding a premultiplication pass that would corrupt
	// the stored values for anti-aliased (partially-transparent) edge pixels.
	nrgba := &image.NRGBA{Pix: img.Pix, Stride: img.Stride, Rect: img.Rect}
	return png.Encode(f, nrgba)
}

// loadImage reads a PNG from path and returns it as *image.RGBA.
// The raw pixel bytes are reinterpreted directly, so straight-alpha data
// written by saveImage round-trips without any alpha conversion.
func loadImage(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	switch v := decoded.(type) {
	case *image.RGBA:
		return v, nil
	case *image.NRGBA:
		// png.Decode returns NRGBA for 8-bit RGBA PNGs.
		// The raw Pix bytes are identical to RGBA layout, so we
		// reinterpret in-place without any alpha conversion.
		return &image.RGBA{Pix: v.Pix, Stride: v.Stride, Rect: v.Rect}, nil
	default:
		bounds := decoded.Bounds()
		rgba := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, decoded.At(x, y))
			}
		}
		return rgba, nil
	}
}
