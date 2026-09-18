package icons

import (
	"embed"
	"fmt"
	"image/color"
	"math"
	"runtime"
	"slices"
	"strings"
	"sync"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/svg"
	"github.com/vibrantgio/svg/parser"
)

// marksFS holds the set's source. Every mark ships in every binary; which one
// a name resolves to is decided at run time, not by build tag.
//
//go:embed marks/*.svg
var marksFS embed.FS

// markDir is the directory inside marksFS the marks live in.
const markDir = "marks"

// platformSep joins a mark's name to the operating system it is drawn for,
// both in a registry key and in Resolve's answer. A name never contains it —
// names are lowercase words joined by "-" — so the two halves stay separable.
const platformSep = "@"

// Name identifies a mark by the control it belongs to.
type Name string

// The marks the set carries. Each names a control, not a picture: what the
// drawing behind one looks like is free to change and does change between
// platforms, and these are the strings call sites store.
const (
	// Sidebar is the control that shows and hides a window's sidebar. One
	// mark serves both directions — which way the control is about to go is
	// the control's to signal, and a mark that morphs cannot be recognised.
	Sidebar Name = "sidebar"

	// Disclosure is the control that opens and closes a row's children. The
	// mark is drawn as the row stands closed; a control showing an open row
	// turns it a quarter turn rather than swapping in a second drawing.
	Disclosure Name = "disclosure"

	// HistoryBack is the control that returns to what was shown before.
	HistoryBack Name = "history-back"

	// HistoryForward is the control that goes back to what the reader had
	// stepped away from.
	HistoryForward Name = "history-forward"

	// Search is the looking glass a search field leads with. It names the
	// control at a glance and is the one mark in this set that operates
	// nothing: it stands in a structure rather than on a control of its own.
	Search Name = "search"

	// Clear is the control that empties a field of what was typed into it.
	Clear Name = "clear"

	// Plus is the control that adds one more of what the window holds — a
	// new note, a new chat. One mark serves every such control: what is
	// added is the control's label to say, not the mark's.
	Plus Name = "plus"

	// Refresh is the control that reads again what the window is showing,
	// from wherever it came from. One mark serves every such control: what
	// is read again is the control's label to say.
	Refresh Name = "refresh"

	// OpenFolder is the control that opens a folder — chooses which folder
	// the window is showing.
	OpenFolder Name = "open-folder"

	// Folder is the thing an entry stands for when it holds other entries.
	// It names a thing rather than a control, which is what tells it from
	// OpenFolder: that one is drawn on the control that chooses which folder
	// a window is showing, and this one on the row that is a folder.
	Folder Name = "folder"

	// Document is the thing an entry stands for when it is one piece of
	// content — a note, a file, a page. It names a thing, as Folder does.
	Document Name = "document"

	// Check is the mark that says this is the one: it stands beside the
	// option a control is currently holding, such as the current item of an
	// open menu. It names a state rather than a control, and it says which
	// one is held rather than which one the pointer is over — that is the
	// fill's to say.
	Check Name = "check"

	// ChevronPair is the mark on a control that holds one of several values:
	// two chevrons stacked point to point, the upper pointing up and the
	// lower down. It says the value can move either way and never which way
	// the menu opens, which is why the platform draws a pair here.
	ChevronPair Name = "chevron-pair"

	// Chevron is the mark on a control that opens a menu of actions: one
	// chevron pointing down, the pair's lower half. It says the menu goes
	// down from here, which is the one thing the pair cannot say.
	Chevron Name = "chevron"
)

// The search mark's drawing, as fractions of the square it is drawn in. A
// control placing the looking glass by the platform's own numbers needs two
// things the square does not tell it: where the drawing starts inside the
// square, and where the lens's centre falls in it.
//
// They are stated rather than measured off the parsed file because they are
// that file's contract. search.svg draws its lens to the set's measured round
// and curved allowance of 13 units, which with the handle puts the drawing
// from unit 3.9 to unit 20.095 on both axes, and carries its lens centred on
// unit 10.4, which is not the drawing's own centre and cannot be: the lens
// stands leading and high and the handle runs to the lower trailing corner.
// TestSearchMarkIsDrawnWhereAFieldExpectsIt holds the drawing to all three.
const (
	// SearchDrawingOrigin is the square's leading and top edge to the
	// drawing's first pixel.
	SearchDrawingOrigin = 3.9 / 24.0
	// SearchDrawingSize is the side of the drawing inside the square.
	SearchDrawingSize = 16.195 / 24.0
	// SearchLensCentre is the square's leading and top edge to the centre of
	// the lens, which is the point a field aligns on its centre row. The
	// band's widening is spent outward on both edges of every band, so the
	// lens's centre stands where the grid puts it at every size.
	SearchLensCentre = 10.4 / 24.0
)

