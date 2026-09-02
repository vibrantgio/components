package chip_test

import (
	"context"
	"image"
	"math"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/components/chip"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// live subscribes to a chip observable and returns the widget it emitted. A
// static theme emits once synchronously, so the subscription is finished by
// the time Wait returns.
func live(t *testing.T, props chip.Props) layout.Widget {
	t.Helper()
	if props.Shaper == nil {
		// The live path would otherwise take the theme's own shaper, which
		// resolves against whatever fonts the machine has. Every measurement
		// below is of a pill sized to its label, so the faces are pinned.
		props.Shaper = defaultShaper(t)
	}
	var w layout.Widget
	if err := chip.Chip(rx.Of(theme.Default()), props).Subscribe(context.Background(),
		func(next layout.Widget, _ error, done bool) {
			if !done && next != nil {
				w = next
			}
		}).Wait(); err != nil {
		t.Fatalf("Chip subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("Chip emitted no widget")
	}
	return w
}

// driver lays a widget out against a router, one frame per call, and hands
// back the dimensions it reported. Clicked has to run inside the frame that
// processes the events, which is why every assertion below drives a frame
// rather than querying the widget.
func driver(w layout.Widget, r *gioinput.Router, size image.Point) func() layout.Dimensions {
	ops := new(op.Ops)
	return func() layout.Dimensions {
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
}

// TestChipReportsItsBoxAndHitsTheFloor is the pointer-target contract: the
// widget measures the chip it drew, while what the pointer may land on is the
// 44 dp floor centred on it. The click below is outside the drawn chip on the
// y axis and inside the slop, the only place the two can be told apart.
func TestChipReportsItsBoxAndHitsTheFloor(t *testing.T) {
	var clicked int
	w := live(t, chip.Props{
		Label:   "gpt-5",
		OnClick: func(_ layout.Context) { clicked++ },
	})

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(300, 120))

	dims := drive() // register the input area
	box := int(tokens.Comfortable.ChipHeight())
	if dims.Size.Y != box {
		t.Fatalf("chip measured %d px tall, want the density's chip height %d", dims.Size.Y, box)
	}
	if dims.Size.X >= 300 {
		t.Fatalf("chip measured %d px wide at a 300 px constraint: a chip is sized to its content", dims.Size.X)
	}

	// The hit rect is 44 px centred on the 32 px chip: −6..38 on the y axis.
	pos := f32.Pt(float32(dims.Size.X)/2, 36)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if clicked != 1 {
		t.Errorf("click in the slop below the chip: OnClick fired %d times, want 1", clicked)
	}
}

// TestChipActivatesFromTheKeyboard drives the chip through a caller-owned
// clickable and activates it with Space and with Enter. Both arrive at the one
// dispatch branch OnClick and Props.Message share; the message half cannot be
// observed from outside mvu (its collector is unexported), so OnClick stands
// for the pair.
func TestChipActivatesFromTheKeyboard(t *testing.T) {
	var clicked int
	var click widget.Clickable
	w := live(t, chip.Props{
		Label:     "main",
		Icon:      chevron,
		Clickable: &click,
		Message:   "chip-activated",
		OnClick:   func(_ layout.Context) { clicked++ },
	})

	r := new(gioinput.Router)
	size := image.Pt(200, 80)
	drive := driver(w, r, size)

	// Frame 1 registers the clickable's focus filter; the container then
	// drives focus to the tag it owns.
	focusOnce := true
	composed := func(gtx layout.Context) layout.Dimensions {
		dims := w(gtx)
		if focusOnce {
			gtx.Execute(key.FocusCmd{Tag: &click})
			focusOnce = false
		}
		return dims
	}
	driver(composed, r, size)()
	drive() // frame 2: focus is applied

	probe := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         new(op.Ops),
		Source:      r.Source(),
	}
	if !probe.Focused(&click) {
		t.Fatal("the caller-owned clickable is not focused: container-driven focus failed")
	}

	r.Queue(
		key.Event{Name: key.NameSpace, State: key.Press},
		key.Event{Name: key.NameSpace, State: key.Release},
	)
	drive()
	if clicked != 1 {
		t.Fatalf("Space activation: OnClick fired %d times, want 1", clicked)
	}

	r.Queue(
		key.Event{Name: key.NameReturn, State: key.Press},
		key.Event{Name: key.NameReturn, State: key.Release},
	)
	drive()
	if clicked != 2 {
		t.Errorf("Enter activation: OnClick fired %d times in total, want 2", clicked)
	}
}

// TestFocusedChipMeasuresTheSameBox: a focused chip's edge is the focus ring,
// taking the outline's place rather than being drawn beside it, so nothing
// about the box moves when focus arrives. A ring drawn outside or inside the
// outline would show here as a chip that grew or a label that shifted.
func TestFocusedChipMeasuresTheSameBox(t *testing.T) {
	var click widget.Clickable
	w := live(t, chip.Props{
		Label:     "main",
		Icon:      chevron,
		Clickable: &click,
	})

	r := new(gioinput.Router)
	size := image.Pt(200, 80)
	drive := driver(w, r, size)

	atRest := drive()

	focusOnce := true
	composed := func(gtx layout.Context) layout.Dimensions {
		dims := w(gtx)
		if focusOnce {
			gtx.Execute(key.FocusCmd{Tag: &click})
			focusOnce = false
		}
		return dims
	}
	driver(composed, r, size)()
	focused := drive()

	probe := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(size),
		Ops:         new(op.Ops),
		Source:      r.Source(),
	}
	if !probe.Focused(&click) {
		t.Fatal("the chip did not take focus: the live path registered no focusable tag")
	}
	if focused.Size != atRest.Size {
		t.Errorf("focused chip measured %v, at rest %v: the ring must replace the outline, not widen the chip",
			focused.Size, atRest.Size)
	}
}

