package button_test

import (
	"context"
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/button"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/internal/focus"
	tcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// crossIcon is a deterministic "×" glyph painter — two diagonal clip.Stroke
// lines filling a sizePx×sizePx box — used to exercise the icon-only button.
// Vector strokes (no font/SVG rasterisation) keep golden output stable.
func crossIcon(gtx layout.Context, sizePx int, col color.NRGBA) {
	w := float32(sizePx)
	stroke := float32(gtx.Dp(2))
	if stroke < 1 {
		stroke = 1
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(w, w))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())

	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(w, 0))
	p.LineTo(f32.Pt(0, w))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper. See
// AGENTS.md.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// ---- Golden-image tests ----

// TestButtonGolden records or diffs the four canonical button states:
// light-normal, dark-normal, light-focused, light-pressed.
func TestButtonGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	// Zero corner radius keeps the edges sharp: anti-aliased rounded corners
	// vary slightly between GPU context initialisations. The label is real
	// text — F4.2 split the shaper so a golden pins its faces explicitly, and
	// Latin text in the pinned Roboto faces rasterises identically everywhere,
	// so the old empty label bought nothing and hid every typography
	// regression (F4.4).
	sharpRadius := tokens.RadiusScale{} // all zeros → sharp corners, no AA
	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  button.RenderState
	}{
		{"light-normal", tokens.DefaultLight, button.RenderState{}},
		{"dark-normal", tokens.DefaultDark, button.RenderState{}},
		{"light-focused", tokens.DefaultLight, button.RenderState{Focused: true}},
		{"light-pressed", tokens.DefaultLight, button.RenderState{Pressed: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := button.Render(
				shaper, "Save Changes",
				tc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// ---- Emphasis registers (G0A.1) ----

// onGround paints the whole canvas in the scheme's app-background pin before
// drawing w over it. Every emphasis golden is recorded this way, and it has
// to be: a ghost button paints no ground of its own, so against the headless
// window's own clear colour its register would be indistinguishable from a
// component that failed to draw. The background pin is the system's default
// ground — the step a tinted fill is a card over — so it is also the ground
// on which the three registers separate the way the scale intends.
func onGround(c tokens.ColorTokens, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// The emphasis matrix: every register, in every interaction state, at both
// densities. Names come from Emphasis.String, so the stored files read
// emph-ghost-compact-hovered.png and the golden names cannot drift from the
// vocabulary the rest of the design system uses.
var (
	emphasisRegisters = []button.Emphasis{button.Filled, button.Tonal, button.Ghost}

	emphasisDensities = []struct {
		name string
		d    tokens.Density
	}{
		{"comfortable", tokens.Comfortable},
		{"compact", tokens.Compact},
	}

	emphasisStates = []struct {
		name string
		s    button.RenderState
	}{
		{"normal", button.RenderState{}},
		{"hovered", button.RenderState{Hovered: true}},
		{"focused", button.RenderState{Focused: true}},
		{"pressed", button.RenderState{Pressed: true}},
		{"disabled", button.RenderState{Disabled: true}},
	}
)

// TestButtonEmphasisGolden records or diffs the text button across the whole
// register × state × density matrix. The pre-existing goldens are untouched
// by design: they are the proof that the zero register still renders exactly
// today's filled button, so this test stores its own files under an emph-
// prefix rather than re-recording theirs.
func TestButtonEmphasisGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	sharpRadius := tokens.RadiusScale{} // all zeros → sharp corners, no AA
	colors := tokens.DefaultLight

	for _, reg := range emphasisRegisters {
		for _, den := range emphasisDensities {
			for _, st := range emphasisStates {
				state := st.s
				state.Emphasis = reg
				name := "emph-" + reg.String() + "-" + den.name + "-" + st.name
				t.Run(name, func(t *testing.T) {
					w := button.Render(
						shaper, "Save Changes",
						colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, den.d,
						state,
					)
					golden.Render(t, name, size, onGround(colors, w))
				})
			}
		}
	}
}

// TestIconButtonEmphasisGolden records or diffs the icon-only button in every
// register, idle and focused, at both densities: icon-only composes with the
// emphasis axis rather than being a register of its own. The focused row is
// the visible half of the rule that the focus ring does not scale down with
// emphasis — the ring is the same 2 dp of the same colour on the ghost square
// as on the filled one.
func TestIconButtonEmphasisGolden(t *testing.T) {
	size := image.Pt(60, 60)
	sharpRadius := tokens.RadiusScale{}
	colors := tokens.DefaultLight

	states := []struct {
		name string
		s    button.RenderState
	}{
		{"normal", button.RenderState{}},
		{"focused", button.RenderState{Focused: true}},
	}
	for _, reg := range emphasisRegisters {
		for _, den := range emphasisDensities {
			for _, st := range states {
				state := st.s
				state.Emphasis = reg
				name := "emph-icon-" + reg.String() + "-" + den.name + "-" + st.name
				t.Run(name, func(t *testing.T) {
					w := button.RenderIcon(crossIcon, colors, tokens.Spacing, sharpRadius, den.d, state)
					golden.Render(t, name, size, onGround(colors, w))
				})
			}
		}
	}
}

// onLevel2Ground paints the whole canvas in the level-2 surface fill — the
// raised storey patterns/modal's dialog occupies — before drawing w over it.
// It is the local ground of the I3.1 goldens: a ghost hosted on a raised
// surface sits on that surface's own step, not the window ground.
func onLevel2Ground(c tokens.ColorTokens, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, c.SurfaceAt(tokens.Level2), clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// TestGhostIconButtonWashesAboveRaisedGroundGolden records or diffs the
// icon-only ghost hovering on a level-2 ground — the modal-close
// configuration whose hover wash the window-ground walk resolved to the
// very colour it sits on. The stored image is the fix made visible: a wash
// square that reads against its raised ground.
func TestGhostIconButtonWashesAboveRaisedGroundGolden(t *testing.T) {
	size := image.Pt(60, 60)
	colors := tokens.DefaultLight
	w := button.RenderIcon(crossIcon, colors, tokens.Spacing, tokens.RadiusScale{}, tokens.Comfortable,
		button.RenderState{Emphasis: button.Ghost, Ground: tokens.Level2, Hovered: true})
	golden.Render(t, "emph-icon-ghost-level2-hovered", size, onLevel2Ground(colors, w))
}

// TestGhostWashDiffersFromItsRaisedGround is the I3.1 defect assertion in
// pixels: a ghost hosted on a level-2 surface must hover in a wash that
// differs from the ground it sits on. Before the wash walked from the local
// ground it resolved to neutral 300 — exactly the level-2 fill — and this
// test's sampled pixel equalled the ground: an invisible hover. The wash
// must now be the host surface's own one-rung walk, which since ADR-022 is
// asked of the storey rather than of a ramp index: tokens.ColorTokens.StateAt
// walks the neutral ladder from whatever the storey is actually filled with,
// and the light level-2 fill is off the ramp entirely.
func TestGhostWashDiffersFromItsRaisedGround(t *testing.T) {
	size := image.Pt(60, 60)
	colors := tokens.DefaultLight
	img := golden.Capture(t, size, onLevel2Ground(colors, button.RenderIcon(
		crossIcon, colors, tokens.Spacing, tokens.RadiusScale{}, tokens.Comfortable,
		button.RenderState{Emphasis: button.Ghost, Ground: tokens.Level2, Hovered: true},
	)))
	if img == nil {
		return // headless unavailable; Capture called t.Skip
	}

	// A pixel inside the 36 dp button square but left of the glyph box
	// (which is inset by PaddingY, 8 dp): background wash, no glyph stroke,
	// no ring. Opaque fills: NRGBA == RGBA byte for byte.
	at := img.RGBAAt(3, 18)
	ground := colors.SurfaceAt(tokens.Level2)
	if at == (color.RGBA{R: ground.R, G: ground.G, B: ground.B, A: ground.A}) {
		t.Fatalf("hover wash at (3,18) equals the level-2 ground %v: the ghost wash is invisible on its host surface", ground)
	}
	want := colors.StateAt(tokens.Level2, tokens.StateHover)
	if at != (color.RGBA{R: want.R, G: want.G, B: want.B, A: want.A}) {
		t.Errorf("hover wash at (3,18) = %v, want the level-2 surface's own one-rung walk %v", at, want)
	}
}

// TestEmphasisRegistersAreVisuallyDistinct confirms the three registers are
// three different pictures. Without it the matrix above could record the same
// filled button thirty times and still pass on every future run.
func TestEmphasisRegistersAreVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	colors := tokens.DefaultLight

	shot := func(e button.Emphasis) *image.RGBA {
		return golden.Capture(t, size, onGround(colors, button.Render(
			shaper, "Click me",
			colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
			button.RenderState{Emphasis: e},
		)))
	}
	filled, tonal, ghost := shot(button.Filled), shot(button.Tonal), shot(button.Ghost)
	if filled == nil || tonal == nil || ghost == nil {
		return // headless unavailable; Capture called t.Skip
	}
	for _, p := range []struct {
		name string
		a, b *image.RGBA
	}{
		{"filled vs tonal", filled, tonal},
		{"tonal vs ghost", tonal, ghost},
		{"filled vs ghost", filled, ghost},
	} {
		if n := golden.PixelDiff(p.a, p.b); n == 0 {
			t.Errorf("%s render identically; the emphasis register changed nothing", p.name)
		}
	}
}

// TestGhostRestsTransparent confirms the quietest register paints no ground:
// a ghost button at rest is pixel-identical to the bare surface it sits on,
// everywhere except where its label is.
func TestGhostRestsTransparent(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	colors := tokens.DefaultLight

	bare := golden.Capture(t, size, onGround(colors, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}))
	ghost := golden.Capture(t, size, onGround(colors, button.Render(
		shaper, "Click me",
		colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{Emphasis: button.Ghost},
	)))
	filled := golden.Capture(t, size, onGround(colors, button.Render(
		shaper, "Click me",
		colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{},
	)))
	if bare == nil || ghost == nil || filled == nil {
		return
	}
	// The label is roughly a tenth of a 300×60 canvas; the filled ground is
	// most of it. The ghost must be far closer to the bare surface than the
	// filled button is — and it must still draw a label.
	ghostDiff := golden.PixelDiff(bare, ghost)
	filledDiff := golden.PixelDiff(bare, filled)
	if ghostDiff == 0 {
		t.Error("a ghost button drew nothing at all; the label must still be there")
	}
	if ghostDiff*4 >= filledDiff {
		t.Errorf("ghost differs from the bare surface in %d px against the filled button's %d: the ghost is painting a ground", ghostDiff, filledDiff)
	}
}

// ---- A pinned fill on the Filled register (AJ1.1) ----

// pinnedFill and pinnedInk are a pair no scheme carries: a fixed red of the
// kind a caller pins when the meaning of an action, rather than the palette,
// chooses its colour, and the ink that reads over it. They are ordinary
// colour values on purpose — the point of the pair is that nothing in the
// theme decides them.
var (
	pinnedFill = color.NRGBA{0xb3, 0x26, 0x1e, 0xff}
	pinnedInk  = color.NRGBA{0xff, 0xff, 0xff, 0xff}
)

// TestPinnedFillGolden records or diffs the pinned filled button through
// every interaction state in both schemes. Both schemes because that is
// where the pair earns its keep: the two tiles hold the same red at rest
// while the stock filled goldens beside them hold two different violets, and
// the walked states are the same red taken toward each scheme's own 900 end
// — down in light, up in dark, which is what "the register's treatments,
// applied to the caller's colours" looks like in pixels.
func TestPinnedFillGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	sharpRadius := tokens.RadiusScale{} // all zeros → sharp corners, no AA

	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		for _, st := range emphasisStates {
			state := st.s
			state.Fill, state.OnFill = pinnedFill, pinnedInk
			name := "pin-" + sc.name + "-" + st.name
			t.Run(name, func(t *testing.T) {
				w := button.Render(
					shaper, "Delete",
					sc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
					state,
				)
				golden.Render(t, name, size, onGround(sc.colors, w))
			})
		}
	}
}

// TestUnpinnedFillDrawsTheStockButton is the zero-value proof in pixels: a
// state that names no pin, or only half of one, renders the filled button
// byte for byte as it rendered before the pair existed. It is the assertion
// the pre-existing goldens make across the whole matrix, made here against
// the two half-written pins those goldens cannot reach.
func TestUnpinnedFillDrawsTheStockButton(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	colors := tokens.DefaultLight

	shot := func(s button.RenderState) *image.RGBA {
		return golden.Capture(t, size, onGround(colors, button.Render(
			shaper, "Delete",
			colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
			s,
		)))
	}
	stock := shot(button.RenderState{})
	if stock == nil {
		return // headless unavailable; Capture called t.Skip
	}
	for _, c := range []struct {
		name string
		s    button.RenderState
	}{
		{"a fill with no ink", button.RenderState{Fill: pinnedFill}},
		{"an ink with no fill", button.RenderState{OnFill: pinnedInk}},
		{"a transparent pair", button.RenderState{Fill: color.NRGBA{R: 0xb3}, OnFill: color.NRGBA{R: 0xff}}},
	} {
		if n := golden.PixelDiff(stock, shot(c.s)); n != 0 {
			t.Errorf("%s moved %d pixels; an unset pair must leave the register exactly where it was", c.name, n)
		}
	}
	// The control: a whole pair does move the picture, so the assertions
	// above are about the pair being unset rather than about it being inert.
	if n := golden.PixelDiff(stock, shot(button.RenderState{Fill: pinnedFill, OnFill: pinnedInk})); n == 0 {
		t.Error("a pinned pair rendered the stock button; the pin reached nothing")
	}
}

// TestPinnedFillCarriesARingThatReadsOnIt holds the half of the register a
// pinned fill could quietly break. The focus ring is not a fixed colour: it
// is the primary rung that clears the non-text floor against the ground it
// circles, and the ground it circles is now the caller's. So the ring must
// be chosen against the pin — and drawn in that colour — or a keyboard user
// loses the button on the one action that most needs confirming.
func TestPinnedFillCarriesARingThatReadsOnIt(t *testing.T) {
	size := image.Pt(60, 60)
	side := int(tokens.Comfortable.ControlHeight) // 1 px per dp in the harness
	w := int(focus.Width)

	for _, scheme := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		ring := focus.Ring(scheme.colors, pinnedFill)
		if got := tcolor.ContrastRatio(ring, pinnedFill); got < focus.Floor {
			t.Errorf("%s: ring %v measures %.2f:1 against the pinned fill %v",
				scheme.name, ring, got, pinnedFill)
		}
		img := golden.Capture(t, size, onGround(scheme.colors, button.RenderIcon(
			crossIcon, scheme.colors, tokens.Spacing, tokens.RadiusScale{}, tokens.Comfortable,
			button.RenderState{Fill: pinnedFill, OnFill: pinnedInk, Focused: true},
		)))
		if img == nil {
			return // headless unavailable; Capture called t.Skip
		}
		// A pixel on the ring's left band, clear of both corners: the band
		// spans w to 2w inside the button's own edge.
		if at := img.RGBAAt(w, side/2); !nearlyEqual(at, ring) {
			t.Errorf("%s: ring pixel at (%d,%d) = %v, want the rung measured against the pin %v",
				scheme.name, w, side/2, at, ring)
		}
	}
}

