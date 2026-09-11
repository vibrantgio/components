package badge_test

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/components/badge"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/theme/tokens"
)

// cross is the second deterministic sign the disc specimens draw, beside
// [check]. Two signs rather than one because the disc keeps the obligation a
// bare glyph badge carries: a set of glyph badges owes distinct shapes, and a
// disc puts a field of hue behind the sign without making hue enough on its
// own.
//
// Its arms span the same 0.16 to 0.84 of the box [check] does, so the two fill
// their boxes to one extent: a pair of signs at two extents reads as two icon
// sets rather than as one shape varying.
func cross(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	stroke := float32(gtx.Dp(unit.Dp(1.5)))
	if stroke < 1 {
		stroke = 1
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w*0.16, w*0.16))
	p.LineTo(f32.Pt(w*0.84, w*0.84))
	p.MoveTo(f32.Pt(w*0.84, w*0.16))
	p.LineTo(f32.Pt(w*0.16, w*0.84))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

// blank is a sign that paints nothing. It leaves a disc badge showing its
// fill and nothing else, which is the only way to measure the circle itself
// rather than the circle plus whatever was drawn on it.
func blank(layout.Context, int, color.NRGBA) {}

// dot fills the middle half of the box it is handed, on whole pixels so its
// edges are crisp and its painted extent is exactly the box it claims. A
// symmetric sign is what a centring assertion needs: the check is asymmetric
// by design, so what it paints says nothing about where the badge put it.
func dot(gtx layout.Context, sizePx int, col color.NRGBA) {
	q := sizePx / 4
	paint.FillShape(gtx.Ops, col, clip.Rect{
		Min: image.Pt(q, q), Max: image.Pt(sizePx-q, sizePx-q),
	}.Op())
}

// square fills the whole box it is handed, corner to corner. It is the worst
// case the disc has to hold: a [badge.Glyph] is a painter the package cannot
// inspect, so the size it hands one has to keep even a sign that uses all of
// it inside the circle.
func square(gtx layout.Context, sizePx int, col color.NRGBA) {
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: image.Pt(sizePx, sizePx)}.Op())
}

// discState is the render state a disc badge is drawn in.
func discState() badge.RenderState {
	return badge.RenderState{Disc: true}
}

// paintedBox reports the bounding box of the pixels that are not the page,
// searched over the badge's own box plus a pixel of margin on every side, in
// coordinates measured from the badge's top-left corner. An empty box comes
// back as ok false.
//
// The margin is the point: a fill that overran the box the badge reported
// would land in it, so the measurement can catch a circle wider than the
// badge as well as one narrower.
func paintedBox(img *image.RGBA, page color.NRGBA, size image.Point) (image.Rectangle, bool) {
	box := image.Rectangle{Min: image.Pt(1<<30, 1<<30), Max: image.Pt(-1<<30, -1<<30)}
	found := false
	for dy := -1; dy <= size.Y; dy++ {
		for dx := -1; dx <= size.X; dx++ {
			c := img.RGBAAt(goldenInset+dx, goldenInset+dy)
			if (color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}) == page {
				continue
			}
			found = true
			box.Min.X = min(box.Min.X, dx)
			box.Min.Y = min(box.Min.Y, dy)
			box.Max.X = max(box.Max.X, dx+1)
			box.Max.Y = max(box.Max.Y, dy+1)
		}
	}
	return box, found
}

// exactBox is [paintedBox] narrowed to one colour: the bounding box of the
// pixels that are exactly that colour and no other.
func exactBox(img *image.RGBA, want color.NRGBA, size image.Point) (image.Rectangle, bool) {
	box := image.Rectangle{Min: image.Pt(1<<30, 1<<30), Max: image.Pt(-1<<30, -1<<30)}
	found := false
	for dy := 0; dy < size.Y; dy++ {
		for dx := 0; dx < size.X; dx++ {
			c := img.RGBAAt(goldenInset+dx, goldenInset+dy)
			if (color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}) != want {
				continue
			}
			found = true
			box.Min.X = min(box.Min.X, dx)
			box.Min.Y = min(box.Min.Y, dy)
			box.Max.X = max(box.Max.X, dx+1)
			box.Max.Y = max(box.Max.Y, dy+1)
		}
	}
	return box, found
}

// centre of a box, in half-pixel units so an even span and an odd one are
// both exact.
func centre(r image.Rectangle) (float64, float64) {
	return float64(r.Min.X+r.Max.X) / 2, float64(r.Min.Y+r.Max.Y) / 2
}

