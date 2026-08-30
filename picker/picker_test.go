package picker_test

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/chip"
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

// TestOpenFieldStacksTheSharedMenuUnderItsTrigger is the one-surface contract:
// the menu an open field drops is [Menu], not a second drawing of it. The
// composite below draws the closed trigger and then the standalone menu at the
// trigger's own height, and the open field has to be that image exactly.
func TestOpenFieldStacksTheSharedMenuUnderItsTrigger(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(200, row*(1+len(options)))

	open := golden.Capture(t, size, field(t, picker.FieldState{
		Open: true, Options: options, Selected: 1,
	}))
	composed := golden.Capture(t, size, func(gtx layout.Context) layout.Dimensions {
		trigger := field(t, picker.FieldState{Options: options, Selected: 1})(gtx)
		off := op.Offset(image.Pt(0, trigger.Size.Y)).Push(gtx.Ops)
		rows := menu(t, picker.MenuState{Options: options, Selected: 1})(gtx)
		off.Pop()
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, trigger.Size.Y+rows.Size.Y)}
	})
	if n := golden.PixelDiff(open, composed); n != 0 {
		t.Errorf("an open field differs from its own trigger plus a standalone menu in %d pixels; both triggers stand under the one surface", n)
	}
}

// TestOpenFieldMeasuresTheTriggerPlusEveryRow: the open widget reports the
// whole stack, because what it drew is what a container has to make room for.
func TestOpenFieldMeasuresTheTriggerPlusEveryRow(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	dims := measure(t, image.Pt(200, 400), field(t, picker.FieldState{Open: true, Options: options}))
	if want := row * (1 + len(options)); dims.Size.Y != want {
		t.Errorf("open field measured %d px tall, want %d px (trigger + %d rows of %d)", dims.Size.Y, want, len(options), row)
	}
	if dims.Size.X != 200 {
		t.Errorf("open field measured %d px wide, want the 200 px it was offered: a menu is as wide as its trigger", dims.Size.X)
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

// TestAnchorIsSizedToItsValue: the chrome register's trigger is a control
// around its value, not a bar across its container — it clamps to what it is
// offered and otherwise reports its own width, at the density's control
// height.
func TestAnchorIsSizedToItsValue(t *testing.T) {
	w := picker.RenderAnchor(defaultShaper(t), "Anthropic · Opus 5", tokens.DefaultLight,
		tokens.Spacing, tokens.Radius, tokens.DefaultTypography.LabelLarge,
		tokens.Comfortable, picker.AnchorState{})
	dims := measure(t, image.Pt(400, 200), w)
	if dims.Size.X >= 400 {
		t.Errorf("anchor measured %d px wide at a 400 px constraint: a picker's trigger is sized to its value", dims.Size.X)
	}
	if want := int(tokens.Comfortable.ControlHeight); dims.Size.Y != want {
		t.Errorf("anchor height = %d px, want the density's control height %d px", dims.Size.Y, want)
	}
}

// TestAnchorAndTheChipForwarderAgree holds the forwarder honest: chip's
// deprecated RenderAnchor and this package's draw the same control from the
// same geometry, so a caller that has not moved yet loses no pixel.
func TestAnchorAndTheChipForwarderAgree(t *testing.T) {
	const value = "Anthropic · Opus 5"
	size := image.Pt(240, 60)
	for _, s := range []struct {
		name string
		st   picker.AnchorState
	}{
		{"rest", picker.AnchorState{}},
		{"hovered", picker.AnchorState{Hovered: true}},
		{"pressed", picker.AnchorState{Pressed: true}},
		{"focused", picker.AnchorState{Focused: true}},
		{"raised", picker.AnchorState{Ground: tokens.Level2}},
	} {
		t.Run(s.name, func(t *testing.T) {
			shaper := defaultShaper(t)
			mine := golden.Capture(t, size, picker.RenderAnchor(shaper, value, tokens.DefaultDark,
				tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable, s.st))
			theirs := golden.Capture(t, size, chip.RenderAnchor(shaper, value, tokens.DefaultDark,
				tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelLarge, tokens.Comfortable,
				chip.RenderState(s.st)))
			if n := golden.PixelDiff(mine, theirs); n != 0 {
				t.Errorf("chip.RenderAnchor and picker.RenderAnchor differ in %d pixels", n)
			}
		})
	}
}
