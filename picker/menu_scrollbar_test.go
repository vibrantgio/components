// An internal test: it builds two reference renderings stackRows' public
// surface no longer offers a switch for — the capped viewport with its bar
// held back, and the uncapped stack the scrollbar coupling never touched —
// so it needs drawOptionRow directly, which is unexported on purpose (see
// menu_contrast_test.go).
package picker

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// scrollTestOptions is a catalogue long enough to overflow any cap the tests
// below reach for — the same shape TestCappedMenuIsTheCapAndScrolls uses.
func scrollTestOptions() []string {
	opts := make([]string, 40)
	for i := range opts {
		opts[i] = "Option " + string(rune('A'+i%26))
	}
	return opts
}

// scrollTestTokens is the resolvedTokens both references below draw with —
// the platform's light colours at comfortable density, the same pair
// RenderMenu hands drawMenu, carrying the theme's own bar with nothing
// spent on it: the fading overlay bar, which is what the WhenScrolling and
// Automatic settings come to.
func scrollTestTokens() resolvedTokens {
	return resolvedTokens{
		platform: tokens.PlatformLight,
		body:     tokens.DefaultTypography.BodyLarge,
		spacing:  tokens.Spacing,
		density:  tokens.Comfortable,
		bar:      menuBar(tokens.PlatformLight, nil),
	}
}

// scrollTestRowHeight is what one option row measures at comfortable density
// under scrollTestTokens: one BodyLarge line box plus the density's vertical
// padding, floored at the density's control height. picker_test.go carries
// the same derivation (rowHeight) for its own, external, test package; this
// file is internal (package picker, for drawOptionRow) and cannot use it.
func scrollTestRowHeight() int {
	d := tokens.Comfortable
	h := int(tokens.DefaultTypography.BodyLarge.LineHeight + 2*d.PaddingY)
	if floor := int(d.ControlHeight); h < floor {
		h = floor
	}
	return h
}

// suppressedCappedMenu reproduces the capped path stackRows drew before this
// task: the same rows in the same components/list viewport, laid out through
// [list.LayoutSelectable] rather than [list.LayoutSelectableScrollbar] — no
// scrollbar.Style in sight. Production code has no switch to ask for this
// rendering any more, so the test builds it directly to serve as "the same
// menu with the bar suppressed."
func suppressedCappedMenu(shaper *text.Shaper, tok resolvedTokens, s MenuState) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		n := len(s.Options)
		fieldW := gtx.Constraints.Max.X
		capPx := gtx.Dp(s.MaxHeight)
		if capPx < 1 {
			capPx = 1
		}
		viewGtx := gtx
		viewGtx.Constraints = layout.Constraints{
			Min: image.Pt(fieldW, 0),
			Max: image.Pt(fieldW, capPx),
		}
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		return list.LayoutSelectable(viewGtx, list.NewState(), idx, func(gtx layout.Context, i int, _ bool) layout.Dimensions {
			return drawOptionRow(gtx, shaper, tok, i == s.Selected, i == highlightedRow(s), s.Options[i])
		})
	}
}

// suppressedCappedMenuAt is suppressedCappedMenu with the rows standing at
// first, so it can be compared against a menu whose own rows have been
// scrolled there.
func suppressedCappedMenuAt(shaper *text.Shaper, tok resolvedTokens, s MenuState, first int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		n := len(s.Options)
		fieldW := gtx.Constraints.Max.X
		capPx := gtx.Dp(s.MaxHeight)
		if capPx < 1 {
			capPx = 1
		}
		viewGtx := gtx
		viewGtx.Constraints = layout.Constraints{
			Min: image.Pt(fieldW, 0),
			Max: image.Pt(fieldW, capPx),
		}
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		return list.LayoutSelectable(viewGtx, list.NewStateAt(first), idx, func(gtx layout.Context, i int, _ bool) layout.Dimensions {
			return drawOptionRow(gtx, shaper, tok, i == s.Selected, i == highlightedRow(s), s.Options[i])
		})
	}
}

// stackedRows reproduces stackRows' uncapped branch directly: n rows stacked
// one after the other with no components/list and no bar anywhere near them.
// That branch is untouched by this task, so a menu built through the public
// RenderMenu entry point with no MaxHeight must draw pixel-for-pixel what
// this draws — which is the sense in which the uncapped path is "byte-
// identical to before."
func stackedRows(shaper *text.Shaper, tok resolvedTokens, s MenuState) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		fieldW := gtx.Constraints.Max.X
		totalH := 0
		for i := range s.Options {
			off := op.Offset(image.Pt(0, totalH)).Push(gtx.Ops)
			rowGtx := gtx
			rowGtx.Constraints = layout.Constraints{
				Min: image.Pt(0, 0),
				Max: image.Pt(fieldW, gtx.Constraints.Max.Y),
			}
			dims := drawOptionRow(rowGtx, shaper, tok, i == s.Selected, i == highlightedRow(s), s.Options[i])
			off.Pop()
			totalH += dims.Size.Y
		}
		return layout.Dimensions{Size: image.Pt(fieldW, totalH)}
	}
}

