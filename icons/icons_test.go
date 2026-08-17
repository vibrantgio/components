package icons_test

import (
	"image"
	"image/color"
	"runtime"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/components/icons"
)

var (
	white = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	black = color.NRGBA{A: 0xff}
)

// TestPlatformDrawingAnswersFirst is the hit half of the per-system rule: a
// system the set carries a drawing for gets that drawing.
func TestPlatformDrawingAnswersFirst(t *testing.T) {
	got, ok := icons.New("darwin").Resolve(icons.Placeholder)
	if !ok {
		t.Fatalf("%q does not resolve on darwin", icons.Placeholder)
	}
	if want := "placeholder@darwin"; got != want {
		t.Errorf("resolved to %q, want %q", got, want)
	}
}

// TestFallbackAnswersWhereThereIsNoPlatformDrawing is the other half: a system
// with no drawing of its own gets the one that serves every platform, rather
// than another system's idiom or nothing at all.
func TestFallbackAnswersWhereThereIsNoPlatformDrawing(t *testing.T) {
	for _, goos := range []string{"windows", "linux", "freebsd", "js"} {
		got, ok := icons.New(goos).Resolve(icons.Placeholder)
		if !ok {
			t.Errorf("%s: %q does not resolve", goos, icons.Placeholder)
			continue
		}
		if want := string(icons.Placeholder); got != want {
			t.Errorf("%s: resolved to %q, want %q", goos, got, want)
		}
	}
}

// TestTheSetResolvesForTheRunningSystem covers the default seam: the
// package-level entry points answer for runtime.GOOS without being told what
// that is, and without a build tag deciding it.
func TestTheSetResolvesForTheRunningSystem(t *testing.T) {
	if !icons.Has(icons.Placeholder) {
		t.Fatalf("%q does not resolve on %s", icons.Placeholder, runtime.GOOS)
	}
	if icons.Mark(icons.Placeholder) == nil {
		t.Fatalf("no painter for %q on %s", icons.Placeholder, runtime.GOOS)
	}
	got, _ := icons.Resolve(icons.Placeholder)
	want, _ := icons.New(runtime.GOOS).Resolve(icons.Placeholder)
	if got != want {
		t.Errorf("resolved to %q, want %q", got, want)
	}
	if len(icons.Names()) == 0 {
		t.Error("the set is empty")
	}
}

// TestUnknownNameHasNoMark: a name the set does not carry yields no painter on
// any platform, which is what a control's icon slot reads as "no icon".
func TestUnknownNameHasNoMark(t *testing.T) {
	for _, goos := range []string{"darwin", "windows"} {
		s := icons.New(goos)
		if s.Has("no-such-mark") {
			t.Errorf("%s: Has reports a mark that was never drawn", goos)
		}
		if s.Mark("no-such-mark") != nil {
			t.Errorf("%s: Mark returned a painter for a mark that was never drawn", goos)
		}
	}
}

// TestNamesListEachMarkOnce: a name is one entry however many platform
// drawings stand behind it.
func TestNamesListEachMarkOnce(t *testing.T) {
	got := icons.New("darwin").Names()
	want := []icons.Name{icons.Placeholder}
	if len(got) != len(want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names = %v, want %v", got, want)
		}
	}
}

// TestRegisterFillsRegistry: the set goes into the registry the library
// already has, under the keys resolution looks up, and as SVG.
func TestRegisterFillsRegistry(t *testing.T) {
	r := icon.New()
	icons.Register(r)

	for _, key := range []string{"placeholder", "placeholder@darwin"} {
		got, ok := r.Icon(key)
		if !ok {
			t.Errorf("%q missing from the registry", key)
			continue
		}
		if got.Kind() != icon.KindSVG {
			t.Errorf("%q registered as kind %v, want KindSVG", key, got.Kind())
		}
		if got.SVG() == nil {
			t.Errorf("%q registered without a drawing", key)
		}
	}
}

// TestRegistriesDoNotShareDrawings: each registry gets its own parsed copies,
// because a drawing is resized in place while its ops are built and two owners
// resizing one drawing would fight.
func TestRegistriesDoNotShareDrawings(t *testing.T) {
	a, b := icon.New(), icon.New()
	icons.Register(a)
	icons.Register(b)

	x, _ := a.Icon("placeholder")
	y, _ := b.Icon("placeholder")
	if x.SVG() == y.SVG() {
		t.Error("two registries were handed the same drawing")
	}
}

// TestMarkRendersAtEverySizeItIsDrawnAt walks the whole path — name, platform
// resolution, registry, built ops, rendered pixels — at the sizes the library
// draws icons at. The second pass renders from ops built during the first, in
// a later frame with a different op list, which is the case the cache exists
// for.
func TestMarkRendersAtEverySizeItIsDrawnAt(t *testing.T) {
	mark := icons.New("darwin").Mark(icons.Placeholder)
	if mark == nil {
		t.Fatal("no painter for the placeholder")
	}
	first := map[int]int{}
	for pass := range 2 {
		for _, px := range []int{16, 20, 24} {
			img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
			ink := painted(img)
			if ink == 0 {
				t.Errorf("%d px, pass %d: nothing was drawn", px, pass)
			}
			if ink == px*px {
				t.Errorf("%d px, pass %d: the whole square was covered, so the drawing is not a mark", px, pass)
			}
			if pass == 0 {
				first[px] = ink
				continue
			}
			if ink != first[px] {
				t.Errorf("%d px: replaying the built drawing covered %d pixels, and building it covered %d", px, ink, first[px])
			}
		}
	}
}