// TestTheDiscIsTheLineBoxAcross is the geometry ruling in one assertion: the
// disc is a circle the glyph's line box across, which is the box a labelled
// badge's line already reserves at that density. So a disc badge measures the
// same line-box square a bare one does, and a row that held one holds the
// other unmoved — which is the whole reason the disc costs a caller nothing.
func TestTheDiscIsTheLineBoxAcross(t *testing.T) {
	shaper := defaultShaper(t)
	for _, d := range []struct {
		name string
		d    tokens.Density
	}{
		{"comfortable", tokens.Comfortable},
		{"compact", tokens.Compact},
	} {
		style := badge.Style(tokens.DefaultTypography, d.d)
		want := int(style.LineHeight)
		t.Run(d.name, func(t *testing.T) {
			disc := badge.Render(shaper, "", check, badge.Success,
				tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, discState())
			bare := badge.Render(shaper, "", check, badge.Success,
				tokens.PlatformLight, tokens.Spacing, tokens.Radius, style,
				badge.RenderState{})
			size := measure(t, disc)
			if size.X != want || size.Y != want {
				t.Errorf("a disc badge measured %v, want the %d dp line box square", size, want)
			}
			if got := measure(t, bare); got != size {
				t.Errorf("a disc badge measured %v and a bare one %v: the disc must not move the box", size, got)
			}

			// The painted circle, measured rather than assumed. A fill
			// drawn at the wrong diameter reports the same size and looks
			// wrong, so the size assertion above cannot stand alone.
			w := badge.Render(shaper, "", blank, badge.Success,
				tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, discState())
			img := golden.Capture(t, goldenSize, onPage(tokens.PlatformLight, w))
			if img == nil {
				return // headless unavailable; Capture called t.Skip
			}
			page := tokens.PlatformLight.ControlBackground
			box, ok := paintedBox(img, page, size)
			if !ok {
				t.Fatal("a disc badge painted nothing at all")
			}
			if box.Dx() != want || box.Dy() != want {
				t.Errorf("the disc painted %d×%d, want the %d dp line box across both ways",
					box.Dx(), box.Dy(), want)
			}
			if box.Min.X != 0 || box.Min.Y != 0 {
				t.Errorf("the disc starts at %v inside the badge, want the badge's own corner — it is not inscribed in the box it reported",
					box.Min)
			}
		})
	}
}

// TestTheGlyphIsCentredInTheDisc pins the other half of the geometry: the sign
// shares the disc's square, so its centre and the circle's are one point. A
// sign drawn a pixel off centre inside a circle is the defect a still image
// of one badge cannot show and a row of five makes obvious.
func TestTheGlyphIsCentredInTheDisc(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, sc := range goldenSchemes {
		t.Run(sc.name, func(t *testing.T) {
			w := badge.Render(shaper, "", dot, badge.Info,
				sc.p, tokens.Spacing, tokens.Radius, style, discState())
			size := measure(t, w)
			img := golden.Capture(t, goldenSize, onCard(sc.p, w))
			if img == nil {
				return // headless unavailable; Capture called t.Skip
			}
			page := sc.p.CardFill
			discBox, ok := paintedBox(img, page, size)
			if !ok {
				t.Fatal("a disc badge painted nothing at all")
			}
			fg := badge.Foreground(sc.p)
			signBox, ok := exactBox(img, fg, size)
			if !ok {
				t.Fatalf("no pixel of the sign came back in its own foreground %v", fg)
			}
			dcx, dcy := centre(discBox)
			scx, scy := centre(signBox)
			if diff := dcx - scx; diff > 1 || diff < -1 {
				t.Errorf("the sign's centre is %.1f across and the disc's is %.1f: the sign is not centred in the disc", scx, dcx)
			}
			if diff := dcy - scy; diff > 1 || diff < -1 {
				t.Errorf("the sign's centre is %.1f down and the disc's is %.1f: the sign is not centred in the disc", scy, dcy)
			}
			if signBox.Dx() >= discBox.Dx() || signBox.Dy() >= discBox.Dy() {
				t.Errorf("the sign measured %v and the disc %v: the sign is not inside the circle", signBox.Size(), discBox.Size())
			}
		})
	}
}