// cappedMenuRows reproduces the capped path the live menu takes, against a
// rows state the caller keeps: the bar is an overlay bar and shows while the
// rows move, so a test that wants to see it has to carry the state from one
// frame to the next the way a running menu does.
func cappedMenuRows(shaper *text.Shaper, tok resolvedTokens, s MenuState, rows *list.State) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		capPx := gtx.Dp(s.MaxHeight)
		if capPx < 1 {
			capPx = 1
		}
		return stackRows(gtx, rows, len(s.Options), capPx, tok, func(gtx layout.Context, i int) layout.Dimensions {
			return drawOptionRow(gtx, shaper, tok, i == s.Selected, i == highlightedRow(s), s.Options[i])
		})
	}
}

// TestCappedOverflowingMenuDrawsTheBarOnlyInTheGutter is BI2.4's core claim in
// pixels: a capped menu whose catalogue overflows the cap draws something the
// same menu with the bar suppressed does not, and everything that differs
// sits inside the bar's own gutter — the trailing scrollbar.Style.Width() of
// the plane. Nothing about a row's own pixels moves to make room for it: the
// bar is Overlay, not Occupy.
//
// The bar is caught on a SCROLLING frame, because that is the only frame an
// overlay bar is drawn on: the first frame gives it a viewport to compare
// against and the second moves the rows under it.
func TestCappedOverflowingMenuDrawsTheBarOnlyInTheGutter(t *testing.T) {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	tok := scrollTestTokens()
	opts := scrollTestOptions()
	row := scrollTestRowHeight()
	cap := unit.Dp(row * 5)
	state := MenuState{Options: opts, Selected: 2, MaxHeight: cap}
	size := image.Pt(200, row*5)

	rows := list.NewState()
	golden.Capture(t, size, cappedMenuRows(shaper, tok, state, rows))
	rows.ScrollTo(8)
	withBar := golden.Capture(t, size, cappedMenuRows(shaper, tok, state, rows))
	suppressed := golden.Capture(t, size, suppressedCappedMenuAt(shaper, tok, state, 8))

	if n := golden.PixelDiff(withBar, suppressed); n == 0 {
		t.Fatal("a capped, overflowing menu drew identically to the same menu with the bar suppressed; the bar never rendered")
	}

	barPx := unit.Metric{PxPerDp: 1, PxPerSp: 1}.Dp(scrollbar.FromTokens(tok.platform, tok.platform.WindowBackground).Width())
	gutter := size.X - barPx
	bounds := withBar.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < gutter; x++ {
			if withBar.RGBAAt(x, y) != suppressed.RGBAAt(x, y) {
				t.Fatalf("pixel (%d,%d) differs outside the %d px trailing gutter; the bar reached into a row's own pixels", x, y, barPx)
			}
		}
	}
}

// TestCappedMenuThatFitsStaysBareWithTheBarWired pins the other half of the
// same rule: handing the capped path a scrollbar.Style costs it nothing when
// the cap never bites, because scrollbar.Style.Layout itself draws nothing
// once the viewport already shows the whole of the content.
func TestCappedMenuThatFitsStaysBareWithTheBarWired(t *testing.T) {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	tok := scrollTestTokens()
	row := scrollTestRowHeight()
	cap := unit.Dp(row * 5)
	state := MenuState{Options: []string{"Alpha", "Beta", "Gamma"}, Selected: 1, MaxHeight: cap}
	size := image.Pt(200, row*5)

	withBar := golden.Capture(t, size, RenderMenu(shaper, tok.platform, tok.spacing,
		tok.body, tok.density, state))
	suppressed := golden.Capture(t, size, suppressedCappedMenu(shaper, tok, state))

	if n := golden.PixelDiff(withBar, suppressed); n != 0 {
		t.Errorf("a capped menu whose rows all fit differs from the bar-suppressed rendering in %d pixel(s); want none", n)
	}
}

// TestUncappedMenuStaysBareWithTheScrollbarCoupling is the uncapped half of
// BI2.4: stackRows' maxH<=0 branch never touches components/list or
// components/scrollbar, so an uncapped menu drawn through the public
// RenderMenu entry point must still match plain stacked rows pixel for pixel
// — the same drawing the uncapped path made before this task, since nothing
// in it changed.
func TestUncappedMenuStaysBareWithTheScrollbarCoupling(t *testing.T) {
	shaper := tokens.DefaultTypography.DeterministicShaper()
	tok := scrollTestTokens()
	opts := scrollTestOptions()
	state := MenuState{Options: opts, Selected: 3, Hovered: 6}
	row := scrollTestRowHeight()
	size := image.Pt(200, row*len(opts))

	uncapped := golden.Capture(t, size, RenderMenu(shaper, tok.platform, tok.spacing,
		tok.body, tok.density, state))
	reference := golden.Capture(t, size, stackedRows(shaper, tok, state))

	if n := golden.PixelDiff(uncapped, reference); n != 0 {
		t.Errorf("an uncapped menu differs from plain stacked rows in %d pixel(s); the scrollbar coupling reached a path it must not touch", n)
	}
}
