package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/theme/tokens"
)

// dumpDir, when set, makes TestEverythingDump write the whole page out as
// PNG tiles instead of skipping. It is a diagnostic and never a comparison:
// the page is far taller than a stored golden should be, and looking at it
// whole is a review step, not a test.
//
//	go test . -gallery.dump=/tmp/everything
var dumpDir = flag.String("gallery.dump", "", "write the everything page to this directory as PNG tiles")

// pageWidth is the width every stored section image is measured at — wide
// enough that the shell's three columns and the pricing tiers are not
// squeezed into a shape no window would show them in.
const pageWidth = 900

// tileHeight bounds one captured image. The whole page is several times a
// screen tall and past what a headless window will allocate in one piece, so
// the dump comes out in tiles.
const tileHeight = 4000

// testInventory builds the inventory a test draws from.
//
// Two things differ from the running gallery, both so that the same bytes
// come out on any machine: the shaper resolves no system fonts, and the icon
// set is pinned to one platform rather than taken from the host. The control
// marks are per-platform by design — a sidebar mark is drawn the way its
// platform draws it — so an unpinned set would store a different image on a
// Mac than on anything else.
func testInventory(t *testing.T) *inventory {
	t.Helper()
	inv := newInventory(tokens.DefaultTypography.DeterministicShaper())
	inv.marks = icons.New("darwin")
	return inv
}

// testGallery builds the window's state without a window behind it: enough
// for the inventory pages and the sidebar, and nothing for the per-family
// pages, whose live widgets newGallery builds against a real one.
func testGallery(t *testing.T) *gallery {
	t.Helper()
	g := &gallery{shaper: tokens.DefaultTypography.DeterministicShaper()}
	g.nav = make([]widget.Clickable, len(pageNames))
	g.scrollSt = make([]*list.State, len(pageNames))
	for i := range g.scrollSt {
		g.scrollSt[i] = list.NewState()
	}
	g.inv = testInventory(t)
	return g
}

type scheme struct {
	name   string
	colors tokens.ColorTokens
}

func schemes() []scheme {
	return []scheme{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	}
}

// TestGroupGoldens stores one image per group per scheme. A group is the unit
// because the question these images answer is the everything page's own —
// is every family present, labelled, and drawn — and a family that stopped
// rendering shows up in its group's image whether or not anyone thought to
// name it in a test.
func TestGroupGoldens(t *testing.T) {
	for _, sc := range schemes() {
		sc := sc
		inv := testInventory(t)
		for _, grp := range inv.groups(sc.colors) {
			grp := grp
			name := fmt.Sprintf("%s-%s", groupSlug(grp.name), sc.name)
			t.Run(name, func(t *testing.T) {
				w := column(sectionItems(inv, sc.colors, grp))
				total := measure(w, pageWidth, 1<<20)
				// A group taller than one capture is stored in tiles
				// rather than cropped: a family that fell off the
				// bottom would be a family no image covers.
				for y, n := 0, 0; y < total.Y; y, n = y+tileHeight, n+1 {
					size := image.Pt(pageWidth, min(tileHeight, total.Y-y))
					tile := name
					if total.Y > tileHeight {
						tile = fmt.Sprintf("%s-%d", name, n)
					}
					golden.Render(t, tile, size, ground(sc.colors, shiftUp(w, y)))
				}
			})
		}
	}
}

// TestEverythingRenders lays the whole page out at once, in both schemes,
// and asserts that it comes out taller than the sum of its parts is short.
// It is the cheap standing check that the page still assembles — the stored
// images say what it looks like, this says it exists at all — and it runs
// where no GPU does.
func TestEverythingRenders(t *testing.T) {
	for _, sc := range schemes() {
		inv := testInventory(t)
		var items []layout.Widget
		groups := inv.groups(sc.colors)
		if len(groups) != 4 {
			t.Fatalf("%s: %d groups, want 4", sc.name, len(groups))
		}
		sections := 0
		for _, grp := range groups {
			if len(grp.sections) == 0 {
				t.Errorf("%s: group %q is empty", sc.name, grp.name)
			}
			sections += len(grp.sections)
			items = append(items, sectionItems(inv, sc.colors, grp)...)
		}
		if sections < 30 {
			t.Errorf("%s: %d sections, want the whole published surface", sc.name, sections)
		}
		size := measure(column(items), pageWidth, 1<<20)
		if size.Y < 4000 {
			t.Errorf("%s: page measured %d tall, want the whole inventory", sc.name, size.Y)
		}
	}
}