// ringGround is the ground a focused button's ring circles, per register:
// the register's own resting background, and — for the ghost, which paints
// none — the host surface showing through it. It is the test's own copy of
// the rule drawButton applies, kept here so the assertion below measures the
// ring against a ground stated independently of the code that painted it.
func ringGround(c tokens.ColorTokens, e button.Emphasis) color.NRGBA {
	switch e {
	case button.Tonal:
		return c.StateColor(tokens.RolePrimary, 200, tokens.StateFocus)
	case button.Ghost:
		return c.Ramps.Neutral.Step(200) // the level-1 surface a ghost assumes
	default:
		return c.SolidStateColor(tokens.RolePrimary, tokens.StateFocus)
	}
}

// TestFocusRingIsTheSameRingInEveryRegister is the pixel proof of the rule
// that keyboard visibility does not scale down with emphasis: the ring is the
// same shape, in the same place, at the same width in all three registers, and
// in each of them it reaches the non-text contrast floor against the ground it
// circles. So a ghost button's ring is neither thinner, dimmer nor smaller
// than a filled one's.
//
// What the registers do not share is the rung, and asserting one flat colour
// across all three is what this test used to do. It cannot be asked for and
// have the ring read as well: a filled button's ring circles the primary fill
// and a ghost's circles the surface, and no single rung of the primary ramp
// clears 3:1 against both. Sameness belongs to the ring's geometry; the rung
// belongs to the ground.
//
// The geometry is stated here rather than compared between registers — the
// outermost ring.Width dp of the button's own square, and nothing outside it
// — so "the same ring" is a claim about a band written down once and held by
// all three, and the ring's containment in the footprint is held too. The four
// corner pixels are excused: a stroke's corner is anti-aliased against
// whatever is behind it, which is a different colour in every register.
//
// Both schemes, because a light scheme's ring walks down its ramp from the
// mid-value rung and a dark scheme's walks up.
func TestFocusRingIsTheSameRingInEveryRegister(t *testing.T) {
	size := image.Pt(60, 60)
	side := int(tokens.Comfortable.ControlHeight) // 1 px per dp in the harness
	w := int(focus.Width)

	// onBand reports whether p is in the ring's band — the square annulus
	// spanning w to 2w inside the button's own edge, the ring held its own
	// width clear of that edge — and whether it is one of the anti-aliased
	// corners the colour check excuses.
	onBand := func(p image.Point) (band, corner bool) {
		if p.X < 0 || p.Y < 0 || p.X >= side || p.Y >= side {
			return false, false
		}
		inX := p.X >= w && p.X < side-w
		inY := p.Y >= w && p.Y < side-w
		if !inX || !inY {
			return false, false // the clear gap, or the surface beyond it
		}
		edgeX := p.X < 2*w || p.X >= side-2*w
		edgeY := p.Y < 2*w || p.Y >= side-2*w
		return edgeX || edgeY, edgeX && edgeY
	}

	for _, scheme := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		colors := scheme.colors
		for _, e := range []button.Emphasis{button.Filled, button.Tonal, button.Ghost} {
			// The rung this register's ring lands on must reach the floor
			// against the ground it circles.
			ground := ringGround(colors, e)
			ring := focus.Ring(colors, ground)
			if got := tcolor.ContrastRatio(ring, ground); got < focus.Floor {
				t.Errorf("%s %s: ring %v measures %.2f:1 against the ground it circles %v",
					scheme.name, e, ring, got, ground)
			}

			img := golden.Capture(t, size, onGround(colors, button.RenderIcon(
				crossIcon, colors, tokens.Spacing, tokens.RadiusScale{}, tokens.Comfortable,
				button.RenderState{Emphasis: e, Focused: true},
			)))
			missing, leaked := 0, 0
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					p := image.Pt(x, y)
					band, corner := onBand(p)
					isRing := nearlyEqual(img.RGBAAt(x, y), ring)
					switch {
					case band && !corner && !isRing:
						missing++
					case !band && isRing:
						leaked++
					}
				}
			}
			if missing > 0 {
				t.Errorf("%s %s: %d pixels of the ring's band are not the ring colour %v",
					scheme.name, e, missing, ring)
			}
			if leaked > 0 {
				t.Errorf("%s %s: %d pixels in the ring colour %v fall outside the ring's band",
					scheme.name, e, leaked, ring)
			}
		}
	}
}

