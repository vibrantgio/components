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
	got, ok := icons.New("darwin").Resolve(icons.Sidebar)
	if !ok {
		t.Fatalf("%q does not resolve on darwin", icons.Sidebar)
	}
	if want := "sidebar@darwin"; got != want {
		t.Errorf("resolved to %q, want %q", got, want)
	}
}

// TestFallbackAnswersWhereThereIsNoPlatformDrawing is the other half: a system
// with no drawing of its own gets the one that serves every platform, rather
// than another system's idiom or nothing at all.
func TestFallbackAnswersWhereThereIsNoPlatformDrawing(t *testing.T) {
	for _, goos := range []string{"windows", "linux", "freebsd", "js"} {
		got, ok := icons.New(goos).Resolve(icons.Sidebar)
		if !ok {
			t.Errorf("%s: %q does not resolve", goos, icons.Sidebar)
			continue
		}
		if want := string(icons.Sidebar); got != want {
			t.Errorf("%s: resolved to %q, want %q", goos, got, want)
		}
	}
}

// TestTheSetResolvesForTheRunningSystem covers the default seam: the
// package-level entry points answer for runtime.GOOS without being told what
// that is, and without a build tag deciding it.
func TestTheSetResolvesForTheRunningSystem(t *testing.T) {
	if !icons.Has(icons.Sidebar) {
		t.Fatalf("%q does not resolve on %s", icons.Sidebar, runtime.GOOS)
	}
	if icons.Mark(icons.Sidebar) == nil {
		t.Fatalf("no painter for %q on %s", icons.Sidebar, runtime.GOOS)
	}
	got, _ := icons.Resolve(icons.Sidebar)
	want, _ := icons.New(runtime.GOOS).Resolve(icons.Sidebar)
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
	want := []icons.Name{icons.Disclosure, icons.HistoryBack, icons.HistoryForward, icons.Sidebar}
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

	for _, key := range []string{"sidebar", "sidebar@darwin"} {
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

	x, _ := a.Icon("sidebar")
	y, _ := b.Icon("sidebar")
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
	set := icons.New("darwin")
	for _, name := range set.Names() {
		mark := set.Mark(name)
		if mark == nil {
			t.Fatalf("no painter for %q", name)
		}
		first := map[int]int{}
		for pass := range 2 {
			for _, px := range []int{16, 20, 24} {
				img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
				covered := painted(img)
				if covered == 0 {
					t.Errorf("%q, %d px, pass %d: nothing was drawn", name, px, pass)
				}
				if covered == px*px {
					t.Errorf("%q, %d px, pass %d: the whole square was covered, so the drawing is not a mark", name, px, pass)
				}
				if pass == 0 {
					first[px] = covered
					continue
				}
				if covered != first[px] {
					t.Errorf("%q, %d px: replaying the built drawing covered %d pixels, and building it covered %d", name, px, covered, first[px])
				}
			}
		}
	}
}

// TestEveryMarkComesOutAtFullStrength is the set's evenness rule read off the
// pixels: whichever way a mark's edges run, its darkest pixel has to arrive at
// the colour the control asked for, or near enough that no eye separates the
// marks standing side by side.
//
// An axis-aligned band on the grid gets there for nothing — it covers device
// pixels whole. A 45 degree band does not: it crosses a pixel corner to corner
// and covers a whole one only from √2 px across, and the backend composites in
// linear light, so a band covering 91% of a pixel's area arrives at 67% of the
// colour and reads grey beside a mark that covers whole ones. That is what the
// diagonal marks carry a heavier measure for, and this is the check that
// notices if one of them stops carrying it.
func TestEveryMarkComesOutAtFullStrength(t *testing.T) {
	// A shade this near the colour asked for is that colour: 0x14 of 0xff is
	// under a hundredth of the light a white surface gives back.
	const solid = 0x14

	set := icons.New("darwin")
	for _, name := range set.Names() {
		mark := set.Mark(name)
		if mark == nil {
			t.Fatalf("no painter for %q", name)
		}
		for _, px := range []int{16, 20, 24} {
			img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
			darkest := 0xff
			for y := range px {
				for x := range px {
					if v := int(img.RGBAAt(x, y).R); v < darkest {
						darkest = v
					}
				}
			}
			if darkest > solid {
				t.Errorf("%q at %d px: the darkest pixel came out at %#02x, and a mark at the set's weight comes out at %#02x or below — the band is too thin for its direction",
					name, px, darkest, solid)
			}
		}
	}
}

// TestMarkTakesTheColourItIsGiven: colour reaches the mark from the call site,
// which is what keeps it out of the built ops.
func TestMarkTakesTheColourItIsGiven(t *testing.T) {
	mark := icons.New("windows").Mark(icons.Sidebar)
	if mark == nil {
		t.Fatal("no painter for the sidebar")
	}
	const px = 24
	red := color.NRGBA{R: 0xff, A: 0xff}
	img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, red) })

	// Every painted pixel has to be the colour asked for laid over white —
	// the two channels the colour has none of stay equal to each other and
	// never rise above the one it is made of. A curve's outermost pixels are
	// barely tinted, so how much colour a pixel carries is not the test; that
	// the colour is the right one is.
	var seen, solid int
	for y := range px {
		for x := range px {
			c := img.RGBAAt(x, y)
			if c.R == 0xff && c.G == 0xff && c.B == 0xff {
				continue
			}
			seen++
			if c.G != c.B || c.R < c.G {
				t.Fatalf("pixel at %d,%d is %v, not a shade of the colour asked for", x, y, c)
			}
			if c.G < 0x40 {
				solid++
			}
		}
	}
	if seen == 0 {
		t.Error("nothing was drawn")
	}
	if solid == 0 {
		t.Error("no pixel took the colour at full strength")
	}
}