// TestTheDiscWearsItsStatusFillAndForeground samples what actually landed:
// the status's own [badge.Fill] behind the sign and [badge.Foreground] on it.
// The disc wears the system colour itself and never a tint of it, which is
// what this samples.
//
// Sampled rather than diffed because what is asserted is which colour landed
// where, and a pixel count cannot say that. Neutral is in the set and is not
// a special case: it wears the platform's grey.
func TestTheDiscWearsItsStatusFillAndForeground(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, sc := range goldenSchemes {
		for _, st := range goldenStatuses {
			t.Run(sc.name+" "+st.label, func(t *testing.T) {
				fill := badge.Fill(sc.p, st.status)
				fg := badge.Foreground(sc.p)
				page := sc.p.ControlBackground

				// The circle alone, so the centre pixel is the fill and
				// nothing has been drawn over it.
				plain := badge.Render(shaper, "", blank, st.status,
					sc.p, tokens.Spacing, tokens.Radius, style, discState())
				size := measure(t, plain)
				img := golden.Capture(t, goldenSize, onPage(sc.p, plain))
				if img == nil {
					return // headless unavailable; Capture called t.Skip
				}
				if got := badgePixel(t, img, size.X/2, size.Y/2); !sameColour(got, fill) {
					t.Errorf("the disc's middle pixel is %v, want the status's fill %v", got, fill)
				}
				// The corner is outside an inscribed circle, so what is
				// there is the surface: a square fill here would be the
				// worded badge's container drawn on the wrong utterance.
				if got := badgePixel(t, img, 0, 0); !sameColour(got, page) {
					t.Errorf("the disc badge's top-left pixel is %v, want the page %v — the disc is not a circle", got, page)
				}
				// And nothing overran the box the badge reported.
				if got := badgePixel(t, img, -1, size.Y/2); !sameColour(got, page) {
					t.Errorf("the pixel before the disc's left edge is %v, want the page %v", got, page)
				}
				if got := badgePixel(t, img, size.X, size.Y/2); !sameColour(got, page) {
					t.Errorf("the pixel after the disc's right edge is %v, want the page %v", got, page)
				}

				// The sign on it, at the same middle pixel.
				signed := badge.Render(shaper, "", dot, st.status,
					sc.p, tokens.Spacing, tokens.Radius, style, discState())
				img = golden.Capture(t, goldenSize, onPage(sc.p, signed))
				if img == nil {
					return
				}
				if got := badgePixel(t, img, size.X/2, size.Y/2); !sameColour(got, fg) {
					t.Errorf("the sign on the disc is %v, want the status's foreground over its fill %v", got, fg)
				}
				if fg == fill {
					t.Errorf("the %s disc's fill and foreground are the same colour %v: nothing on it can be read", st.label, fill)
				}
			})
		}
	}
}

// TestTheBareSignStaysTheDefault is the ruling that the disc is asked for and
// never assumed. The zero [badge.RenderState] draws what it drew before — no
// fill anywhere in a glyph badge's box — and the two renders differ, so a disc
// that quietly became the default would fail here rather than in a golden
// someone regenerated.
func TestTheBareSignStaysTheDefault(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, sc := range goldenSchemes {
		bare := badge.Render(shaper, "", check, badge.Success,
			sc.p, tokens.Spacing, tokens.Radius, style, badge.RenderState{})
		size := measure(t, bare)
		img := golden.Capture(t, goldenSize, onPage(sc.p, bare))
		if img == nil {
			return // headless unavailable; Capture called t.Skip
		}
		page := sc.p.ControlBackground
		for _, p := range []image.Point{{}, {X: size.X - 1}, {Y: size.Y - 1}, {X: size.X / 2, Y: 0}} {
			if got := badgePixel(t, img, p.X, p.Y); !sameColour(got, page) {
				t.Errorf("%s: the default glyph badge's pixel %v is %v, want the page %v — it has grown a disc unasked",
					sc.name, p, got, page)
			}
		}
		disc := badge.Render(shaper, "", check, badge.Success,
			sc.p, tokens.Spacing, tokens.Radius, style, discState())
		if n := golden.PixelDiff(img, golden.Capture(t, goldenSize, onPage(sc.p, disc))); n == 0 {
			t.Errorf("%s: a disc badge is pixel-identical to a bare one", sc.name)
		}
	}
}

// TestALabelIgnoresTheDisc pins the documented answer to the case that is not
// a form: a badge with words already wears the container the disc would add,
// so asking for both changes nothing rather than nesting one inside the other.
func TestALabelIgnoresTheDisc(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, tc := range []struct {
		name  string
		label string
		glyph badge.Glyph
	}{
		{"word", "Passing", nil},
		{"word and sign", "Passing", check},
	} {
		t.Run(tc.name, func(t *testing.T) {
			render := func(s badge.RenderState) layout.Widget {
				return badge.Render(shaper, tc.label, tc.glyph, badge.Success,
					tokens.PlatformLight, tokens.Spacing, tokens.Radius, style, s)
			}
			plain := render(badge.RenderState{})
			asked := render(badge.RenderState{Disc: true})
			if got, want := measure(t, asked), measure(t, plain); got != want {
				t.Errorf("a labelled badge asked for a disc measured %v and one not asked %v", got, want)
			}
			a := golden.Capture(t, goldenSize, onPage(tokens.PlatformLight, plain))
			b := golden.Capture(t, goldenSize, onPage(tokens.PlatformLight, asked))
			if a == nil || b == nil {
				return // headless unavailable; Capture called t.Skip
			}
			if n := golden.PixelDiff(a, b); n != 0 {
				t.Errorf("a labelled badge drew %d pixels differently when asked for a disc: a label ignores it", n)
			}
		})
	}
}

