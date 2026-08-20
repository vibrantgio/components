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

// Two brand colours far enough apart that nothing derived from one survives
// the other: opposite sides of the hue circle, both vivid.
var (
	seedA = color.NRGBA{R: 0x1e, G: 0x66, B: 0xff, A: 0xff}
	seedB = color.NRGBA{R: 0xff, G: 0x6a, B: 0x00, A: 0xff}
)

// testInventory builds the inventory a test draws from, with the shaper
// resolving no system fonts and the control marks pinned to one platform, so
// the same bytes come out on any machine.
func testInventory(t *testing.T) *Inventory {
	t.Helper()
	return NewForOS(tokens.DefaultTypography.DeterministicShaper(), "darwin")
}

// shot captures one section's slot exactly as the column lays it out: the
// scheme's ground, the section's own height, and nothing else on it.
func shot(t *testing.T, c tokens.ColorTokens, s Section) *image.RGBA {
	t.Helper()
	return golden.Capture(t, image.Pt(sampleWidth, int(s.Height)+40), sectionBody(c, s))
}

// changed reports what share of a section's pixels moved between two
// palettes, as a percentage.
func changed(a, b *image.RGBA) float64 {
	bounds := a.Bounds()
	return 100 * float64(golden.PixelDiff(a, b)) / float64(bounds.Dx()*bounds.Dy())
}

// TestNoSectionIsPinnedToAScheme is the standing hunt for a surface that
// draws itself out of something other than the tokens it was handed.
//
// Light and dark of one seed invert the neutral ramps every section stands
// on, so a section that follows its tokens cannot come out looking similar in
// the two. One that does — a popup drawn from a default palette, a fill
// remembered from a previous frame — is the defect this looks for, and it is
// invisible on a page where everything around it changed correctly.
func TestNoSectionIsPinnedToAScheme(t *testing.T) {
	inv := testInventory(t)
	light, dark := tokens.FromSeed(seedA)
	lit, drk := inv.Groups(light), inv.Groups(dark)
	for g := range lit {
		for i := range lit[g].Sections {
			s := lit[g].Sections[i]
			t.Run(s.Name, func(t *testing.T) {
				pct := changed(shot(t, light, s), shot(t, dark, drk[g].Sections[i]))
				t.Logf("%s: %.3f%% of the slot follows the scheme", s.Name, pct)
				if pct < schemeFloor {
					t.Errorf("%s changed %.1f%% between light and dark, want at least %.0f%% — part of it is drawn from something other than the tokens it was handed",
						s.Name, pct, schemeFloor)
				}
			})
		}
	}
}

// schemeFloor is the share of a section's slot that must move when the scheme
// flips. Every section measures over 99.98% today — the ground itself
// inverts, so all a section leaves standing is the odd antialiased pixel
// where two edges happen to blend to the same byte — and the floor is set
// just under the whole slot rather than at some cautious fraction. A fraction
// is what lets a pinned surface hide: the recorded case was a popup drawn
// light on a dark page, and its panel is a fifth of the slot it sits in, so
// anything below about 80% would have called it well themed.
const schemeFloor = 99.5

// TestTheSeedReachesEveryGroup: a brand colour is meant to reach all four
// groups, not only the palette swatches at the top. Each group is captured
// with its section bodies alone — no banner, whose fill is the seed's primary
// and would carry the whole assertion by itself.
func TestTheSeedReachesEveryGroup(t *testing.T) {
	inv := testInventory(t)
	a, _ := tokens.FromSeed(seedA)
	b, _ := tokens.FromSeed(seedB)
	for g, grp := range inv.Groups(a) {
		other := inv.Groups(b)[g]
		t.Run(grp.Name, func(t *testing.T) {
			moved := 0.0
			for i, s := range grp.Sections {
				moved += changed(shot(t, a, s), shot(t, b, other.Sections[i]))
			}
			t.Logf("%s: %.1f%% summed across %d sections", grp.Name, moved, len(grp.Sections))
			if moved == 0 {
				t.Errorf("group %q drew identically under two brand colours — the seed does not reach it", grp.Name)
			}
		})
	}
}

// TestTheCodeSpecimenIsLast pins where the syntax plate sits. The column is
// for judging a theme, and code at the top of it takes the attention the rest
// of the surface is there to get; the plate is the last thing on the page, and
// the lookup a caller scrolls by has to follow it there rather than keep
// pointing at the row it used to be on.
func TestTheCodeSpecimenIsLast(t *testing.T) {
	inv := testInventory(t)
	c, _ := tokens.FromSeed(seedA)

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

// TestTheSchemeControlIsAControl measures the light/dark control on both
// sides of the scheme: the target each half offers a pointer, and how far each
// glyph stands off the ground it is drawn on.
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
		c    tokens.ColorTokens
	}{{"light", tokens.DefaultLight}, {"dark", tokens.DefaultDark}} {
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
				ink, ground := schemeSegmentInks(sc.c, selected)
				ratio := themecolor.ContrastRatio(ink, ground)
				t.Logf("%s scheme, %s segment, selected=%v: glyph %.2f:1 on its ground",
					sc.name, map[bool]string{false: "sun", true: "moon"}[dark], selected, ratio)
				if ratio < schemeGlyphFloor {
					t.Errorf("%s: a glyph on a segment with selected=%v measures %.2f:1, want at least %.1f:1",
						sc.name, selected, ratio, schemeGlyphFloor)
				}
			}
		}
		whole := SchemeSwitch(sc.c, false)(gtx)
		if want := image.Pt(int(SchemeSwitchW), int(SchemeSwitchH)); whole.Size != want {
			t.Errorf("%s: the control measured %v, want %v", sc.name, whole.Size, want)
		}
		// The track has to be findable on the page before either glyph on it
		// can say anything, and the page under it is the scheme's background.
		if r := themecolor.ContrastRatio(schemeTrack(sc.c), sc.c.Background); r < schemeTrackFloor {
			t.Errorf("%s: the control's track measures %.2f:1 against the page, want at least %.2f:1",
				sc.name, r, schemeTrackFloor)
		} else {
			t.Logf("%s scheme: track %.2f:1 against the page", sc.name, r)
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
		drawn = SchemeTarget(gtx, btn.Layout, SchemeSegment(tokens.DefaultLight, false, true))
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
	schemeGlyphFloor = 3.0
	// schemeTrackFloor is how far the control as a whole has to stand off the
	// page. It is a shape rather than a mark, so the floor is set where a flat
	// fill stops being mistakable for the ground and not at the mark's.
	schemeTrackFloor = 1.3
)