// TestPinnedChipDrawsAtTheEdgeOfTheBox is the pinning seam: the chip is the
// same chip at the same width, drawn at the edge of the box it was offered,
// and the widget reports that box.
//
// Where the chip landed is measured with the pointer rather than with pixels,
// because the target is centred on the drawn chip: a press two pixels in from
// the box's trailing edge reaches a trailing-pinned chip and a press two
// pixels in from its leading edge falls in the slack and reaches nothing. A
// chip that reported the box but drew at the origin would pass a dimension
// check and fail both of these.
func TestPinnedChipDrawsAtTheEdgeOfTheBox(t *testing.T) {
	const label = "OpenAI · gpt-5.5"
	box := image.Pt(300, 120)

	// The chip's own width, which the pin must not change.
	pill := driver(live(t, chip.Props{Label: label, Icon: chevron}), new(gioinput.Router), box)()
	if pill.Size.X >= box.X {
		t.Fatalf("the unpinned chip measured %d px in a %d px box; there is no slack to pin across",
			pill.Size.X, box.X)
	}

	for _, tc := range []struct {
		name   string
		pin    chip.Pin
		hits   int // an x that must reach the pill
		misses int // an x that must fall in the slack
	}{
		{"trailing", chip.PinTrailing, box.X - 2, 2},
		{"leading", chip.PinLeading, 2, box.X - 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var clicked int
			w := live(t, chip.Props{
				Label:   label,
				Icon:    chevron,
				Pin:     tc.pin,
				OnClick: func(_ layout.Context) { clicked++ },
			})
			r := new(gioinput.Router)
			drive := driver(w, r, box)

			dims := drive()
			if dims.Size.X != box.X {
				t.Fatalf("a pinned chip measured %d px wide, want the %d px box it was offered",
					dims.Size.X, box.X)
			}
			if dims.Size.Y != pill.Size.Y {
				t.Errorf("a pinned chip measured %d px tall and the pill is %d: pinning is horizontal only",
					dims.Size.Y, pill.Size.Y)
			}

			press(r, tc.misses, pill.Size.Y/2)
			drive()
			if clicked != 0 {
				t.Errorf("a press %d px in from the box's other edge activated the chip: the pill is not pinned %s",
					tc.misses, tc.name)
			}

			press(r, tc.hits, pill.Size.Y/2)
			drive()
			if clicked != 1 {
				t.Errorf("a press %d px in from the %s edge reached the chip %d times, want once",
					tc.hits, tc.name, clicked)
			}
		})
	}
}

