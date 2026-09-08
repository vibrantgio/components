package picker_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/theme/tokens"
)

// defaultShaper pins the faces every measurement here draws with, so a
// comparison is of the drawing and not of whatever fonts the machine has.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// sharpRadius keeps every corner square. Anti-aliased rounded corners vary
// between GPU context initialisations, and every test below compares two
// renderings pixel for pixel.
var sharpRadius = tokens.RadiusScale{}

// rowHeight is the height a trigger and an option row each draw at: one
// BodyLarge line box plus the density's vertical padding, floored at the
// density's control height.
func rowHeight(d tokens.Density) int {
	h := int(tokens.DefaultTypography.BodyLarge.LineHeight + 2*d.PaddingY)
	if floor := int(d.ControlHeight); h < floor {
		h = floor
	}
	return h
}

// measure lays w out at an exact size and reports what it said it used.
func measure(t *testing.T, size image.Point, w layout.Widget) layout.Dimensions {
	t.Helper()
	var ops op.Ops
	return w(layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         &ops,
	})
}

// field is RenderField at the default light palette and comfortable density.
func field(t *testing.T, s picker.FieldState) layout.Widget {
	t.Helper()
	return picker.RenderField(defaultShaper(t), tokens.DefaultLight, tokens.Spacing,
		sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s)
}

// menu is RenderMenu at the same palette and density.
func menu(t *testing.T, s picker.MenuState) layout.Widget {
	t.Helper()
	return picker.RenderMenu(defaultShaper(t), tokens.DefaultLight, tokens.Spacing,
		tokens.DefaultTypography.BodyLarge, tokens.Comfortable, s)
}

var options = []string{"Alpha", "Beta", "Gamma"}

// TestFieldTriggerShowsTheValue is the single-choice contract in pixels: the
// closed trigger draws the option Selected names and nothing else about the
// list, so a picker holding the second of three is indistinguishable from one
// whose only option is that same string.
func TestFieldTriggerShowsTheValue(t *testing.T) {
	size := image.Pt(200, 44)
	ofThree := golden.Capture(t, size, field(t, picker.FieldState{Options: options, Selected: 1}))
	ofOne := golden.Capture(t, size, field(t, picker.FieldState{Options: []string{"Beta"}}))
	if n := golden.PixelDiff(ofThree, ofOne); n != 0 {
		t.Errorf("a closed trigger holding %q out of three differs from one holding it alone in %d pixels; the trigger shows the value and nothing else", options[1], n)
	}
}

// TestFieldTriggerHeightIsItsLineBoxOverTheFloor holds the trigger to the
// sizing rule every control in the system takes: a control height is a floor,
// not a height, so the trigger draws max(ControlHeight, line box + 2×PaddingY)
// — 40 dp comfortable, over the 36 dp floor. The 44 dp pointer floor is the
// live path's and is measured in picker_live_test.go.
func TestFieldTriggerHeightIsItsLineBoxOverTheFloor(t *testing.T) {
	for _, d := range []struct {
		name string
		d    tokens.Density
	}{{"comfortable", tokens.Comfortable}, {"compact", tokens.Compact}} {
		t.Run(d.name, func(t *testing.T) {
			w := picker.RenderField(defaultShaper(t), tokens.DefaultLight, tokens.Spacing,
				tokens.Radius, tokens.DefaultTypography.BodyLarge, d.d,
				picker.FieldState{Options: options})
			dims := measure(t, image.Pt(200, 200), w)
			if want := rowHeight(d.d); dims.Size.Y != want {
				t.Errorf("trigger height = %d px, want %d px", dims.Size.Y, want)
			}
		})
	}
}