// SearchDrawingOriginPx is the square's leading and top edge to the drawing's
// first pixel, in device pixels, for a mark drawn at sizePx in this window.
//
// It is not SearchDrawingOrigin scaled. Below 24 dp the band is drawn at the
// device width its own size's capture gives, and half of that widening is
// spent outward on every edge, so the drawing reaches that much further
// leading and higher than the grid puts it. A control placing the looking
// glass by the platform's own measured inset places the drawing's first pixel
// and so has to ask for this.
func SearchDrawingOriginPx(gtx layout.Context, sizePx int) float32 {
	w := float64(widening(gtx.Metric.PxPerDp, sizePx)) / 1000
	return float32(SearchDrawingOrigin*float64(sizePx) - w/2)
}

// SearchDrawingSizePx is the side of the drawing inside the square, in device
// pixels, for a mark drawn at sizePx in this window: SearchDrawingSize scaled
// and the band's whole widening, half of it spent past each edge. A control
// holding the platform's measured clear space past the glyph measures it from
// the last pixel the glyph covers, which is this.
func SearchDrawingSizePx(gtx layout.Context, sizePx int) float32 {
	w := float64(widening(gtx.Metric.PxPerDp, sizePx)) / 1000
	return float32(SearchDrawingSize*float64(sizePx) + w)
}

// The sidebar mark's pane, as fractions of the square it is drawn in. The
// mark is one figure on both platforms the set draws it for — macOS adds the
// list lines inside the leading column and moves nothing — so the pane, the
// band and the seam below are the contract for every drawing behind the
// name.
//
// They are stated rather than measured off the parsed file because they are
// that file's contract, and because they are what a reading off a rendered
// mark is checked against: the two mark files carry the capture the numbers
// come from and where each band's whole device pixel lands.
// TestSidebarMarkKeepsTheMeasuredPane holds both drawings to all five.
const (
	// SidebarPaneLeading is the square's leading edge to the pane's, which is
	// the set's square keyline.
	SidebarPaneLeading = 2.5 / 24.0
	// SidebarPaneTop is the square's top edge to the pane's.
	SidebarPaneTop = 4.5 / 24.0
	// SidebarPaneWidth and SidebarPaneHeight are the pane's outer extent: a
	// landscape rectangle in the proportion a window with a sidebar has, the
	// width drawn to the keyline and the height to the capture's own 15.12.
	SidebarPaneWidth  = 19.0 / 24.0
	SidebarPaneHeight = 15.0 / 24.0
	// SidebarBand is the weight every edge of the mark is drawn at on the
	// grid, which is the set's band at 24 dp. Below that the band is drawn at
	// the device width the size's own capture gives and comes out wider than
	// this in units — the grid places the band's centreline and the device
	// width sets how far it reaches either side of it.
	SidebarBand = 1.4 / 24.0
	// SidebarSeamLeading is the square's leading edge to the seam's, which
	// leaves the leading column just under half the width of the trailing
	// one: the measured 4.99 against 10.21, one to 2.05 over an inner pane
	// of 16.2.
	SidebarSeamLeading = 8.75 / 24.0
	// SidebarListBand is the weight the list lines inside the leading column
	// are drawn at, the set's second band, and SidebarListPeriod the
	// distance between two of their centres. Both are measured off the same
	// toggle; only the macOS drawing carries them.
	SidebarListBand   = 0.93 / 24.0
	SidebarListPeriod = 2.20 / 24.0
)

