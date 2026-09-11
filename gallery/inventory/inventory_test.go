package inventory

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/vibrantgio/components/golden"
	themecolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// sampleWidth is the width a section is captured at here — wide enough that
// the shell's three columns and the pricing tiers lay out in a shape a window
// would actually show them in.
const sampleWidth = 900

// testInventory builds the inventory a test draws from, with the shaper
// resolving no system fonts and the control marks pinned to one platform, so
// the same bytes come out on any machine.
func testInventory(t *testing.T) *Inventory {
	t.Helper()
	return NewForOS(tokens.DefaultTypography.DeterministicShaper(), "darwin")
}

// shot captures one section's slot exactly as the column lays it out: the
// scheme's content plane, the section's own height, and nothing else on it.
func shot(t *testing.T, c tokens.PlatformColors, s Section) *image.RGBA {
	t.Helper()
	return golden.Capture(t, image.Pt(sampleWidth, int(s.Height)+40), sectionBody(c, s))
}

// changed reports what share of a section's pixels moved between the two
// recorded sets, as a percentage.
func changed(a, b *image.RGBA) float64 {
	bounds := a.Bounds()
	return 100 * float64(golden.PixelDiff(a, b)) / float64(bounds.Dx()*bounds.Dy())
}

// share reports what share of img carries exactly want, as a percentage.
// Alpha is ignored: a capture is opaque.
func share(img *image.RGBA, want color.NRGBA) float64 {
	b := img.Bounds()
	hit := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if c := img.RGBAAt(x, y); c.R == want.R && c.G == want.G && c.B == want.B {
				hit++
			}
		}
	}
	return 100 * float64(hit) / float64(b.Dx()*b.Dy())
}

// TestNoSectionIsPinnedToAnAppearance is the standing hunt for a surface that
// draws itself out of something other than the set it was handed.
//
// The two recorded sets invert every plane a section stands on, so a section
// that follows the set it is handed cannot come out looking similar in the
// two. One that does — a popup drawn from a default set, a fill remembered
// from a previous frame — is the defect this looks for, and it is invisible
// on a page where everything around it changed correctly.
//
// It is asked in two ways, because one is no longer enough. The platform
// reports one accent and one set of system colours for BOTH appearances, so a
// specimen that is largely accent fill holds that share still by the
// platform's own answer: the area floor had to come down, and a defect a
// fifth of a slot across would clear it. The second question closes that —
// the other appearance's own content plane may not survive into this one,
// and a panel drawn from a default set is exactly that plane.
func TestNoSectionIsPinnedToAnAppearance(t *testing.T) {
	inv := testInventory(t)
	light, dark := tokens.PlatformLight, tokens.PlatformDark
	lit, drk := inv.Groups(light), inv.Groups(dark)
	for g := range lit {
		for i := range lit[g].Sections {
			s := lit[g].Sections[i]
			t.Run(s.Name, func(t *testing.T) {
				lit, drk := shot(t, light, s), shot(t, dark, drk[g].Sections[i])
				if lit == nil || drk == nil {
					return // headless unavailable; Capture called t.Skip
				}
				if !schemeFixedSections[s.Name] {
					pct := changed(lit, drk)
					t.Logf("%s: %.3f%% of the slot follows the appearance", s.Name, pct)
					if pct < schemeFloor {
						t.Errorf("%s changed %.1f%% between the two appearances, want at least %.0f%% — part of it is drawn from something other than the set it was handed",
							s.Name, pct, schemeFloor)
					}
				}
				if got := share(drk, light.ControlBackground); got > planeFloor {
					t.Errorf("%s: %.2f%% of the dark slot is the light set's own content plane, over the %.1f%% glyphs and marks account for — something in it is drawn from a default set",
						s.Name, got, planeFloor)
				}
				if got := share(lit, dark.ControlBackground); got > planeFloor {
					t.Errorf("%s: %.2f%% of the light slot is the dark set's own content plane, over the %.1f%% glyphs and marks account for — something in it is drawn from a default set",
						s.Name, got, planeFloor)
				}
			})
		}
	}
}

// schemeFixedSections are the sections excused from the AREA half of the hunt
// above, and the only ones. Both are sections whose subject is something that
// does not follow the appearance, so holding still is what they are there to
// show: the pinned row carries a fill the caller chose from outside the set,
// and the platform-set section prints both appearances at once, which is what
// makes a coverage legible as the pair it is. Neither is excused from the
// plane half — a default-set panel inside either would still be caught.
var schemeFixedSections = map[string]bool{
	"components-button-pinned": true,
	"foundations-platform":     true,
}