// TestOpenFieldFloatsTheSharedMenuUnderItsTrigger is the one-surface contract:
// the menu an open field drops is [Menu], not a second drawing of it. The
// composite below draws the closed trigger, the standalone menu at the
// trigger's own height, and the edge the field draws around that menu's plane
// — and the open field has to be that image exactly.
//
// The edge is in the composite because it is NOT the menu's. A menu handed to
// a pattern is circled by that pattern's own surface and would wear two lines,
// so the plane's edge belongs to whoever put the plane there, which inline is
// the field. What the shared-surface contract holds is that the rows are one
// drawing; the edge around them is the field's frame and is asserted here as
// the third term rather than folded into either.
func TestOpenFieldFloatsTheSharedMenuUnderItsTrigger(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(200, row*(1+len(options)))

	open := golden.Capture(t, size, field(t, picker.FieldState{
		Open: true, Options: options, Selected: 1,
	}))
	composed := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		trigger := field(t, picker.FieldState{Options: options, Selected: 1})(gtx)
		off := op.Offset(image.Pt(0, trigger.Size.Y)).Push(gtx.Ops)
		rows := menu(t, picker.MenuState{Options: options, Selected: 1})(gtx)
		planeEdge(gtx, rows.Size)
		off.Pop()
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, trigger.Size.Y+rows.Size.Y)}
	})
	if n := golden.PixelDiff(open, composed); n != 0 {
		t.Errorf("an open field differs from its own trigger plus a standalone menu plus the plane's edge in %d pixels", n)
	}
}

// planeEdge redraws the field's own edge around a menu box, so the composition
// tests can name the three things an open field is made of instead of
// comparing it to two of them. It is the derivation the field draws, spelled
// once here: the neutral step that clears the graphic floor against the
// level-3 plane the line circles, one dp inside the box on all four sides.
func planeEdge(gtx layout.Context, size image.Point) {
	edge := tokens.DefaultLight.MarkOn(tokens.RoleNeutral,
		tokens.DefaultLight.SurfaceAt(tokens.Level3), tokens.GraphicFloor)
	w := gtx.Dp(1)
	for _, r := range []image.Rectangle{
		{Max: image.Pt(size.X, w)},
		{Min: image.Pt(0, size.Y-w), Max: size},
		{Max: image.Pt(w, size.Y)},
		{Min: image.Pt(size.X-w, 0), Max: size},
	} {
		paint.FillShape(gtx.Ops, edge, clip.Rect(r).Op())
	}
}

// TestOpenFieldReportsTheTriggerAlone is the floating-surface contract in
// measurements: the menu is deferred to the end of the frame and bounded by
// the window, so it takes no room from the container the trigger stands in.
// An open field reports what the closed one reports, either direction, and a
// container places the two identically.
//
// The second half is why that is not simply a field that stopped drawing: the
// band below the trigger, captured off an open field, is the standalone menu
// with the field's own plane edge around it, pixel for pixel. The menu still
// paints whole where the reported box no longer reaches.
func TestOpenFieldReportsTheTriggerAlone(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	closed := measure(t, image.Pt(200, 400), field(t, picker.FieldState{Options: options}))
	if closed.Size != (image.Pt(200, row)) {
		t.Fatalf("closed field measured %v, want the trigger's %v", closed.Size, image.Pt(200, row))
	}
	for _, d := range []struct {
		name string
		drop picker.Drop
	}{{"down", picker.DropDown}, {"up", picker.DropUp}} {
		open := measure(t, image.Pt(200, 400), field(t, picker.FieldState{
			Open: true, Drop: d.drop, Options: options,
		}))
		if open != closed {
			t.Errorf("dropping %s, an open field measured %v against the closed field's %v; the menu floats and asks its container for no room",
				d.name, open, closed)
		}
	}

	size := image.Pt(200, row*(1+len(options)))
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	onBg := func(w layout.Widget) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
			w(gtx)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}
	}
	open := golden.Capture(t, size, onBg(field(t, picker.FieldState{
		Open: true, Options: options, Selected: 1,
	})))
	plane := golden.Capture(t, size, onBg(func(gtx layout.Context) layout.Dimensions {
		off := op.Offset(image.Pt(0, row)).Push(gtx.Ops)
		rows := menu(t, picker.MenuState{Options: options, Selected: 1})(gtx)
		planeEdge(gtx, rows.Size)
		off.Pop()
		return rows
	}))
	painted := false
	for y := row; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			if a, b := px(open, x, y), px(plane, x, y); a != b {
				t.Fatalf("(%d,%d), below the box the open field reported, is %v and the menu's own drawing is %v", x, y, a, b)
			}
			if px(open, x, y) != bg {
				painted = true
			}
		}
	}
	if !painted {
		t.Error("nothing at all was drawn below the trigger; an open field that reports its trigger must still float its menu")
	}
}