// nearlyEqual reports whether a captured pixel is the given token colour, to
// within the three units per channel the GPU's own rounding moves a flat fill
// by: a solid band comes back with its last row or column a step or three off
// in one channel. Nothing else drawn on these canvases is within an order of
// that distance from the ring, so the slack costs the assertion nothing.
func nearlyEqual(got color.RGBA, want color.NRGBA) bool {
	const slack = 3
	off := func(a, b uint8) bool {
		if a > b {
			return a-b > slack
		}
		return b-a > slack
	}
	return !off(got.R, want.R) && !off(got.G, want.G) && !off(got.B, want.B) && got.A == want.A
}

// TestGhostIconButtonKeepsFullHitTarget is the rule the modal's close button
// depends on next: quieting a button's colours must not shrink what the
// pointer can hit. A ghost icon button draws a 36 dp square and still accepts
// a click at y=38, inside the 44 dp floor and outside the visual.
func TestGhostIconButtonKeepsFullHitTarget(t *testing.T) {
	var clicked int
	w := materialize(t, button.Button(rx.Of(theme.Default()), button.Props{
		Icon:        crossIcon,
		Description: "close",
		Emphasis:    button.Ghost,
		OnClick:     func(_ layout.Context) { clicked++ },
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(120, 120)
	drive := func() layout.Dimensions {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(size),
			Ops:         ops,
			Source:      r.Source(),
		}
		dims := w(gtx)
		r.Frame(ops)
		return dims
	}

	dims := drive()
	side := int(tokens.Comfortable.ControlHeight)
	if dims.Size != image.Pt(side, side) {
		t.Fatalf("ghost icon button visual = %v, want %dx%d (the register must not resize the control)", dims.Size, side, side)
	}

	// The hit rect is 44 px centred on the 36 px square: -4..40 on both axes.
	pos := f32.Pt(18, 38)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if clicked != 1 {
		t.Errorf("click in the hit slop below a ghost icon button: OnClick fired %d times, want 1", clicked)
	}
}

// ---- Density (E1.3) ----

// densityTheme returns a theme whose density is d, with sharp corners for
// golden determinism (anti-aliased rounded corners vary between GPU context
// initialisations).
func densityTheme(d tokens.Density) theme.Theme {
	th := theme.Default()
	th.Density = rx.Of(d)
	th.Radius = rx.Of(tokens.RadiusScale{})
	return th
}

// materialize subscribes to a component observable and returns its last
// emitted widget.
func materialize(t *testing.T, obs rx.Observable[layout.Widget]) layout.Widget {
	t.Helper()
	var w layout.Widget
	if err := obs.Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("component did not emit a widget")
	}
	return w
}

// TestButtonCompactGolden records or diffs the text button at tokens.Compact
// through the live pipeline (the density observable is the only way in; the
// static Render path is pinned to Comfortable). The 28 dp bar against the
// 36 dp light-normal golden is the density divergence landing.
func TestButtonCompactGolden(t *testing.T) {
	w := materialize(t, button.Button(rx.Of(densityTheme(tokens.Compact)), button.Props{
		Label: "Save Changes",
		// The live path would otherwise take the theme's fallback Shaper,
		// which resolves against the machine's fonts. A golden pins its faces.
		Shaper: defaultShaper(t),
	}))
	golden.Render(t, "light-compact", image.Pt(300, 60), w)
}

// ---- Accessibility tests ----

// TestButtonVisualHeightIsControlHeight checks the drawn button is exactly the
// density's control height (E1.3: 36 dp Comfortable), not the old 44 dp — the
// 44 dp figure is a pointer-target floor, verified separately below.
//
// Comfortable is the density where the floor and the content box agree:
// LabelLarge's 20 dp line box plus 2×8 dp of padding is 36, which is
// ComfortableControlHeight exactly. The two readings are pulled apart at
// Compact by TestCompactButtonClearsTheControlHeightFloor.
func TestButtonVisualHeightIsControlHeight(t *testing.T) {
	shaper := defaultShaper(t)

	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(300, 120)),
		Ops:         &ops,
	}

	dims := button.Render(
		shaper, "OK",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{},
	)(gtx)

	want := int(tokens.Comfortable.ControlHeight) // 36 dp at 1:1 scale
	if dims.Size.Y != want {
		t.Errorf("button height = %d px, want %d px (ControlHeight at 1:1 scale)", dims.Size.Y, want)
	}
}

