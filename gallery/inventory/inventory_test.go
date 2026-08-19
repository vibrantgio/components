package inventory

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

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
				if dims.Size.X < int(schemeTargetFloor) || dims.Size.Y < int(schemeTargetFloor) {
					t.Errorf("%s: a segment is %v, under the %v target floor",
						sc.name, dims.Size, schemeTargetFloor)
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

// The floors the control is held to.
const (
	// schemeTargetFloor is the smallest either side of a segment may get. It
	// is the standing minimum for something pointed at rather than aimed at,
	// and a control on a page somebody is reading is pointed at.
	schemeTargetFloor unit.Dp = 32
	// schemeGlyphFloor is the contrast a glyph needs against what is behind
	// it. A glyph is a graphic and not a line of text, which is the lower of
	// the two standing floors.
	schemeGlyphFloor = 3.0
	// schemeTrackFloor is how far the control as a whole has to stand off the
	// page. It is a shape rather than a mark, so the floor is set where a flat
	// fill stops being mistakable for the ground and not at the mark's.
	schemeTrackFloor = 1.3
)