// TestOpenFieldFloatsTheSharedMenuOverItsTrigger is the same contract the
// other way up: a field told there is no room below it floats the one surface
// ABOVE its trigger, over whatever the window laid out before it. The trigger
// stays where the caller put it — which is why the field below is laid out a
// menu's height down the capture, the room the upward menu takes back — and
// the plane's edge travels with the plane, so the composite carries it here
// too and the direction changes the side and nothing else.
func TestOpenFieldFloatsTheSharedMenuOverItsTrigger(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(200, row*(1+len(options)))

	open := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		off := op.Offset(image.Pt(0, row*len(options))).Push(gtx.Ops)
		defer off.Pop()
		return field(t, picker.FieldState{
			Open: true, Drop: picker.DropUp, Options: options, Selected: 1,
		})(gtx)
	})
	composed := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		rows := menu(t, picker.MenuState{Options: options, Selected: 1})(gtx)
		planeEdge(gtx, rows.Size)
		off := op.Offset(image.Pt(0, rows.Size.Y)).Push(gtx.Ops)
		// The same trigger, which means the same DIRECTION: its mark points
		// the way its menu goes, so a downward trigger under an upward menu
		// would be a different control.
		trigger := field(t, picker.FieldState{Options: options, Selected: 1, Drop: picker.DropUp})(gtx)
		off.Pop()
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, trigger.Size.Y+rows.Size.Y)}
	})
	if n := golden.PixelDiff(open, composed); n != 0 {
		t.Errorf("an upward field differs from a standalone menu plus its own trigger under it in %d pixels; the direction changes the order and nothing else", n)
	}
}

// TestOpenFieldDrawsItsPlaneEdgeBothWaysUp: the edge is the plane's, so the
// two directions draw the same one — an open field differs from the same field
// with the edge suppressed by the same pixels either way up. There is no
// suppression switch, so this measures it the way the drawing allows: the edge
// is the only thing the field adds to the composite, and the composite without
// it must differ, in both directions, by a count that is the same.
func TestOpenFieldDrawsItsPlaneEdgeBothWaysUp(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(200, row*(1+len(options)))
	counts := map[string]int{}

	for _, d := range []struct {
		name string
		drop picker.Drop
	}{{"down", picker.DropDown}, {"up", picker.DropUp}} {
		triggerY, menuY := 0, row
		if d.drop == picker.DropUp {
			triggerY, menuY = row*len(options), 0
		}
		open := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
			off := op.Offset(image.Pt(0, triggerY)).Push(gtx.Ops)
			defer off.Pop()
			return field(t, picker.FieldState{
				Open: true, Drop: d.drop, Options: options, Selected: 1,
			})(gtx)
		})
		bare := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
			off := op.Offset(image.Pt(0, triggerY)).Push(gtx.Ops)
			field(t, picker.FieldState{Options: options, Selected: 1, Drop: d.drop})(gtx)
			off.Pop()
			off = op.Offset(image.Pt(0, menuY)).Push(gtx.Ops)
			menu(t, picker.MenuState{Options: options, Selected: 1})(gtx)
			off.Pop()
			return layout.Dimensions{Size: size}
		})
		n := golden.PixelDiff(open, bare)
		if n == 0 {
			t.Errorf("%s: an open field is identical to its trigger and rows with no edge around the plane", d.name)
		}
		counts[d.name] = n
	}
	if counts["down"] != counts["up"] {
		t.Errorf("the plane's edge covers %d pixels dropping down and %d dropping up; it is the plane's edge and does not change with the direction",
			counts["down"], counts["up"])
	}
}

