package icons_test

import (
	"image"
	"image/color"
	"math"
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
	want := []icons.Name{icons.Check, icons.Chevron, icons.ChevronPair, icons.Clear, icons.Disclosure, icons.Document, icons.Folder, icons.HistoryBack, icons.HistoryForward, icons.OpenFolder, icons.Plus, icons.Refresh, icons.Search, icons.Sidebar}
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

// TestEveryMarkComesOutAtTheStrengthItsBandGives walks the set at all three
// sizes and holds each mark to the deepest value its own drawing can reach.
//
// The set draws ONE measured band of 1.4 units, which is 1.40 px at 24 dp,
// 1.17 at 20 and 0.93 at 16. Only the first of those covers a device pixel, so
// at 24 dp a mark arrives at the control's own colour wherever a band's
// leading edge stands on a whole unit or just under one, and at the two
// smaller sizes it arrives there only where two bands cross, a figure is
// solid, or a diagonal runs through a pixel's own centre. What each mark
// reaches everywhere else is READ OFF THE RENDER and recorded below with the
// arithmetic that gives it. A mark drawn differently moves its entry with it,
// which is the point: the set may not quietly lose weight.
func TestEveryMarkComesOutAtTheStrengthItsBandGives(t *testing.T) {
	// A shade this near the colour asked for is that colour: 0x14 of 0xff is
	// under a hundredth of the light a white surface gives back.
	const solid = 0x14

	// A control's mark is not a symbol filling the mark box: the chevron pair
	// and the single chevron are drawn at their MEASURED eight by eleven and
	// eight by five, which the library spends at 24 dp and nowhere else, and
	// eight units of a 24-unit grid cannot hold a whole device pixel at 16 or
	// 20. Their files say so; they are read at the size they are drawn at.
	oneSize := map[icons.Name]bool{icons.Chevron: true, icons.ChevronPair: true}

	// What each mark reaches where the band lands no whole device pixel.
	//
	// The four figures that arrive at the colour everywhere are the ones with
	// a solid junction in them: the plus's crossing, the folder's and the open
	// folder's square corners, the refresh mark's arrowhead. The document
	// reaches it at 20 and 24 from its fold's two lines, which stand on 11 and
	// 9.5; the search mark from the handle meeting the ring; the sidebar from
	// the seam at 8.75, which lands at 24 dp alone.
	//
	// The diagonals are phase, not weight. A 45-degree band covers a whole
	// device pixel only where a pixel's own centre lies on its centre line,
	// and a pixel's centre stands at a half in both axes — so an arm running
	// x+y = c reaches a whole pixel at 24 dp when c is a whole unit, at 20 dp
	// when five sixths of it is, and at 16 dp when two thirds of it is. The
	// check's arms run x-y = -8 and x+y = 26, whole at 24 dp and neither at
	// the other two. The three chevrons' arms are steeper than 45 degrees and
	// stand on no such line at all; what carries them at 24 dp is the mitre at
	// the apex, which is solid. The clear mark carries no band at all — a
	// solid disc with a cross knocked out of it — so it arrives at the colour
	// at every size, and its entry went with the bare cross it replaced.
	reaches := map[icons.Name]map[int]int{
		icons.Check:          {16: 0x80, 20: 0x5d},
		icons.Chevron:        {24: 0x2c},
		icons.ChevronPair:    {24: 0x2c},
		icons.Disclosure:     {16: 0x5f, 20: 0x58},
		icons.Document:       {16: 0x5a},
		icons.HistoryBack:    {16: 0x5f, 20: 0x55},
		icons.HistoryForward: {16: 0x5f, 20: 0x58},
		icons.Search:         {16: 0x48},
		icons.Sidebar:        {16: 0x45, 20: 0x46},
	}

	set := icons.New("darwin")
	for _, name := range set.Names() {
		mark := set.Mark(name)
		if mark == nil {
			t.Fatalf("no painter for %q", name)
		}
		sizes := []int{16, 20, 24}
		if oneSize[name] {
			sizes = []int{24}
		}
		for _, px := range sizes {
			img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
			darkest := 0xff
			for y := range px {
				for x := range px {
					if v := int(img.RGBAAt(x, y).R); v < darkest {
						darkest = v
					}
				}
			}
			want := solid
			if m, ok := reaches[name]; ok {
				if v, ok := m[px]; ok {
					want = v
				}
			}
			// One 255th of slack: the rasterizer rounds, and the readings
			// above are what it rounded to.
			if darkest > want+1 {
				t.Errorf("%q at %d px: the darkest pixel came out at %#02x, and this mark's own bands reach %#02x — it has lost weight",
					name, px, darkest, want)
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
// control's colour instead of replacing it, which is how a faint element
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

// TestSearchMarkIsDrawnWhereAFieldExpectsIt holds search.svg to the three
// numbers a search field places it by: the drawing fills the set's 20-unit
// allowance exactly, so it starts [icons.SearchDrawingOrigin] into the square
// and is [icons.SearchDrawingSize] of it, and its lens is centred on
// [icons.SearchLensCentre] rather than on the drawing's own middle.
//
// The lens's centre is read off the pixels the only way a raster offers it:
// the lens's leftmost pixel stands on the row through its centre, and its
// topmost pixel on the column through it. A drawing that moved the lens would
// move both, and a field aligning the mark on a field's centre row would put
// the glyph high or low without anything else noticing.
func TestSearchMarkIsDrawnWhereAFieldExpectsIt(t *testing.T) {
	mark := icons.New("darwin").Mark(icons.Search)
	if mark == nil {
		t.Fatal("the set carries no search mark")
	}
	for _, px := range []int{16, 20, 24} {
		img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
		drawn := drawnBounds(img)
		want := image.Rect(
			int(icons.SearchDrawingOrigin*float64(px)),
			int(icons.SearchDrawingOrigin*float64(px)),
			int((icons.SearchDrawingOrigin+icons.SearchDrawingSize)*float64(px)+0.999),
			int((icons.SearchDrawingOrigin+icons.SearchDrawingSize)*float64(px)+0.999),
		)
		if drawn != want {
			t.Errorf("at %d px the drawing covered %v and the allowance it is authored to fill is %v", px, drawn, want)
		}
		centre := icons.SearchLensCentre * float64(px)
		if row := leftmostDrawnRow(img); !within(float64(row)+0.5, centre, 1) {
			t.Errorf("at %d px the lens's leftmost pixel is on row %d, and its centre is stated at %.2f", px, row, centre)
		}
		if col := topmostDrawnColumn(img); !within(float64(col)+0.5, centre, 1) {
			t.Errorf("at %d px the lens's topmost pixel is in column %d, and its centre is stated at %.2f", px, col, centre)
		}
	}
}

// TestClearMarkIsDrawnWhereAFieldExpectsIt holds clear.svg to the two numbers
// a search field places it by: the disc stands [icons.ClearDiscOrigin] into
// the square and is [icons.ClearDiscSize] of it. The field places the mark by
// the disc and not by the square, because the clearance the platform leaves
// past it is measured to the disc's own last pixel.
//
// The knockout is read at the same time: the cross has to reach the surface
// under the mark, and a mark that filled its disc solid would still land the
// bounds above.
func TestClearMarkIsDrawnWhereAFieldExpectsIt(t *testing.T) {
	mark := icons.New("darwin").Mark(icons.Clear)
	if mark == nil {
		t.Fatal("the set carries no clear mark")
	}
	for _, px := range []int{16, 20, 24} {
		img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
		drawn := drawnBounds(img)
		want := image.Rect(
			int(icons.ClearDiscOrigin*float64(px)),
			int(icons.ClearDiscOrigin*float64(px)),
			int((icons.ClearDiscOrigin+icons.ClearDiscSize)*float64(px)+0.999),
			int((icons.ClearDiscOrigin+icons.ClearDiscSize)*float64(px)+0.999),
		)
		if drawn != want {
			t.Errorf("at %d px the disc covered %v and the allowance it is authored to fill is %v", px, drawn, want)
		}
		// The cross is knocked out of the disc and not painted over it, so
		// the middle of the mark reads the surface it stands on. At 24 dp the
		// arms are 1.18 px across and the crossing is the one place a whole
		// pixel stands inside them.
		mid := px / 2
		if px == 24 {
			if c := img.RGBAAt(mid, mid); c.R < 0xf0 {
				t.Errorf("at %d px the crossing reads %#02x, and the cross is knocked out to the surface under the mark", px, c.R)
			}
		}
		// Nothing outside the disc is drawn: the cross is a hole in the
		// figure, not a mark of its own reaching past it.
		if c := img.RGBAAt(0, mid); c.R != 0xff {
			t.Errorf("at %d px the square's leading edge reads %#02x on the mark's centre row, and the disc does not reach it", px, c.R)
		}
	}
}

// drawnBounds is the bounding box of everything the mark drew.
func drawnBounds(img *image.RGBA) image.Rectangle {
	out := image.Rectangle{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if c := img.RGBAAt(x, y); c.R == 0xff && c.G == 0xff && c.B == 0xff {
				continue
			}
			p := image.Rect(x, y, x+1, y+1)
			if out.Empty() {
				out = p
				continue
			}
			out = out.Union(p)
		}
	}
	return out
}

// leftmostDrawnRow is the row carrying the darkest pixel of the drawing's
// leading column, which on a magnifier is the lens's left extreme.
func leftmostDrawnRow(img *image.RGBA) int {
	b := img.Bounds()
	x := drawnBounds(img).Min.X
	row, darkest := b.Min.Y, 0x100
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if v := int(img.RGBAAt(x, y).R); v < darkest {
			row, darkest = y, v
		}
	}
	return row
}

// topmostDrawnColumn is leftmostDrawnRow's other axis: the column carrying the
// darkest pixel of the drawing's top row, the lens's top extreme.
func topmostDrawnColumn(img *image.RGBA) int {
	b := img.Bounds()
	y := drawnBounds(img).Min.Y
	col, darkest := b.Min.X, 0x100
	for x := b.Min.X; x < b.Max.X; x++ {
		if v := int(img.RGBAAt(x, y).R); v < darkest {
			col, darkest = x, v
		}
	}
	return col
}

// within reports whether got is no further than slack from want. got is a
// pixel's own centre, so a lens centred on a pixel boundary is within a pixel
// of either neighbour.
func within(got, want, slack float64) bool {
	d := got - want
	return d <= slack && -d <= slack
}

// sidebarPx are the sizes the library draws icons at, which is the range the
// sidebar mark's pane is read at below.
var sidebarPx = []int{16, 20, 24}

// TestSidebarMarkKeepsTheMeasuredPane reads the sidebar mark's own geometry
// off a render of it and holds it to the contract icons.SidebarPane* states.
//
// The reading is taken from the drawing that serves every platform, which is
// the pane and the seam and nothing else; the macOS drawing is held to the
// same pane by TestTheMacDrawingIsTheSamePane below, which is why the list
// lines do not have to be read around here.
//
// Coverage is linearised before it is read. Gio mixes the mark into the
// surface in linear light and the capture stores the result encoded, so the
// stored byte is not the coverage — the organization's macOS reference records
// the same rule for reading a component's own capture. A band's edges then
// come out exactly: a run's leading edge is its first covered column plus what
// that column is missing, and its trailing edge is its last plus what that one
// carries.
func TestSidebarMarkKeepsTheMeasuredPane(t *testing.T) {
	const (
		paneTrailing = icons.SidebarPaneLeading + icons.SidebarPaneWidth
		paneBottom   = icons.SidebarPaneTop + icons.SidebarPaneHeight
		// The square's own centre, which stands inside the pane on both axes:
		// below the top corners and above the bottom ones, and in the trailing
		// column clear of the seam.
		paneMiddle = 0.5
	)
	mark := icons.New("windows").Mark(icons.Sidebar)
	if mark == nil {
		t.Fatal("no painter for the sidebar")
	}
	for _, px := range sidebarPx {
		img := shoot(t, px, func(gtx layout.Context) { mark(gtx, px, black) })
		cov := coverage(img)

		// A row through the pane's middle crosses the two sides and the
		// seam and nothing else: it runs below the top corners and above
		// the bottom ones.
		rows := runs(cov.row(at(px, paneMiddle)))
		if len(rows) != 3 {
			t.Fatalf("%d px: a row through the pane crossed %d bands, want the two sides and the seam", px, len(rows))
		}
		checkBand(t, px, "the leading side", rows[0], icons.SidebarPaneLeading)
		checkBand(t, px, "the seam", rows[1], icons.SidebarSeamLeading)
		checkBand(t, px, "the trailing side", rows[2], paneTrailing-icons.SidebarBand)

		// A column through the pane's trailing half crosses the top and the
		// bottom and neither the seam nor the list lines.
		cols := runs(cov.col(at(px, paneMiddle)))
		if len(cols) != 2 {
			t.Fatalf("%d px: a column through the pane crossed %d bands, want the top and the bottom", px, len(cols))
		}
		checkBand(t, px, "the top", cols[0], icons.SidebarPaneTop)
		checkBand(t, px, "the bottom", cols[1], paneBottom-icons.SidebarBand)
	}
}

// TestTheMacDrawingIsTheSamePane: the two drawings behind the name differ
// inside the leading column and nowhere else. macOS adds the list lines that
// platform puts in a source list; the pane, the band and the seam are one
// figure on both, so a control swapping platforms swaps no geometry.
func TestTheMacDrawingIsTheSamePane(t *testing.T) {
	fallback := icons.New("windows").Mark(icons.Sidebar)
	mac := icons.New("darwin").Mark(icons.Sidebar)
	if fallback == nil || mac == nil {
		t.Fatal("no painter for the sidebar")
	}
	for _, px := range sidebarPx {
		a := shoot(t, px, func(gtx layout.Context) { fallback(gtx, px, black) })
		b := shoot(t, px, func(gtx layout.Context) { mac(gtx, px, black) })
		// The leading column is the inner pane's leading edge to the
		// seam's, and it is the only place the two are allowed to differ.
		lo := at(px, icons.SidebarPaneLeading+icons.SidebarBand)
		hi := at(px, icons.SidebarSeamLeading) + 1
		for y := range px {
			for x := range px {
				if x >= lo && x < hi {
					continue
				}
				if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
					t.Fatalf("%d px: the two drawings differ at %d,%d, outside the leading column",
						px, x, y)
				}
			}
		}
	}
}

// at converts a fraction of the mark's square to a pixel index in a square of
// px. Truncating is what puts the square's centre on the pixel that straddles
// it at every size the set is drawn at.
func at(px int, fraction float64) int { return int(fraction * float64(px)) }

// checkBand holds one run of covered pixels to a band of the set's weight
// standing at lead, both stated as fractions of the mark's square.
//
// A run of ONE covered pixel pins its width and nothing else: a band can fall
// entirely inside a single device pixel, and the share that pixel carries says
// how thick the band is without saying where inside it the band stands. Such a
// run is held to its width and to both edges lying within the pixel, which is
// all the raster offers.
//
// The band is a DEVICE width, so what is wanted here is the width the set's
// own rule gives at this size — 1.32 px below 20 dp and 1.40 from 20 up — and
// the grid's placement with half the widening spent outward on either edge.
func checkBand(t *testing.T, px int, what string, r run, lead float64) {
	t.Helper()
	// A band's edges land on eighths of a pixel at these sizes, and the
	// rasterizer's own rounding is well inside that.
	const tolerance = 0.03
	w := widen(px)
	want := struct{ lead, trail, width float64 }{
		lead:  lead*float64(px) - w/2,
		trail: (lead+icons.SidebarBand)*float64(px) + w/2,
		width: bandPx(px),
	}
	if math.Abs(r.width-want.width) > tolerance {
		t.Errorf("%d px: %s covers %.3f px, want %.3f — the band is not the set's weight",
			px, what, r.width, want.width)
	}
	if r.n == 1 {
		lo, hi := float64(r.first), float64(r.first+1)
		if want.lead < lo-tolerance || want.trail > hi+tolerance {
			t.Errorf("%d px: %s covers pixel %d alone, and the band stated at %.3f to %.3f does not fit inside it",
				px, what, r.first, want.lead, want.trail)
		}
		return
	}
	if math.Abs(r.lead-want.lead) > tolerance {
		t.Errorf("%d px: %s starts at %.3f px, want %.3f", px, what, r.lead, want.lead)
	}
	if math.Abs(r.trail-want.trail) > tolerance {
		t.Errorf("%d px: %s ends at %.3f px, want %.3f", px, what, r.trail, want.trail)
	}
}

// run is one band read off a line of coverage: where it begins and ends, in
// pixels from the line's origin, how much of a pixel it covers, and which
// pixels it lies on.
type run struct {
	lead, trail, width float64
	first, n           int
}

// runs splits a line of coverage into the bands that cover it.
func runs(line []float64) []run {
	var out []run
	for i := 0; i < len(line); i++ {
		if line[i] <= 0 {
			continue
		}
		j := i
		var sum float64
		for j < len(line) && line[j] > 0 {
			sum += line[j]
			j++
		}
		out = append(out, run{
			lead:  float64(i) + 1 - line[i],
			trail: float64(j-1) + line[j-1],
			width: sum,
			first: i,
			n:     j - i,
		})
		i = j
	}
	return out
}

// plane is one capture's coverage, a mark drawn in black on white read as the
// share of each pixel the mark covers.
type plane struct {
	v    []float64
	size int
}

func (p plane) row(y int) []float64 { return p.v[y*p.size : (y+1)*p.size] }

func (p plane) col(x int) []float64 {
	out := make([]float64, p.size)
	for y := range out {
		out[y] = p.v[y*p.size+x]
	}
	return out
}

// coverage reads a capture as coverage, linearising each channel first: the
// rasterizer mixes the mark into the surface in linear light and the capture
// stores the result encoded, so the stored byte is not the share of the pixel
// the mark took.
func coverage(img *image.RGBA) plane {
	b := img.Bounds()
	p := plane{size: b.Dx(), v: make([]float64, b.Dx()*b.Dy())}
	for y := range b.Dy() {
		for x := range b.Dx() {
			p.v[y*p.size+x] = 1 - linear(img.RGBAAt(b.Min.X+x, b.Min.Y+y).R)
		}
	}
	return p
}

// linear is the light one channel of an sRGB-encoded capture carries.
func linear(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