// sectionNamed returns the components section with the given name, drawn
// from c.
func sectionNamed(t *testing.T, inv *Inventory, c tokens.PlatformColors, name string) Section {
	t.Helper()
	for _, s := range inv.Components(c) {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("no section named %q", name)
	return Section{}
}

// TestThePinnedFillHoldsWhileTheAppearanceFlips is the appearance hunt
// inverted, on the one row that must fail it. Both cells are captured in both
// appearances: the theme's own cell has to read the set's own accent for the
// appearance it was handed, and the pinned one has to come back the exact
// colour it was handed, because that one is the caller's.
//
// The two halves are asserted differently, and deliberately. The platform
// reports ONE accent for both appearances, so "the theme's own fill moved"
// is no longer a claim that can be made about it — what is asserted instead
// is that the cell reads the accent of the set it was drawn from, which a row
// that stopped drawing, or that drew from a default set, fails just as
// loudly.
func TestThePinnedFillHoldsWhileTheAppearanceFlips(t *testing.T) {
	inv := testInventory(t)
	light, dark := tokens.PlatformLight, tokens.PlatformDark
	lit := shot(t, light, sectionNamed(t, inv, light, "components-button-pinned"))
	drk := shot(t, dark, sectionNamed(t, inv, dark, "components-button-pinned"))
	if lit == nil || drk == nil {
		return // headless unavailable; Capture called t.Skip
	}

	// A pixel inside each cell's button and left of its label: the row is
	// laid out from the section's own margin, one cell wide with the gap
	// between, and the button is the full cell width and 36 dp tall.
	y := int(SectionPadY) + 18
	stockX := int(SectionPadX) + 10
	pinnedX := int(SectionPadX+ButtonCellW+ButtonCellGap) + 10

	opaque := func(v color.NRGBA) color.RGBA {
		return color.RGBA{R: v.R, G: v.G, B: v.B, A: 0xff}
	}
	for _, c := range []struct {
		name string
		img  *image.RGBA
		set  tokens.PlatformColors
	}{{"light", lit, light}, {"dark", drk, dark}} {
		if got, want := c.img.RGBAAt(stockX, y), opaque(c.set.ControlAccent); got != want {
			t.Errorf("%s: the theme's own cell at (%d,%d) = %v, want the set's own accent %v — this half of the row is not drawn from the set it was handed",
				c.name, stockX, y, got, want)
		}
		if got, want := c.img.RGBAAt(pinnedX, y), opaque(PinnedFill); got != want {
			t.Errorf("%s: the pinned cell at (%d,%d) = %v, want the colour it was pinned to %v",
				c.name, pinnedX, y, got, want)
		}
	}
}

// The two floors the hunt above holds a section to.
const (
	// schemeFloor is the share of a section's slot that must move between the
	// two appearances. It is not the whole slot: the platform's accent and
	// its system colours are one value in both appearances, so a specimen
	// that is largely accent fill holds that share still by the platform's
	// own answer. The row of five filled buttons is the lowest at 88.7%, the
	// recommended pricing tier next at 96.3%, and every section without a
	// large accent fill measures over 99.9%.
	schemeFloor = 85.0

	// planeFloor is how much of a slot may read as the OTHER appearance's own
	// content plane. It is what keeps the loosened area floor from letting a
	// pinned surface hide. White and near-black occur legitimately — the
	// foreground the platform pairs with an accent fill is white, and a
	// syntax palette may name either — so the floor stands above what glyphs
	// and marks account for and far below what a panel would put there.
	planeFloor = 2.0
)

// TestTheCodeSpecimenIsLast pins where the syntax plate sits. The column is
// for judging a theme, and code at the top of it takes the attention the rest
// of the surface is there to get; the plate is the last thing on the page, and
// the lookup a caller scrolls by has to follow it there rather than keep
// pointing at the row it used to be on.
func TestTheCodeSpecimenIsLast(t *testing.T) {
	inv := testInventory(t)
	c := tokens.PlatformLight

	groups := inv.Groups(c)
	last := groups[len(groups)-1].Sections
	if got := last[len(last)-1].Name; got != CodeSectionName() {
		t.Errorf("the page ends on section %q, want %q", got, CodeSectionName())
	}

	// The rows Items lays out are, per group, a banner and then a heading and
	// a body per section, with one closing row under the lot. The code's
	// heading is therefore the third row from the end, and ItemIndex has to
	// say so — a lookup left pointing at the old row would scroll a caller to
	// whatever moved into it.
	items := inv.Items(c)
	row := inv.ItemIndex(c, CodeSectionName())
	if want := len(items) - 3; row != want {
		t.Errorf("ItemIndex(%q) = %d, want %d — the lookup did not follow the section", CodeSectionName(), row, want)
	}
	if inv.ItemIndex(c, "no-such-section") != -1 {
		t.Error("ItemIndex answered a row for a section that does not exist")
	}
}

// TestTheSchemeControlIsAControl measures the light/dark control on both sides
// of the scheme: the target each half offers a pointer, and how far each glyph
// stands off the surface it is drawn on.
//
// Both halves are measured, not only the current one. A segment control says
// what it does by showing the choice that is not in force beside the one that
// is, and a glyph that has faded into the track says only that something is
// selected.
func TestTheSchemeControlIsAControl(t *testing.T) {
	// The whole control, and one segment of it, at one dp per pixel.
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(400, 400)},
		Ops:         new(op.Ops),
	}
	for _, sc := range []struct {
		name string
		c    tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		for _, dark := range []bool{false, true} {
			for _, selected := range []bool{false, true} {
				dims := SchemeSegment(sc.c, dark, selected)(gtx)
				if want := image.Pt(int(SchemeSegmentW), int(SchemeSwitchH)); dims.Size != want {
					t.Errorf("%s: a segment measured %v, want %v", sc.name, dims.Size, want)
				}
				if dims.Size.X < int(schemeTargetFloor) {
					t.Errorf("%s: a segment is %v wide, under the %v target floor",
						sc.name, dims.Size.X, schemeTargetFloor)
				}
				if dims.Size.Y < int(schemeDrawnFloor) {
					t.Errorf("%s: a segment draws %v tall, under the %v a target owes before any slop is spent on it",
						sc.name, dims.Size.Y, schemeDrawnFloor)
				}
				foreground, fill := schemeSegmentColors(sc.c, selected)
				ratio := themecolor.Magnitude(foreground, fill)
				t.Logf("%s scheme, %s segment, selected=%v: glyph |Lc| %.2f on the fill behind it",
					sc.name, map[bool]string{false: "sun", true: "moon"}[dark], selected, ratio)
				if ratio < schemeGlyphFloor {
					t.Errorf("%s: a glyph on a segment with selected=%v measures |Lc| %.2f, want at least |Lc| %.1f",
						sc.name, selected, ratio, schemeGlyphFloor)
				}
			}
		}
		whole := SchemeSwitch(sc.c, false)(gtx)
		if want := image.Pt(int(SchemeSwitchW), int(SchemeSwitchH)); whole.Size != want {
			t.Errorf("%s: the control measured %v, want %v", sc.name, whole.Size, want)
		}
		// The track is the platform's push button, which reads eight percent
		// off the plane behind it in the light appearance. That is the
		// platform's own separation, so what is pinned here is that the two
		// are not the same colour rather than a ratio of this library's
		// choosing.
		if schemeTrack(sc.c) == sc.c.WindowBackground {
			t.Errorf("%s: the control's track is the window's own plane; nothing marks it as a control",
				sc.name)
		} else {
			t.Logf("%s appearance: track %v against the plane's %v",
				sc.name, schemeTrack(sc.c), sc.c.WindowBackground)
		}
	}
}