// TestCappedMenuIsTheCapAndScrolls: a cap is a plane height, so a menu given
// one draws exactly that and no more however many options it holds, while the
// same cap over a short list changes nothing — the rows are still shorter than
// it.
func TestCappedMenuIsTheCapAndScrolls(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	long := make([]string, 40)
	for i := range long {
		long[i] = "Option " + string(rune('A'+i%26))
	}
	cap := unit.Dp(row * 5)

	capped := measure(t, image.Pt(200, 2000), menu(t, picker.MenuState{Options: long, MaxHeight: cap}))
	if want := row * 5; capped.Size.Y != want {
		t.Errorf("a %d-row menu capped at %d px measured %d px tall, want the cap", len(long), want, capped.Size.Y)
	}
	uncapped := measure(t, image.Pt(200, 2000), menu(t, picker.MenuState{Options: long}))
	if want := row * len(long); uncapped.Size.Y != want {
		t.Errorf("an uncapped %d-row menu measured %d px tall, want every row: %d px", len(long), uncapped.Size.Y, want)
	}
	short := measure(t, image.Pt(200, 2000), menu(t, picker.MenuState{Options: options, MaxHeight: cap}))
	if want := row * len(options); short.Size.Y != want {
		t.Errorf("a %d-row menu under the same cap measured %d px tall, want its own %d px: a cap is a ceiling, not a height",
			len(options), short.Size.Y, want)
	}
}

// TestCappedFieldFloatsExactlyTheCap: the cap reaches the field the same way
// it reaches the menu, so an open field over a catalogue floats a plane
// exactly the cap tall rather than one forty rows tall — and measures its
// trigger either way, because the plane is not the field's to make room for.
// The extent is read off the pixels, which is where a floating plane's height
// is now visible at all.
func TestCappedFieldFloatsExactlyTheCap(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	long := make([]string, 40)
	for i := range long {
		long[i] = "Option " + string(rune('A'+i%26))
	}
	cap := unit.Dp(row * 5)
	dims := measure(t, image.Pt(200, 2000), field(t, picker.FieldState{
		Open: true, Options: long, MaxHeight: cap,
	}))
	if dims.Size.Y != row {
		t.Errorf("an open capped field measured %d px tall, want the trigger's %d px", dims.Size.Y, row)
	}

	size := image.Pt(200, row*8)
	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, menuCover, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return field(t, picker.FieldState{Open: true, Options: long, MaxHeight: cap})(gtx)
	})
	if last := row*6 - 1; px(img, 100, last) == menuCover {
		t.Errorf("y=%d, the last row inside the cap, was left unpainted; the plane is shorter than the cap it was given", last)
	}
	if past := row * 6; px(img, 100, past) != menuCover {
		t.Errorf("y=%d, one pixel past the trigger plus the cap, is %v and not the %v behind it; the plane is taller than its cap",
			past, px(img, 100, past), menuCover)
	}
}

// TestTriggerDrawsItsPromptApartFromItsValue is the empty-state contract: a
// field holding no value says so, in the prompt's own foreground, and the two prompts
// are two sentences — a field with options and none picked asks the reader to
// choose, and one with no options reports that there is nothing to choose.
// None of the three drawings may be the same image.
func TestTriggerDrawsItsPromptApartFromItsValue(t *testing.T) {
	size := image.Pt(200, 44)
	picked := golden.Capture(t, size, field(t, picker.FieldState{
		Options: options, Selected: 1, Placeholder: "Choose one…", NoOptions: "Nothing to pick",
	}))
	unpicked := golden.Capture(t, size, field(t, picker.FieldState{
		Options: options, Selected: -1, Placeholder: "Choose one…", NoOptions: "Nothing to pick",
	}))
	empty := golden.Capture(t, size, field(t, picker.FieldState{
		Selected: -1, Placeholder: "Choose one…", NoOptions: "Nothing to pick",
	}))
	if n := golden.PixelDiff(picked, unpicked); n == 0 {
		t.Error("a field holding a value draws the same image as one holding none")
	}
	if n := golden.PixelDiff(unpicked, empty); n == 0 {
		t.Error("a field asking the reader to choose draws the same image as one with nothing to choose")
	}

	// The prompt is drawn in the prompt's foreground, not the body's: the same
	// wording as a VALUE is a different image.
	asValue := golden.Capture(t, size, field(t, picker.FieldState{
		Options: []string{"Choose one…"},
	}))
	if n := golden.PixelDiff(unpicked, asValue); n == 0 {
		t.Error("a prompt is drawn in the same colour as a value; an unanswered field reads as answered")
	}
}