// TestCompactButtonClearsTheControlHeightFloor pins the answer F4.4 asked for:
// a Compact button draws taller than CompactControlHeight, and that is
// correct, because ControlHeight is a floor and not a height.
//
// The measurement that raised it found 29 px against a floor of 28 — the
// glyph ink box of 17 px plus 2×6 dp — and reproduced it with an empty label,
// so it was never text's doing. With the line box honoured the number is 32:
// LabelLarge's 20 dp line height plus the same 12 dp of padding. Both are over
// the floor; only the second is derivable from the tokens, which is why it is
// the one worth pinning.
//
// The label is deliberately empty. A control's height must not depend on which
// letters it happens to contain, and this is the assertion that says so.
func TestCompactButtonClearsTheControlHeightFloor(t *testing.T) {
	shaper := defaultShaper(t)

	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(300, 120)),
		Ops:         &ops,
	}

	style := tokens.DefaultTypography.LabelLarge
	d := tokens.Compact
	dims := button.Render(
		shaper, "",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, style, d,
		button.RenderState{},
	)(gtx)

	want := int(style.LineHeight + 2*d.PaddingY) // 20 + 12 = 32
	if dims.Size.Y != want {
		t.Errorf("compact button height = %d px, want %d px (LabelLarge line box %v + 2×PaddingY %v)",
			dims.Size.Y, want, style.LineHeight, d.PaddingY)
	}
	if floor := int(d.ControlHeight); dims.Size.Y <= floor {
		t.Errorf("compact button height = %d px, expected to clear the %d px floor; if it no longer does, density.go's table is stale",
			dims.Size.Y, floor)
	}
}