// listLine is a point inside the sidebar's topmost list line at 24 px, where
// one grid unit is one pixel: the bar spans x 5.25 to 8.25 and y 7.5 to 9.
var listLine = image.Pt(6, 8)

// TestPlatformDrawingsDifferOnScreen: resolution is not only a key. The two
// sidebars are told apart at the pixels — the one drawn for macOS carries the
// list lines that platform puts in a source list, and the fallback carries the
// bare pane.
func TestPlatformDrawingsDifferOnScreen(t *testing.T) {
	const px = 24
	shot := func(goos string) color.RGBA {
		mark := icons.New(goos).Mark(icons.Sidebar)
		if mark == nil {
			t.Fatalf("%s: no painter for the sidebar", goos)
		}
		img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
		return img.RGBAAt(listLine.X, listLine.Y)
	}

	if c := shot("windows"); c.R != 0xff || c.G != 0xff || c.B != 0xff {
		t.Errorf("the fallback pane has something where the list lines would be: %v", c)
	}
	if c := shot("darwin"); c.R == 0xff && c.G == 0xff && c.B == 0xff {
		t.Error("the drawing for macOS is missing its list lines")
	}
}

// TestFaintElementStaysFaint: a path authored with fill-opacity modulates the
// control's colour instead of replacing it, which is how a secondary element
// survives being tinted.
func TestFaintElementStaysFaint(t *testing.T) {
	const px = 24
	mark := icons.New("darwin").Mark(icons.Sidebar)
	if mark == nil {
		t.Fatal("no painter for the sidebar")
	}
	img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })

	line := img.RGBAAt(listLine.X, listLine.Y)
	if line.R != line.G || line.G != line.B {
		t.Fatalf("the faint list line is not a shade of the colour asked for: %v", line)
	}
	if line.R < 0x50 || line.R > 0xe0 {
		t.Errorf("the faint list line came out at %d, which is neither faint nor absent", line.R)
	}
}

// TestOneMarkAtTwoSizesInOneFrame is the hazard the built ops are cached
// against: the drawing behind a name is resized in place, so two sizes in one
// frame would fight over it if they shared one.
func TestOneMarkAtTwoSizesInOneFrame(t *testing.T) {
	mark := icons.New("darwin").Mark(icons.Sidebar)
	if mark == nil {
		t.Fatal("no painter for the sidebar")
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