// TestTheTriggersMarkPointsTheWayItsMenuOpens: the triangle announces the
// motion the control has, so an upward field's trigger is not the downward
// one's — closed or open, since the direction is a property of the field and
// not of its open state. And the announcement is the mark's alone: every pixel
// the direction moves lies in the trailing quarter of the control, where the
// mark is drawn, so nothing else about the trigger turns over with it.
func TestTheTriggersMarkPointsTheWayItsMenuOpens(t *testing.T) {
	size := image.Pt(200, 44)
	down := golden.Capture(t, size, field(t, picker.FieldState{Options: options, Selected: 1}))
	up := golden.Capture(t, size, field(t, picker.FieldState{Options: options, Selected: 1, Drop: picker.DropUp}))

	moved, leading := 0, 0
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			if down.At(x, y) == up.At(x, y) {
				continue
			}
			moved++
			if x < size.X-size.X/4 {
				leading++
			}
		}
	}
	if moved == 0 {
		t.Error("an upward field's closed trigger draws the downward one's image: the mark says nothing about where the menu goes")
	}
	if leading != 0 {
		t.Errorf("the drop direction moved %d pixels outside the mark's own column (%d in all); the triangle turns over and nothing else does",
			leading, moved)
	}
}

// TestMenuWithNoOptionsIsNoSurface: an empty menu is not an empty plane, it is
// nothing at all, so a caller that opens one with nothing to offer paints no
// overlay over its content.
func TestMenuWithNoOptionsIsNoSurface(t *testing.T) {
	if dims := measure(t, image.Pt(200, 200), menu(t, picker.MenuState{})); dims.Size != (image.Point{}) {
		t.Errorf("menu with no options measured %v, want the zero size", dims.Size)
	}
}

// TestMenuSelectedRowIsDrawnApartFromTheRest: the selected row leaves the
// menu's own plane for the theme's inverse pair, so which row is selected
// changes the drawing. The ratios that pairing has to clear are measured in
// menu_contrast_test.go; this is the pixels saying the pairing is reached.
func TestMenuSelectedRowIsDrawnApartFromTheRest(t *testing.T) {
	size := image.Pt(200, rowHeight(tokens.Comfortable)*len(options))
	first := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0}))
	second := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 1}))
	none := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: -1}))
	if n := golden.PixelDiff(first, second); n == 0 {
		t.Error("a menu with its first row selected renders identically to one with its second selected")
	}
	if n := golden.PixelDiff(first, none); n == 0 {
		t.Error("a menu with a selected row renders identically to one with none; an out-of-range index selects nothing")
	}
}

// TestToolbarIsSizedToItsValue: the chrome variant's trigger is a control
// around its value, not a bar across its container — it clamps to what it is
// offered and otherwise reports its own width, at the density's control
// height.
func TestToolbarIsSizedToItsValue(t *testing.T) {
	w := picker.RenderToolbar(defaultShaper(t), "Anthropic · Opus 5", tokens.DefaultLight,
		tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, picker.ToolbarState{})
	dims := measure(t, image.Pt(400, 200), w)
	if dims.Size.X >= 400 {
		t.Errorf("toolbar measured %d px wide at a 400 px constraint: a picker's trigger is sized to its value", dims.Size.X)
	}
	if want := int(tokens.Comfortable.ControlHeight); dims.Size.Y != want {
		t.Errorf("toolbar height = %d px, want the density's control height %d px", dims.Size.Y, want)
	}
}