// TestButtonMinHitTarget checks the live button's pointer target extends to
// the 44 dp WCAG 2.5.5 floor even though the visual control is only 36 dp
// tall: a click below the visual bounds, inside the hit slop, activates it.
func TestButtonMinHitTarget(t *testing.T) {
	var clicked int
	w := materialize(t, button.Button(rx.Of(theme.Default()), button.Props{
		Label:   "OK",
		OnClick: func(_ layout.Context) { clicked++ },
		Shaper:  defaultShaper(t),
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 120)
	drive := func() layout.Dimensions {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(size),
			Ops:         ops,
			Source:      r.Source(),
		}
		dims := w(gtx)
		r.Frame(ops)
		return dims
	}

	dims := drive() // register the input area
	if dims.Size.Y != int(tokens.Comfortable.ControlHeight) {
		t.Fatalf("visual height = %d px, want %d", dims.Size.Y, int(tokens.Comfortable.ControlHeight))
	}

	// The hit rect is 44 px centred on the 36 px visual: -4..40. Click at
	// y=38 — outside the visual, inside the hit slop.
	pos := f32.Pt(150, 38)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if clicked != 1 {
		t.Errorf("click in the hit slop below the visual: OnClick fired %d times, want 1", clicked)
	}
}

// TestButtonDisabledIsVisuallyDistinct confirms disabled state produces
// different pixels from enabled state.
func TestButtonDisabledIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgEnabled := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{},
	))
	imgDisabled := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{Disabled: true},
	))

	if imgEnabled == nil || imgDisabled == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(imgEnabled, imgDisabled); n == 0 {
		t.Error("disabled and enabled buttons render identically; expected visual difference")
	}
}