// TestDiscGolden records the whole vocabulary as discs under both signs, in
// both schemes, on the content plane and in a dialog. Two signs across one row
// of statuses is the obligation drawn: the disc adds a field of the status's
// hue and the shapes still have to differ, because a reader who separates
// neither red nor green reads the check and the cross and nothing else.
func TestDiscGolden(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, sc := range goldenSchemes {
		name := "badge-" + sc.name + "-disc"
		t.Run(name, func(t *testing.T) {
			ws := make([]layout.Widget, 0, 2*len(goldenStatuses))
			for _, glyph := range []badge.Glyph{check, cross} {
				for _, st := range goldenStatuses {
					ws = append(ws, badge.Render(shaper, "", glyph, st.status,
						sc.p, tokens.Spacing, tokens.Radius, style, discState()))
				}
			}
			golden.Render(t, name, goldenSize, onPage(sc.p, row(ws...)))
		})
	}
}

// TestPropsCarryTheDiscToTheDrawing is the seam between the two paths: the
// live badge copies [badge.Props.Disc] into the state it draws in, so a caller
// who asked for a disc gets one and not the bare sign the field's zero value
// would leave. The box is unchanged either way, which is why size alone cannot
// catch a field that never arrived.
func TestPropsCarryTheDiscToTheDrawing(t *testing.T) {
	shaper := defaultShaper(t)
	props := badge.Props{Glyph: check, Status: badge.Success, Shaper: shaper,
		Description: "The key is live"}
	bare := live(t, props)
	props.Disc = true
	disc := live(t, props)

	if got, want := measure(t, disc), measure(t, bare); got != want {
		t.Errorf("a live disc badge measured %v and a live bare one %v: the disc must not move the box", got, want)
	}
	a := golden.Capture(t, goldenSize, onPage(tokens.PlatformLight, bare))
	b := golden.Capture(t, goldenSize, onPage(tokens.PlatformLight, disc))
	if a == nil || b == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(a, b); n == 0 {
		t.Error("a live badge asked for a disc drew the bare sign: Props.Disc never reached the drawing")
	}
}

// TestASignFillingItsBoxStaysInsideTheDisc is the rule the disc's sign size
// exists for, measured at its worst case: the sign is handed the square
// inscribed in the circle, so a painter that fills that square corner to
// corner still lands inside the fill and never on the rim.
//
// The fresh-eyes pass on the gallery caught the version this replaces. Handed
// the disc's whole square, a check drawn to the Glyph contract — spanning most
// of the box — put its top-right tip on the antialiased rim and read as
// clipped, at every call site that followed the contract.
func TestASignFillingItsBoxStaysInsideTheDisc(t *testing.T) {
	shaper := defaultShaper(t)
	style := badgeStyle()
	for _, sc := range goldenSchemes {
		t.Run(sc.name, func(t *testing.T) {
			w := badge.Render(shaper, "", square, badge.Error,
				sc.p, tokens.Spacing, tokens.Radius, style, discState())
			size := measure(t, w)
			img := golden.Capture(t, goldenSize, onCard(sc.p, w))
			if img == nil {
				return // headless unavailable; Capture called t.Skip
			}
			fg := badge.Foreground(sc.p)
			signBox, ok := exactBox(img, fg, size)
			if !ok {
				t.Fatalf("no pixel of the sign came back in its own foreground %v", fg)
			}
			// Centred on whole pixels: the sign's box and the badge's box
			// share a centre, so their margins match on both sides.
			if l, r := signBox.Min.X, size.X-signBox.Max.X; l != r {
				t.Errorf("the sign has %d px of disc on its left and %d on its right: it is not centred on whole pixels", l, r)
			}
			if top, bot := signBox.Min.Y, size.Y-signBox.Max.Y; top != bot {
				t.Errorf("the sign has %d px of disc above it and %d below: it is not centred on whole pixels", top, bot)
			}
			// Every corner of the sign's box inside the circle, which is
			// what "inscribed" means and what a rim collision breaks.
			radius := float64(size.X) / 2
			centreX, centreY := centre(image.Rectangle{Max: size})
			for _, p := range []image.Point{
				signBox.Min,
				{X: signBox.Max.X, Y: signBox.Min.Y},
				{X: signBox.Min.X, Y: signBox.Max.Y},
				signBox.Max,
			} {
				dx, dy := float64(p.X)-centreX, float64(p.Y)-centreY
				if got := math.Hypot(dx, dy); got > radius {
					t.Errorf("the sign's corner %v is %.2f px from the disc's centre and the disc's radius is %.2f: a sign that fills its box lands on the rim",
						p, got, radius)
				}
			}
		})
	}
}