// TestMenuMarksTheRowUnderThePointer: hover is a row state, so exactly the
// hovered row changes and a menu told the pointer is over nothing is the menu
// at rest. Selection wins over hover, so the selected row under the pointer is
// the selected row.
func TestMenuMarksTheRowUnderThePointer(t *testing.T) {
	size := image.Pt(200, rowHeight(tokens.Comfortable)*len(options))
	rest := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0}))
	second := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0, Hovered: 2}))
	third := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0, Hovered: 3}))
	onSelected := golden.Capture(t, size, menu(t, picker.MenuState{Options: options, Selected: 0, Hovered: 1}))

	if n := golden.PixelDiff(rest, second); n == 0 {
		t.Error("a menu with its second row hovered renders identically to one with no row hovered")
	}
	if n := golden.PixelDiff(second, third); n == 0 {
		t.Error("hovering the second row renders identically to hovering the third; hover marks a row, not the menu")
	}
	if n := golden.PixelDiff(rest, onSelected); n != 0 {
		t.Errorf("hovering the selected row changed %d pixels; the standing answer outranks the transient state fill", n)
	}
}

// ---- Paint-order tests ----
//
// The defect these cover (feeds, 2026-09-06, on the popover): a floating
// surface drawn inline in its anchor's own paint order is painted over by
// every sibling the window lays out after that anchor's slot. The field's
// dropped menu is such a surface and takes the same idiom.

// menuCover is the opaque sibling painted after the field, over everything
// below the trigger's own band — the way a shell paints its main column after
// the row the field stands in.
var menuCover = color.NRGBA{R: 255, G: 0, B: 255, A: 255}

// coveredField lays the field out and then, when covered, paints menuCover
// over every row below the trigger.
func coveredField(w layout.Widget, bg color.NRGBA, triggerH int, covered bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
		w(gtx)
		if covered {
			off := op.Offset(image.Pt(0, triggerH)).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, menuCover, clip.Rect{
				Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y-triggerH),
			}.Op())
			off.Pop()
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// TestOpenMenuIsWholeOverALaterSibling is the paint-order contract: the
// dropped menu stands above the sibling laid out after the field, so every
// pixel of it is the one it draws with nothing over it at all.
func TestOpenMenuIsWholeOverALaterSibling(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(200, row*(1+len(options))+40)
	w := field(t, picker.FieldState{Open: true, Selected: 1, Options: options})
	bg := color.NRGBA{R: 240, G: 240, B: 240, A: 255}

	bare := golden.Capture(t, size, coveredField(w, bg, row, false))
	covered := golden.Capture(t, size, coveredField(w, bg, row, true))

	if got := px(covered, size.X-1, size.Y-1); got != menuCover {
		t.Fatalf("the covering sibling did not paint: (%d,%d) is %v, want %v", size.X-1, size.Y-1, got, menuCover)
	}
	for y := row; y < row*(1+len(options)); y++ {
		for x := 0; x < size.X; x++ {
			if a, b := px(bare, x, y), px(covered, x, y); a != b {
				t.Fatalf("(%d,%d) inside the menu is %v with a later sibling painted and %v without it", x, y, b, a)
			}
		}
	}
}

// px reads one pixel as an opaque NRGBA, so a capture compares against the
// colours it was painted with.
func px(img *image.RGBA, x, y int) color.NRGBA {
	r, g, b, _ := img.At(x, y).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
}

// catalogue is an option list longer than any room a test gives it, so what
// bounds the menu is the room and never the list running out.
func catalogue(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "Option " + string(rune('A'+i%26))
	}
	return out
}