// The set's grid, and the band drawn on it. The grid is 24 units square and
// one unit is one device pixel at 24 dp. The BAND is not a unit width: it is
// a DEVICE width, read off the platform's own symbol standing at the size the
// mark is drawn at, and the units it comes to therefore move with the size.
//
// MEASURED at 1x off the organization's macOS reference, each symbol read
// against its own drawn plateau rather than against a colour name, band
// against covered extent:
//
//	a symbol 12.33 px across   1.32 px   the search field's magnifier, in
//	                                     mail-window.png (x 878-890) and
//	                                     voicememos-window.png (x 657-669)
//	a symbol 16 px across      1.40 px   Mail's compose (mail-window.png,
//	                                     x 415-430)
//	a symbol 19.31 px across   1.39 px   Voice Memos' sidebar toggle
//	                                     (voicememos-window.png, x 106-125)
//
// The platform holds one weight across a symbol four times another's size —
// its narrowest reading stands at 0.85 of its widest over a range the unit
// band would have spanned by a factor of four — so the band is drawn as the
// device width each size's own capture gives and not as a share of the mark.
const (
	// gridUnits is the side of the square every mark is drawn on.
	gridUnits = 24.0
	// bandUnitsAt24 is the band at 24 dp, where one unit is one device pixel
	// and the device width and the unit width are the same number. It is the
	// figure every mark's file states its own placements in.
	bandUnitsAt24 = 1.4
	// bandPxFrom20 is the device width the band takes from 20 dp up, where a
	// square mark drawn to the set's 19-unit keyline covers 15.83 px and more
	// — the size Mail's compose and Voice Memos' sidebar toggle stand at. At
	// 24 dp it is bandUnitsAt24 to the hundredth, so the two meet there.
	bandPxFrom20 = 1.40
	// bandPxBelow20 is the device width the band takes below 20 dp, where a
	// square mark covers under 15.83 px — the size the search field's
	// magnifier stands at. Nothing is captured between 12.33 px and 16, so
	// the set steps at 20 dp rather than running a line between the two.
	bandPxBelow20 = 1.32
)

// bandUnits is the band's width on the grid for a mark drawn at dp points:
// the device width its own size's capture gives, turned back into units. At
// 24 dp and above it is the grid's own 1.4 — above 24 dp no symbol is
// captured, and the unit band is what the set falls back to, which meets the
// device rule exactly at 24.
func bandUnits(dp float64) float64 {
	switch {
	case !(dp > 0) || dp >= gridUnits:
		return bandUnitsAt24
	case dp >= 20:
		return bandPxFrom20 * gridUnits / dp
	default:
		return bandPxBelow20 * gridUnits / dp
	}
}

// widening is how much wider than the grid's 1.4 units a mark's band is drawn
// at this size, in thousandths of a device pixel. pxPerDp is the window's own
// scale: the band is a width in device pixels at 1x and scales with the
// window like everything else the mark draws, so the size in POINTS is what
// picks the reading and the size in PIXELS is what it is spent at.
func widening(pxPerDp float32, px int) int {
	scale := float64(pxPerDp)
	if !(scale > 0) {
		scale = 1
	}
	extra := (bandUnits(float64(px)/scale) - bandUnitsAt24) * float64(px) / gridUnits
	if extra <= 0 {
		return 0
	}
	return int(math.Round(extra * 1000))
}

// Painter draws a mark into a square of sizePx at the current origin, in col.
// It is the shape this library's controls take for an icon slot.
type Painter func(gtx layout.Context, sizePx int, col color.NRGBA)

// Set is the icon set as one operating system sees it: every mark is present,
// and a name resolves to that system's drawing where the set carries one and
// to the drawing that serves every platform where it does not.
//
// A Set caches the ops it builds, so keep the one New returns rather than
// making a fresh one per frame.
type Set struct {
	goos  string
	reg   *icon.Registry
	names []Name

	mu    sync.Mutex
	ops   *op.Ops
	cache map[cacheKey]op.CallOp
}

// cacheKey is what a built drawing is stored under: the registry key that
// answered, the pixel size it was built for, and the widening its band took
// there, in thousandths of a device pixel. Colour is not part of it — the
// painter applies colour outside the recorded ops.
type cacheKey struct {
	entry string
	px    int
	widen int
}

// New returns the set as the named operating system sees it. goos is a
// runtime.GOOS value; any string is accepted, and a system no mark is drawn
// for gets the fallback drawing throughout.
//
// Passing a platform explicitly is the seam this package resolves through:
// there is no build-tag fan-out, so a test or a preview can ask for another
// system's answer on this one.
func New(goos string) *Set {
	s := &Set{
		goos:  goos,
		reg:   icon.New(),
		ops:   new(op.Ops),
		cache: make(map[cacheKey]op.CallOp),
	}
	marks := mustLoadMarks()
	for _, key := range marks.keys() {
		s.reg.Register(key, icon.FromSVG(marks[key]))
		if !strings.Contains(key, platformSep) {
			s.names = append(s.names, Name(key))
		}
	}
	return s
}

// Register adds every mark in the set to r, under the key its file name
// gives it: the plain name for the drawing that serves every platform, and
// name@goos for a platform's own drawing. The icons registered are freshly
// parsed, so a registry an application owns shares nothing with this
// package's own state.
func Register(r *icon.Registry) {
	for key, parsed := range mustLoadMarks() {
		r.Register(key, icon.FromSVG(parsed))
	}
}