// TestFrameDraws lays the sidebar out with every page selected, and the whole
// window out on each inventory page, in both schemes and with no window
// behind it. It is the check that the entries added to the sidebar are
// answered everywhere they are indexed — a nav slot that outran its slice, or
// a page constant that no longer names the page it used to, is a panic here
// and not a crash on someone's first click.
//
// The per-family pages are left out because their state belongs to the live
// widgets newGallery builds against a window, which a test has none of.
func TestFrameDraws(t *testing.T) {
	g := testGallery(t)

	context := func() layout.Context {
		return layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(1100, 800)),
			Ops:         new(op.Ops),
		}
	}
	for _, dark := range []bool{false, true} {
		g.dark = dark
		for page := range pageNames {
			g.page = page
			if d := g.sidebar(context()); d.Size.X == 0 {
				t.Errorf("page %q: the sidebar drew nothing", pageNames[page])
			}
		}
		for _, page := range []int{pageEverything, pagePatterns, pageMarkdown} {
			g.page = page
			if d := g.frame(context()); d.Size.X == 0 {
				t.Errorf("page %q drew nothing", pageNames[page])
			}
		}
	}
}

// TestSectionNamesUnique guards the stored images: two sections sharing a
// name would share a file, and the second would silently overwrite the first.
func TestSectionNamesUnique(t *testing.T) {
	inv := testInventory(t)
	seen := map[string]bool{}
	for _, grp := range inv.groups(tokens.DefaultLight) {
		for _, s := range grp.sections {
			if s.name == "" {
				t.Errorf("group %q: a section has no name", grp.name)
			}
			if s.title == "" {
				t.Errorf("section %q has no title", s.name)
			}
			if seen[s.name] {
				t.Errorf("section name %q is used twice", s.name)
			}
			seen[s.name] = true
		}
	}
}

// TestEverythingDump writes the page out for a look with fresh eyes. It skips
// unless -gallery.dump names a directory.
func TestEverythingDump(t *testing.T) {
	if *dumpDir == "" {
		t.Skip("gallery: pass -gallery.dump=DIR to write the page out")
	}
	if err := os.MkdirAll(*dumpDir, 0o755); err != nil {
		t.Fatalf("gallery: %v", err)
	}
	for _, sc := range schemes() {
		g := testGallery(t)
		g.dark = sc.name == "dark"
		// The banner leads, exactly as it does in the window: a review of
		// the page has to see the switch the page is judged with, or it
		// reports the page as having no way to change scheme.
		items := []layout.Widget{g.pageBanner(sc.colors, "Everything",
			"Every published family in one column, in the current theme.")}
		sections := 0
		for _, grp := range g.inv.groups(sc.colors) {
			sections += len(grp.sections)
			items = append(items, sectionItems(g.inv, sc.colors, grp)...)
		}
		items = append(items, pageEnd(g.shaper, sc.colors, sections))
		w := column(items)
		total := measure(w, pageWidth, 1<<20)
		for y, n := 0, 0; y < total.Y; y, n = y+tileHeight, n+1 {
			h := min(tileHeight, total.Y-y)
			shifted := shiftUp(w, y)
			img := golden.Capture(t, image.Pt(pageWidth, h), ground(sc.colors, shifted))
			path := filepath.Join(*dumpDir, fmt.Sprintf("everything-%s-%d.png", sc.name, n))
			if err := golden.Save(path, img); err != nil {
				t.Fatalf("gallery: save %s: %v", path, err)
			}
			t.Logf("wrote %s", path)
		}
	}
}

// ── Test-side layout helpers ──────────────────────────────────────────────────

// ground paints the scheme's background under w, so a captured image shows
// what the page shows rather than whatever the framebuffer held.
func ground(c tokens.ColorTokens, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// shiftUp draws w with its top dy pixels above the viewport, clipped to it.
// It is how a page taller than one capture comes out in pieces.
func shiftUp(w layout.Widget, dy int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		defer op.Offset(image.Pt(0, -dy)).Push(gtx.Ops).Pop()
		gtx.Constraints.Min.Y = 0
		gtx.Constraints.Max.Y = 1 << 20
		w(gtx)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// measure lays w out off-screen at width and reports the size to capture it
// at, capped at maxHeight.
func measure(w layout.Widget, width, maxHeight int) image.Point {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(width, 1<<20)},
		Ops:         &ops,
	}
	h := w(gtx).Size.Y
	return image.Pt(width, min(max(h, 1), maxHeight))
}

func groupSlug(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}