// TestUpwardMenuStaysInTheRoomAbove is the fitting contract on the side the
// caller asked for: a field pinned near the top of a short container, dropping
// upward, floats a plane no taller than the space above its trigger — and the
// caller's own MaxHeight, three times that space, does not buy it back. The
// available room tightens a preference and never loosens it.
//
// The extent is read off the pixels, which is where a floating plane's height
// is visible at all: the field reports its trigger either way.
func TestUpwardMenuStaysInTheRoomAbove(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	long := catalogue(40)
	above := row * 2
	triggerY := row * 3
	size := image.Pt(200, row*8)

	img := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, menuCover, clip.Rect{Max: gtx.Constraints.Max}.Op())
		off := op.Offset(image.Pt(0, triggerY)).Push(gtx.Ops)
		defer off.Pop()
		return field(t, picker.FieldState{
			Open: true, Drop: picker.DropUp, Options: long, MaxHeight: unit.Dp(row * 6),
			AvailableRoom: func(layout.Context) (int, int) { return above, 0 },
		})(gtx)
	})
	if top := triggerY - above; px(img, 100, top-1) != menuCover {
		t.Errorf("y=%d, one pixel above the room the container reported, is %v and not the %v behind it; the upward menu stands taller than the room above its trigger",
			top-1, px(img, 100, top-1), menuCover)
	}
	if top := triggerY - above; px(img, 100, top) == menuCover {
		t.Errorf("y=%d, the top of the room above the trigger, was left unpainted; the menu is shorter than the room it was given", top)
	}
	if mid := triggerY - 1; px(img, 100, mid) == menuCover {
		t.Errorf("y=%d, directly above the trigger, was left unpainted; no menu was floated at all", mid)
	}
}

// TestFieldWithNoRoomFlipsToTheSideThatHasIt is the fitting contract against
// the side the caller asked for: [Drop] is a preference, and a field told
// there is nothing above it and six rows below it drops DOWNWARD however it
// was asked. Nothing is painted over the container above the trigger, the
// plane below is the room below exactly, and the trigger is the downward
// trigger pixel for pixel — the mark travels with the menu, because a mark
// announcing a direction the menu does not take is a defect.
func TestFieldWithNoRoomFlipsToTheSideThatHasIt(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	long := catalogue(40)
	below := row * 6
	triggerY := row
	size := image.Pt(200, row*9)

	flipped := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, menuCover, clip.Rect{Max: gtx.Constraints.Max}.Op())
		off := op.Offset(image.Pt(0, triggerY)).Push(gtx.Ops)
		defer off.Pop()
		return field(t, picker.FieldState{
			Open: true, Drop: picker.DropUp, Options: long,
			AvailableRoom: func(layout.Context) (int, int) { return 0, below },
		})(gtx)
	})
	for y := 0; y < triggerY; y++ {
		if got := px(flipped, 100, y); got != menuCover {
			t.Fatalf("y=%d, above a trigger with no room above it, is %v and not the %v behind it; the menu did not flip", y, got, menuCover)
		}
	}
	if last := triggerY + row + below - 1; px(flipped, 100, last) == menuCover {
		t.Errorf("y=%d, the last pixel of the room below the trigger, was left unpainted; the flipped menu is shorter than the room it flipped into", last)
	}
	if past := triggerY + row + below; px(flipped, 100, past) != menuCover {
		t.Errorf("y=%d, one pixel past the room below the trigger, is %v and not the %v behind it; the flipped menu is taller than the room it flipped into",
			past, px(flipped, 100, past), menuCover)
	}

	downward := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, menuCover, clip.Rect{Max: gtx.Constraints.Max}.Op())
		off := op.Offset(image.Pt(0, triggerY)).Push(gtx.Ops)
		defer off.Pop()
		return field(t, picker.FieldState{Options: long})(gtx)
	})
	for y := triggerY; y < triggerY+row; y++ {
		for x := 0; x < size.X; x++ {
			if a, b := px(flipped, x, y), px(downward, x, y); a != b {
				t.Fatalf("(%d,%d) of the flipped field's trigger is %v and the downward trigger's is %v; the mark does not point the way the menu went", x, y, a, b)
			}
		}
	}
}