// press queues a click at one point.
func press(r *gioinput.Router, x, y int) {
	pos := f32.Pt(float32(x), float32(y))
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
}

// TestFilterChipTogglesAndReports is the one purpose that holds state: the live
// chip keeps its own selection, seeded by Props, and reports every move through
// OnSelect while OnClick still fires for the activation itself.
func TestFilterChipTogglesAndReports(t *testing.T) {
	var moves []bool
	var clicked int
	w := live(t, chip.Props{
		Label:    "Unread",
		Purpose:  chip.Filter,
		OnSelect: func(_ layout.Context, selected bool) { moves = append(moves, selected) },
		OnClick:  func(_ layout.Context) { clicked++ },
	})

	r := new(gioinput.Router)
	size := image.Pt(300, 120)
	drive := driver(w, r, size)

	dims := drive()
	mid := dims.Size.Y / 2
	press(r, dims.Size.X/2, mid)
	drive()
	press(r, dims.Size.X/2, mid)
	drive()

	if want := []bool{true, false}; len(moves) != len(want) || moves[0] != want[0] || moves[1] != want[1] {
		t.Errorf("OnSelect saw %v, want %v: a filter chip toggles and reports where it landed", moves, want)
	}
	if clicked != 2 {
		t.Errorf("OnClick fired %d times, want 2: a toggle is still an activation", clicked)
	}
}

// TestOnlyAFilterReportsSelection is the anatomy rule on the live path: the
// other three purposes have no selection to move, so nothing they are clicked
// with may reach OnSelect.
func TestOnlyAFilterReportsSelection(t *testing.T) {
	for _, in := range []struct {
		name string
		i    chip.Purpose
	}{
		{"assist", chip.Assist},
		{"input", chip.Input},
		{"suggestion", chip.Suggestion},
	} {
		t.Run(in.name, func(t *testing.T) {
			moved := 0
			w := live(t, chip.Props{
				Label:    "Token",
				Purpose:  in.i,
				Selected: true, // ignored: this purpose carries no selection
				OnSelect: func(_ layout.Context, _ bool) { moved++ },
			})
			r := new(gioinput.Router)
			drive := driver(w, r, image.Pt(300, 120))
			dims := drive()
			press(r, int(tokens.Comfortable.PaddingX), dims.Size.Y/2)
			drive()
			if moved != 0 {
				t.Errorf("OnSelect fired %d times on a %s chip: only a filter chip is selectable", moved, in.name)
			}
		})
	}
}

// TestInputChipDismissesFromItsOwnTarget drives the dismiss mark, which is the
// second pointer target the live path registers: it lies over the body's and
// takes the pointer where the two overlap, so a click on the mark dismisses
// rather than activating.
func TestInputChipDismissesFromItsOwnTarget(t *testing.T) {
	var dismissed, clicked int
	w := live(t, chip.Props{
		Label:     "recipient@example.com",
		Purpose:   chip.Input,
		OnDismiss: func(_ layout.Context) { dismissed++ },
		OnClick:   func(_ layout.Context) { clicked++ },
	})

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(400, 120))
	dims := drive()

	// The mark's centre: half an icon in from where the trailing padding
	// starts, which is where the anatomy puts it.
	markX := dims.Size.X - int(tokens.Comfortable.PaddingX) -
		int(math.Round(float64(chip.MarkDp(tokens.DefaultTypography.LabelLarge))))/2
	press(r, markX, dims.Size.Y/2)
	drive()
	if dismissed != 1 {
		t.Errorf("a click on the dismiss mark dismissed %d times, want 1", dismissed)
	}
	if clicked != 0 {
		t.Errorf("a click on the dismiss mark also activated the chip %d times: the mark's target is on top", clicked)
	}

	// The label's own surface still activates the chip.
	press(r, int(tokens.Comfortable.PaddingX), dims.Size.Y/2)
	drive()
	if clicked != 1 {
		t.Errorf("a click on the label activated the chip %d times, want 1", clicked)
	}
	if dismissed != 1 {
		t.Errorf("a click on the label dismissed the chip %d times, want none beyond the first", dismissed-1)
	}
}
