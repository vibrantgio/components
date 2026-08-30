package golden

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// filled returns an w×h image painted a single colour.
func filled(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

var blue = color.RGBA{R: 100, G: 149, B: 237, A: 255}

// TestCompareMatch is the baseline: identical images compare clean.
func TestCompareMatch(t *testing.T) {
	if err := compare(filled(8, 8, blue), filled(8, 8, blue)); err != nil {
		t.Fatalf("compare of identical images: %v", err)
	}
}

// TestCompareSizeChange: a golden whose dimensions moved must compare as a
// failure, not a pass.
func TestCompareSizeChange(t *testing.T) {
	err := compare(filled(300, 60, blue), filled(44, 44, blue))
	if err == nil {
		t.Fatal("compare of differently sized images returned nil; a size change must fail")
	}
	// Both dimensions have to be named, or the failure does not say what moved.
	for _, want := range []string{"300x60", "44x44"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("compare error %q does not name %s", err, want)
		}
	}
}

// TestComparePixelChange keeps the pixel path honest alongside the size path.
func TestComparePixelChange(t *testing.T) {
	img := filled(8, 8, blue)
	img.Pix[0] = 0
	err := compare(filled(8, 8, blue), img)
	if err == nil {
		t.Fatal("compare of images differing in one pixel returned nil")
	}
	if !strings.Contains(err.Error(), "1 pixel(s) differ") {
		t.Errorf("compare error %q does not report the pixel count", err)
	}
}

// TestPixelDiffPanicsOnSizeMismatch pins the documented contract: there is no
// count to return for images of different shapes, so PixelDiff refuses rather
// than inventing a sentinel that reads as an answer.
func TestPixelDiffPanicsOnSizeMismatch(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("PixelDiff did not panic on mismatched bounds")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "equal bounds") {
			t.Errorf("panic value %v does not explain the precondition", r)
		}
	}()
	PixelDiff(filled(8, 8, blue), filled(9, 8, blue))
}
