package chip_test

import (
	"context"
	"image"
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

// TestChipReportsThePillAndHitsTheFloor is the pointer-target contract: the
// widget measures the pill it drew — a chip is sized to its content, and a row
// of them laid out at 44 dp apiece would be a row of gaps — while what the
// pointer may land on is the 44 dp floor centred on it.
//
// The click below is outside the drawn pill on the y axis and inside the slop,
// which is the only place the two can be told apart.
func TestChipReportsThePillAndHitsTheFloor(t *testing.T) {
	var clicked int
	w := live(t, chip.Props{
		Label:   "gpt-5",
		OnClick: func(_ layout.Context) { clicked++ },
	})

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(300, 120))

	dims := drive() // register the input area
	pill := int(tokens.Comfortable.ControlHeight)
	if dims.Size.Y != pill {
		t.Fatalf("chip measured %d px tall, want the density's control height %d", dims.Size.Y, pill)
	}
	if dims.Size.X >= 300 {
		t.Fatalf("chip measured %d px wide at a 300 px constraint: a chip is sized to its content", dims.Size.X)
	}

	// The hit rect is 44 px centred on the 36 px pill: −4..40 on the y axis.
	pos := f32.Pt(float32(dims.Size.X)/2, 38)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	drive()
	if clicked != 1 {
		t.Errorf("click in the slop below the pill: OnClick fired %d times, want 1", clicked)
	}
}

// TestChipActivatesFromTheKeyboard drives the chip through a caller-owned
// clickable — the tag a container anchoring a popover on the chip focuses —
// and activates it with Space and with Enter. Both are widget.Clickable's, and
// both arrive at the one dispatch branch OnClick and Props.Message share; the
// message half cannot be observed from outside mvu (its collector is
// unexported), so OnClick stands for the pair.
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

// TestFocusedChipMeasuresTheSameBox is the live half of what the package doc
// states about the ring: a focused chip's edge IS the focus ring, taking the
// rim's place rather than being drawn beside it, so nothing about the pill
// moves when focus arrives. A ring drawn outside the rim, or inside it, would
// show up here as a chip that grew or a label that shifted the moment the Tab
// key reached it — and a row of chips that reflows on focus is worse than one
// with no ring at all.
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
		t.Errorf("focused chip measured %v, at rest %v: the ring must replace the rim, not widen the pill",
			focused.Size, atRest.Size)
	}
}
