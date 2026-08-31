package badge_test

import (
	"context"
	"image"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/components/badge"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// live subscribes to a badge observable and returns the widget it emitted. A
// static theme emits once synchronously, so the subscription is finished by
// the time Wait returns.
func live(t *testing.T, props badge.Props) layout.Widget {
	t.Helper()
	if props.Shaper == nil {
		// The live path would otherwise take the theme's own shaper, which
		// resolves against whatever fonts the machine has. Every measurement
		// below is of a badge sized to its label, so the faces are pinned.
		props.Shaper = defaultShaper(t)
	}
	var w layout.Widget
	if err := badge.Badge(rx.Of(theme.Default()), props).Subscribe(context.Background(),
		func(next layout.Widget, _ error, done bool) {
			if !done && next != nil {
				w = next
			}
		}).Wait(); err != nil {
		t.Fatalf("Badge subscribe: %v", err)
	}
	if w == nil {
		t.Fatal("Badge emitted no widget")
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

// press queues a click at one point.
func press(r *gioinput.Router, x, y int) {
	pos := f32.Pt(float32(x), float32(y))
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
}

// TestTheBadgeReportsItsTextAndTheMarkHitsTheFloor is the pointer-target
// contract: the widget measures the words it drew, while what the pointer may
// land on around the close mark is the 24 dp AA floor centred on it. The click
// below is outside the drawn badge on the y axis and inside the mark's slop,
// the only place the two can be told apart.
func TestTheBadgeReportsItsTextAndTheMarkHitsTheFloor(t *testing.T) {
	var dismissed int
	w := live(t, badge.Props{
		Label:     "Filtered by owner",
		Variant:   badge.Info,
		OnDismiss: func(_ layout.Context) { dismissed++ },
	})

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(300, 120))

	dims := drive() // register the input area
	lineBox := int(badge.Style(tokens.DefaultTypography, tokens.Comfortable).LineHeight)
	if dims.Size.Y != lineBox {
		t.Fatalf("badge measured %d px tall, want the %d px line box", dims.Size.Y, lineBox)
	}
	if dims.Size.X >= 300 {
		t.Fatalf("badge measured %d px wide at a 300 px constraint: a badge is sized to its content", dims.Size.X)
	}

	// The mark's drawn square ends at the badge's trailing edge, and its
	// target overhangs by (24 − 8)/2 = 8 px on every side, so a press two
	// pixels below the badge's own box lands in the slop and nowhere else.
	press(r, dims.Size.X-lineBox/4, dims.Size.Y+2)
	drive()
	if dismissed != 1 {
		t.Errorf("a press in the slop below the close mark dismissed %d times, want 1", dismissed)
	}
}

// TestAClickAwayFromTheMarkIsNotADismissal is the same contract read the other
// way: the badge's own words take no pointer input, so a press on the label
// reaches nothing. A badge whose whole box was clickable would be a control.
func TestAClickAwayFromTheMarkIsNotADismissal(t *testing.T) {
	var dismissed int
	w := live(t, badge.Props{
		Label:     "Filtered by owner",
		Variant:   badge.Info,
		OnDismiss: func(_ layout.Context) { dismissed++ },
	})

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(300, 120))
	dims := drive()

	press(r, 2, dims.Size.Y/2)
	drive()
	if dismissed != 0 {
		t.Errorf("a press on the badge's leading word dismissed it %d times, want 0", dismissed)
	}
}

// TestABadgeWithoutDismissTakesNoInput: a badge with a nil OnDismiss draws no
// mark and registers no pointer area, so nothing anywhere in its box — or in
// the air a target would have overhung — answers a press.
func TestABadgeWithoutDismissTakesNoInput(t *testing.T) {
	plain := live(t, badge.Props{Label: "Popular", Variant: badge.Info})
	var dismissed int
	dismissible := live(t, badge.Props{
		Label:     "Popular",
		Variant:   badge.Info,
		OnDismiss: func(_ layout.Context) { dismissed++ },
	})

	r := new(gioinput.Router)
	size := image.Pt(300, 120)
	plainDims := driver(plain, r, size)()
	markedDims := driver(dismissible, r, size)()

	if plainDims.Size.X >= markedDims.Size.X {
		t.Fatalf("a plain badge measured %d px and a dismissible one %d: the mark costs width",
			plainDims.Size.X, markedDims.Size.X)
	}

	drive := driver(plain, r, size)
	drive()
	press(r, plainDims.Size.X-2, plainDims.Size.Y/2)
	drive()
	if dismissed != 0 {
		t.Errorf("a badge with no OnDismiss reported %d dismissals", dismissed)
	}
}

// TestOneDoubleClickIsOneDismissal drains the clickable to empty and reports
// once. Two clicks in one frame on a mark whose label the caller is about to
// take away must not fire twice, and the second must not be left queued to
// fire on the frame after that.
func TestOneDoubleClickIsOneDismissal(t *testing.T) {
	var dismissed int
	w := live(t, badge.Props{
		Label:     "Beta",
		OnDismiss: func(_ layout.Context) { dismissed++ },
	})

	r := new(gioinput.Router)
	drive := driver(w, r, image.Pt(300, 120))
	dims := drive()

	x, y := dims.Size.X-dims.Size.Y/4, dims.Size.Y/2
	press(r, x, y)
	press(r, x, y)
	drive()
	if dismissed != 1 {
		t.Fatalf("a double click on the close mark dismissed %d times, want 1", dismissed)
	}
	drive()
	if dismissed != 1 {
		t.Errorf("a second dismissal arrived on the next frame: %d in total, want 1", dismissed)
	}
}

// TestTheLiveBadgeDrawsWhatRenderDraws is the seam between the two paths: the
// live widget and the pure one report the same box for the same badge, so a
// caller measuring one has measured the other.
func TestTheLiveBadgeDrawsWhatRenderDraws(t *testing.T) {
	shaper := defaultShaper(t)
	pure := measure(t, badge.Render(shaper, "Popular", nil, badge.Success,
		tokens.DefaultLight, tokens.Spacing, tokens.Radius, badgeStyle(), badge.RenderState{}))

	w := live(t, badge.Props{Label: "Popular", Variant: badge.Success, Shaper: shaper})
	got := driver(w, new(gioinput.Router), image.Pt(1000, 1000))()
	if got.Size != pure {
		t.Errorf("the live badge measured %v and Render measured %v", got.Size, pure)
	}
}

// TestEachBadgeNamesItself is the screen-reader contract, and the reason the
// badge scopes its own area: a semantic label attaches to the innermost area
// around it, so two badges laid out side by side must come back as two named
// nodes rather than one label overwriting the other on whatever encloses them.
func TestEachBadgeNamesItself(t *testing.T) {
	first := live(t, badge.Props{Label: "Popular", Variant: badge.Info})
	second := live(t, badge.Props{Label: "Deprecated", Variant: badge.Warning,
		Description: "This model is deprecated"})

	r := new(gioinput.Router)
	drive := driver(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{}.Layout(gtx, layout.Rigid(first), layout.Rigid(second))
	}, r, image.Pt(400, 60))
	drive()

	labels := map[string]string{}
	var walk func([]gioinput.SemanticNode)
	walk = func(ns []gioinput.SemanticNode) {
		for _, n := range ns {
			if n.Desc.Label != "" {
				labels[n.Desc.Label] = n.Desc.Description
			}
			walk(n.Children)
		}
	}
	walk(r.AppendSemantics(nil))

	if _, ok := labels["Popular"]; !ok {
		t.Errorf("no semantic node named %q: %v", "Popular", labels)
	}
	if got, ok := labels["Deprecated"]; !ok {
		t.Errorf("no semantic node named %q: %v", "Deprecated", labels)
	} else if got != "This model is deprecated" {
		t.Errorf("the second badge's description is %q, want the one Props carried", got)
	}
}