// TestTheSchemeTargetOutlivesTheTrack is the other half of the control's size.
// The track is cut to the scale of the strip it stands in; what somebody has
// to land a pointer on is not cut with it, and a press in the slop above or
// below the track is a press on the control.
//
// It also holds the layout to the track: a target that reported its own height
// to the row around it would push that row back open and undo the cut.
func TestTheSchemeTargetOutlivesTheTrack(t *testing.T) {
	// The control stands away from the origin, so the slop above it is at
	// coordinates a pointer can be put on.
	const at = 20
	btn := new(widget.Clickable)
	r := new(gioinput.Router)
	ops := new(op.Ops)
	var drawn layout.Dimensions
	clicked := 0
	drive := func() {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(200, 200)),
			Ops:         ops,
			Source:      r.Source(),
		}
		// Read the frame's presses before drawing it, which is the order a
		// caller reads them in: laying the control out drains them.
		for btn.Clicked(gtx) {
			clicked++
		}
		off := op.Offset(image.Pt(at, at)).Push(gtx.Ops)
		drawn = SchemeTarget(gtx, btn.Layout, SchemeSegment(tokens.PlatformLight, false, true))
		off.Pop()
		r.Frame(ops)
	}
	drive() // register the press area

	if want := image.Pt(int(SchemeSegmentW), int(SchemeSwitchH)); drawn.Size != want {
		t.Fatalf("the target reports %v to the row it stands in, want the track's own %v — a row laid out on that is not at the scale the track was cut to",
			drawn.Size, want)
	}
	slop := (int(SchemeTargetH) - int(SchemeSwitchH)) / 2
	if slop < 1 {
		t.Fatalf("a %v track under a %v target leaves no slop to press in", SchemeSwitchH, SchemeTargetH)
	}
	t.Logf("the track draws %v tall, the target is %v, so the slop is %d px above and below", SchemeSwitchH, SchemeTargetH, slop)

	x := float32(at + int(SchemeSegmentW)/2)
	for _, p := range []struct {
		where string
		pos   f32.Point
	}{
		{"above the track", f32.Pt(x, float32(at-slop)+0.5)},
		{"on the track", f32.Pt(x, float32(at+int(SchemeSwitchH)/2))},
		{"below the track", f32.Pt(x, float32(at+int(SchemeSwitchH)+slop)-0.5)},
	} {
		before := clicked
		r.Queue(
			pointer.Event{Kind: pointer.Press, Position: p.pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			pointer.Event{Kind: pointer.Release, Position: p.pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		)
		drive()
		if clicked == before {
			t.Errorf("a press %s, at %v, reached nothing — the control's target does not cover it", p.where, p.pos)
		}
	}
	// And it stops where the target does: a press a whole target away is a
	// press on whatever else is there.
	before := clicked
	pos := f32.Pt(x, float32(at-int(SchemeTargetH)))
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if clicked != before {
		t.Errorf("a press at %v, a whole target above the control, reached it — the slop does not end", pos)
	}
}

// TestTheSchemeFillHoldsItsGlyph is the floor under the track's height. The
// track is cut to the scale of the strip it stands in and the glyph on it is
// not cut with it, so the fill marking the current half has to go on holding
// that glyph with something to spare — a mark touching the edge of the thing
// that marks it reads as a mark that has outgrown it.
func TestTheSchemeFillHoldsItsGlyph(t *testing.T) {
	fill := SchemeSwitchH - 2*schemeThumbInset
	if fill <= schemeIconSize {
		t.Fatalf("the fill is %v round a %v glyph: the mark is as big as what marks it", fill, schemeIconSize)
	}
	t.Logf("track %v, inset %v either side, so the fill is %v round a %v glyph — %v clear at each edge",
		SchemeSwitchH, schemeThumbInset, fill, schemeIconSize, (fill-schemeIconSize)/2)
}

// The floors the control is held to.
const (
	// schemeTargetFloor is the smallest press area either side of the control
	// may offer. It is the standing minimum for a standalone control —
	// something pointed at rather than aimed at — and it is what a segment is
	// held to whether it draws that tall or has the difference handed to it as
	// slop.
	schemeTargetFloor unit.Dp = 44
	// schemeDrawnFloor is the smallest the drawn track may get. A control cut
	// to the scale of a strip still owes the minimum any target owes before a
	// point of slop is added to it, so the track clears that on its own and
	// the slop is what carries it the rest of the way to schemeTargetFloor.
	schemeDrawnFloor unit.Dp = 24
	// schemeGlyphFloor is the contrast a glyph needs against what is behind
	// it. A glyph is a graphic and not a line of text, which is the lower of
	// the two standing floors.
	schemeGlyphFloor = tokens.GraphicFloor
)

// TestTabItemsDropTheBanner is the whole point of the tab cut: a surface
// showing one group carries no banner repeating the name that was clicked to
// reach it. GroupItems leads with that banner, so the cut is its rows less
// one, plus the closing line.
func TestTabItemsDropTheBanner(t *testing.T) {
	inv := testInventory(t)
	c := tokens.PlatformLight
	for _, grp := range inv.Groups(c) {
		want := 2 * len(grp.Sections) // heading + body per section
		got := len(inv.TabItems(c, grp.Name)) - 1
		if got != want {
			t.Errorf("the %s cut lays out %d rows before its closing line, want %d — the group banner is still in",
				grp.Name, got, want)
		}
	}
}

// TestTabItemsAreNilForAnUnknownGroup pins the wiring guard: a name no group
// carries is a fault a caller's test must be able to see, not a blank column
// on screen.
func TestTabItemsAreNilForAnUnknownGroup(t *testing.T) {
	inv := testInventory(t)
	if rows := inv.TabItems(tokens.PlatformLight, "Nothing Named This"); rows != nil {
		t.Errorf("an unknown group name returned %d rows, want nil", len(rows))
	}
}