// TestButtonFocusRingIsVisuallyDistinct confirms focused state renders
// differently from normal state (the focus ring must add pixels).
func TestButtonFocusRingIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgNormal := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{},
	))
	imgFocused := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and normal buttons render identically; expected focus ring pixels to differ")
	}
}

// TestButtonPressedIsVisuallyDistinct confirms pressed state renders
// differently from normal state.
func TestButtonPressedIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgNormal := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{},
	))
	imgPressed := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{Pressed: true},
	))

	if imgNormal == nil || imgPressed == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgPressed); n == 0 {
		t.Error("pressed and normal buttons render identically; expected visual difference")
	}
}

// TestButtonHoveredIsVisuallyDistinct confirms hovered state renders
// differently from normal state.
func TestButtonHoveredIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	imgNormal := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{},
	))
	imgHovered := golden.Capture(t, size, button.Render(
		shaper, "Click me",
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
		button.RenderState{Hovered: true},
	))

	if imgNormal == nil || imgHovered == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgHovered); n == 0 {
		t.Error("hovered and normal buttons render identically; expected visual difference")
	}
}

// ---- Icon-only variant (GX.3) ----

// TestIconButtonGolden records or diffs the icon-only variant in its idle and
// focused states. Zero corner radius keeps edges sharp; the glyph is a
// clip.Stroke "×" so the render is deterministic.
func TestIconButtonGolden(t *testing.T) {
	size := image.Pt(60, 60)
	sharpRadius := tokens.RadiusScale{} // all zeros → sharp corners, no AA
	cases := []struct {
		name   string
		colors tokens.ColorTokens
		state  button.RenderState
	}{
		{"icon-light-normal", tokens.DefaultLight, button.RenderState{}},
		{"icon-light-focused", tokens.DefaultLight, button.RenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := button.RenderIcon(
				crossIcon,
				tc.colors, tokens.Spacing, sharpRadius, tokens.Comfortable,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// TestIconButtonVisualIsControlHeightSquare checks the icon-only button draws
// as a square the density's control height on a side (E1.3: 36 dp
// Comfortable). The 44 dp pointer-target floor is enforced by the live path's
// hit extension, exercised by TestButtonMinHitTarget and internal/hit.
func TestIconButtonVisualIsControlHeightSquare(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(120, 120)),
		Ops:         &ops,
	}
	dims := button.RenderIcon(
		crossIcon,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.Comfortable,
		button.RenderState{},
	)(gtx)

	want := int(tokens.Comfortable.ControlHeight)
	if dims.Size.X != want || dims.Size.Y != want {
		t.Errorf("icon button size = %v, want %dx%d px (ControlHeight square at 1:1 scale)", dims.Size, want, want)
	}
}

// TestIconButtonCompactGolden records or diffs the icon-only button at
// tokens.Compact through the live pipeline: a 28 dp square with a 16 dp glyph.
func TestIconButtonCompactGolden(t *testing.T) {
	w := materialize(t, button.Button(rx.Of(densityTheme(tokens.Compact)), button.Props{
		Icon:        crossIcon,
		Description: "close",
	}))
	golden.Render(t, "icon-light-compact", image.Pt(60, 60), w)
}

// TestIconButtonFocusRingIsVisuallyDistinct confirms the icon-only button's
// focused state renders differently from idle (the focus ring must add pixels).
func TestIconButtonFocusRingIsVisuallyDistinct(t *testing.T) {
	size := image.Pt(60, 60)

	imgNormal := golden.Capture(t, size, button.RenderIcon(
		crossIcon,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.Comfortable,
		button.RenderState{},
	))
	imgFocused := golden.Capture(t, size, button.RenderIcon(
		crossIcon,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, tokens.Comfortable,
		button.RenderState{Focused: true},
	))

	if imgNormal == nil || imgFocused == nil {
		return
	}
	if n := golden.PixelDiff(imgNormal, imgFocused); n == 0 {
		t.Error("focused and idle icon buttons render identically; expected focus ring pixels to differ")
	}
}

// TestButtonInjectedClickableFocusAndActivate proves the caller-owned-clickable
// path (GX.3): a container drives focus to the supplied *widget.Clickable via
// key.FocusCmd, and Space activation flows through it to OnClick. This is the
// mechanism patterns/modal's focus trap relies on for the close button.
func TestButtonInjectedClickableFocusAndActivate(t *testing.T) {
	shaper := defaultShaper(t)
	var clicked int
	var click widget.Clickable

	obs := button.Button(rx.Of(theme.Default()), button.Props{
		Icon:        crossIcon,
		Description: "Close",
		Clickable:   &click,
		OnClick:     func(_ layout.Context) { clicked++ },
		Shaper:      shaper,
	})
	var w layout.Widget
	if err := obs.Subscribe(context.Background(), func(next layout.Widget, _ error, done bool) {
		if !done && next != nil {
			w = next
		}
	}).Wait(); err != nil {
		t.Fatalf("Button subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("Button did not emit a widget")
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(44, 44)

	drive := func(cw layout.Widget) {
		ops.Reset()
		gtx := layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(size),
			Ops:         ops,
			Source:      r.Source(),
		}
		cw(gtx)
		r.Frame(ops)
	}

	// Frame 1: lay out the button (registers the clickable's focus filter),
	// then a container drives focus to the caller-owned tag.
	focusOnce := true
	composed := func(gtx layout.Context) layout.Dimensions {
		dims := w(gtx)
		if focusOnce {
			gtx.Execute(key.FocusCmd{Tag: &click})
			focusOnce = false
		}
		return dims
	}
	drive(composed)
	// Frame 2: focus is applied.
	drive(w)

	probe := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         new(op.Ops),
		Source:      r.Source(),
	}
	if !probe.Focused(&click) {
		t.Fatal("injected clickable not focused after key.FocusCmd; container-driven focus failed")
	}

	// Space while focused activates the button → OnClick fires through the
	// caller-owned clickable.
	r.Queue(
		key.Event{Name: key.NameSpace, State: key.Press},
		key.Event{Name: key.NameSpace, State: key.Release},
	)
	drive(w)
	if clicked != 1 {
		t.Errorf("Space activation: OnClick fired %d times, want 1", clicked)
	}
}