// TestMarkTakesTheColourItIsGiven: colour reaches the mark from the call site,
// which is what keeps it out of the built ops.
func TestMarkTakesTheColourItIsGiven(t *testing.T) {
	mark := icons.New("windows").Mark(icons.Placeholder)
	if mark == nil {
		t.Fatal("no painter for the placeholder")
	}
	const px = 24
	red := color.NRGBA{R: 0xff, A: 0xff}
	img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, red) })

	var seen int
	for y := range px {
		for x := range px {
			c := img.RGBAAt(x, y)
			if c.R == 0xff && c.G == 0xff && c.B == 0xff {
				continue
			}
			seen++
			if c.G > 0xf0 || c.B > 0xf0 || c.R < c.G || c.R < c.B {
				t.Fatalf("pixel at %d,%d is %v, not a shade of the colour asked for", x, y, c)
			}
		}
	}
	if seen == 0 {
		t.Error("nothing was drawn")
	}
}

// TestPlatformDrawingsDifferOnScreen: resolution is not only a key. The two
// placeholders are told apart at the pixels — the one drawn for macOS carries
// a dot in its middle, the fallback is an empty box.
func TestPlatformDrawingsDifferOnScreen(t *testing.T) {
	const px = 24
	shot := func(goos string) color.RGBA {
		mark := icons.New(goos).Mark(icons.Placeholder)
		if mark == nil {
			t.Fatalf("%s: no painter for the placeholder", goos)
		}
		return shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) }).RGBAAt(px/2, px/2)
	}

	if c := shot("windows"); c.R != 0xff || c.G != 0xff || c.B != 0xff {
		t.Errorf("the fallback box has something in its middle: %v", c)
	}
	if c := shot("darwin"); c.R == 0xff && c.G == 0xff && c.B == 0xff {
		t.Error("the drawing for macOS is missing its dot")
	}
}

// TestFaintElementStaysFaint: a path authored with fill-opacity modulates the
// control's colour instead of replacing it, which is how a secondary element
// survives being tinted.
func TestFaintElementStaysFaint(t *testing.T) {
	const px = 24
	mark := icons.New("darwin").Mark(icons.Placeholder)
	if mark == nil {
		t.Fatal("no painter for the placeholder")
	}
	img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })

	dot := img.RGBAAt(px/2, px/2)
	if dot.R != dot.G || dot.G != dot.B {
		t.Fatalf("the faint dot is not a shade of the colour asked for: %v", dot)
	}
	if dot.R < 0x50 || dot.R > 0xe0 {
		t.Errorf("the faint dot came out at %d, which is neither faint nor absent", dot.R)
	}
}

// TestOneMarkAtTwoSizesInOneFrame is the hazard the built ops are cached
// against: the drawing behind a name is resized in place, so two sizes in one
// frame would fight over it if they shared one.
func TestOneMarkAtTwoSizesInOneFrame(t *testing.T) {
	mark := icons.New("darwin").Mark(icons.Placeholder)
	if mark == nil {
		t.Fatal("no painter for the placeholder")
	}
	const big, small = 24, 16
	size := image.Pt(big+small, big)
	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		paint.Fill(gtx.Ops, white)
		mark(gtx, big, black)
		off := op.Offset(image.Pt(big, 0)).Push(gtx.Ops)
		mark(gtx, small, black)
		off.Pop()
		return layout.Dimensions{Size: size}
	})

	left := painted(img.SubImage(image.Rect(0, 0, big, big)).(*image.RGBA))
	right := painted(img.SubImage(image.Rect(big, 0, big+small, small)).(*image.RGBA))
	if left == 0 {
		t.Error("the larger drawing is missing")
	}
	if right == 0 {
		t.Error("the smaller drawing is missing")
	}
	if left <= right {
		t.Errorf("the larger drawing covers %d pixels and the smaller %d, so they did not keep their own sizes", left, right)
	}
}

// shoot renders draw over white in a square of px and returns the pixels.
func shoot(t *testing.T, px int, draw func(gtx layout.Context)) *image.RGBA {
	t.Helper()
	size := image.Pt(px, px)
	return golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		paint.Fill(gtx.Ops, white)
		draw(gtx)
		return layout.Dimensions{Size: size}
	})
}

// painted counts the pixels that are no longer the white they started as.
func painted(img *image.RGBA) int {
	var n int
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if c := img.RGBAAt(x, y); c.R != 0xff || c.G != 0xff || c.B != 0xff {
				n++
			}
		}
	}
	return n
}