// Resolve reports the registry key that answers name on this set's platform:
// name@goos when the set carries that platform's own drawing, name when it
// falls back to the drawing every platform shares. ok is false for a name the
// set does not carry at all.
func (s *Set) Resolve(name Name) (key string, ok bool) {
	if k := string(name) + platformSep + s.goos; s.carries(k) {
		return k, true
	}
	if k := string(name); s.carries(k) {
		return k, true
	}
	return "", false
}

// Has reports whether the set carries a mark under name.
func (s *Set) Has(name Name) bool {
	_, ok := s.Resolve(name)
	return ok
}

// Names lists every mark in the set, in order. Platform variants are not
// separate entries: a name appears once however many drawings stand behind it.
func (s *Set) Names() []Name {
	return slices.Clone(s.names)
}

// GOOS reports the operating system this set resolves for.
func (s *Set) GOOS() string { return s.goos }

// Mark returns a painter for name, or nil when the set carries no such mark.
// A painter is cheap: it looks the mark up once and closes over the set's
// shared cache, so building one per frame costs nothing worth avoiding.
func (s *Set) Mark(name Name) Painter {
	entry, ok := s.Resolve(name)
	if !ok {
		return nil
	}
	return func(gtx layout.Context, sizePx int, col color.NRGBA) {
		if sizePx <= 0 {
			return
		}
		call := s.drawing(entry, sizePx, widening(gtx.Metric.PxPerDp, sizePx))
		paint.ColorOp{Color: col}.Add(gtx.Ops)
		call.Add(gtx.Ops)
	}
}

// carries reports whether the set holds a drawing under the registry key.
func (s *Set) carries(key string) bool {
	_, ok := s.reg.Icon(key)
	return ok
}

var (
	hostOnce sync.Once
	hostSet  *Set
)

// host is the set as the running process's operating system sees it, built on
// first use so that importing this package parses nothing.
func host() *Set {
	hostOnce.Do(func() { hostSet = New(runtime.GOOS) })
	return hostSet
}

// Mark returns a painter for name as the host operating system draws it, or
// nil when the set carries no such mark.
func Mark(name Name) Painter { return host().Mark(name) }

// Has reports whether the set carries a mark under name.
func Has(name Name) bool { return host().Has(name) }

// Names lists every mark in the set, in order.
func Names() []Name { return host().Names() }

// Resolve reports the registry key that answers name on the host operating
// system. See Set.Resolve.
func Resolve(name Name) (key string, ok bool) { return host().Resolve(name) }

// markSet is the parsed source keyed by registry key.
type markSet map[string]*svg.Icon

// keys returns the set's registry keys in order.
func (m markSet) keys() []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

// loadMarks parses every embedded mark into a fresh set of icons keyed by
// registry key. Each call parses again: the drawings are mutated while their
// ops are built, so two owners must not share one.
func loadMarks() (markSet, error) {
	entries, err := marksFS.ReadDir(markDir)
	if err != nil {
		return nil, err
	}
	out := make(markSet, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		key, err := registryKey(e.Name())
		if err != nil {
			return nil, err
		}
		f, err := marksFS.Open(markDir + "/" + e.Name())
		if err != nil {
			return nil, err
		}
		parsed, err := parser.NewParser(parser.StrictErrorMode).ParseStream(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out[key] = parsed
	}
	return out, nil
}

// mustLoadMarks is loadMarks over source that ships inside the binary. A
// failure means a mark file in this package is malformed or misnamed, which
// is a broken build rather than a runtime condition — the package's own tests
// parse and name-check every file, so this cannot fire on a tree that passes
// them.
func mustLoadMarks() markSet {
	set, err := loadMarks()
	if err != nil {
		panic("icons: " + err.Error())
	}
	return set
}

// registryKey turns a mark's file name into its registry key: "sidebar.svg"
// into "sidebar", "sidebar.darwin.svg" into "sidebar@darwin".
func registryKey(file string) (string, error) {
	base := strings.TrimSuffix(file, ".svg")
	name, goos, hasVariant := strings.Cut(base, ".")
	if err := checkName(name); err != nil {
		return "", fmt.Errorf("%s: %w", file, err)
	}
	if !hasVariant {
		return name, nil
	}
	if goos == "" || strings.Contains(goos, ".") {
		return "", fmt.Errorf("%s: expected <name>.svg or <name>.<goos>.svg", file)
	}
	return name + platformSep + goos, nil
}

// checkName enforces the naming rule: lowercase ASCII words joined by "-".
func checkName(name string) error {
	if name == "" {
		return fmt.Errorf("empty mark name")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
		default:
			return fmt.Errorf("mark name %q: only lowercase letters, digits and %q are allowed", name, "-")
		}
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("mark name %q: %q is a word separator, not an edge", name, "-")
	}
	return nil
}
