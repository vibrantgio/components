package picker_test

// The capped menu's own scroll bar, read at the two settings the platform
// answers with. The menu builds the bar from the colours it is drawn in and
// then offers it to the application, which spends the platform's "Show
// scroll bars" setting on it; what that spending comes to is a bar standing
// on a resting frame or no bar at all, which is what these pixels are.

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// TestTheMenuSpendsTheScrollBarSetting lays a capped menu whose catalogue
// overflows the cap out on a frame nothing has scrolled and nothing is
// pointing at, once with the reader's "always" setting spent on its bar and
// once with the bar as the theme gives it.
//
// With "always" the thumb is there: the bar stands for as long as the rows
// overflow the cap, in a lane of its own, and the rows stop short of the
// lane — so left of the lane the standing menu draws what the fading menu
// draws when it is offered only the width the lane leaves. With either of
// the other two settings it is the platform's overlay bar, which a resting
// frame holds nothing of.
func TestTheMenuSpendsTheScrollBarSetting(t *testing.T) {
	row := rowHeight(tokens.Comfortable)
	size := image.Pt(200, row*5)
	resting := func(width int, spend func(scrollbar.Style) scrollbar.Style) *image.RGBA {
		w := materialize(t, picker.Menu(rx.Of(liveTheme()), picker.MenuProps{
			Options:   catalogue(40),
			Selected:  0,
			MaxHeight: unit.Dp(row * 5),
			Scrollbar: spend,
			Shaper:    tokens.DefaultTypography.DeterministicShaper(),
		}))
		return golden.Capture(t, image.Pt(width, size.Y), func(gtx layout.Context) layout.Dimensions {
			gtx.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
			gtx.Now = time.Unix(1700000000, 0)
			return w(gtx)
		})
	}

	barPx := unit.Metric{PxPerDp: 1, PxPerSp: 1}.Dp(scrollbar.FromTokens(tokens.PlatformLight, tokens.PlatformLight.WindowBackground).Width())
	lane := size.X - barPx

	standing := resting(size.X, func(s scrollbar.Style) scrollbar.Style { return s.Always() })
	fading := resting(size.X, nil)
	narrow := resting(lane, nil)

	if n := golden.PixelDiff(standing, fading); n == 0 {
		t.Fatal("with scroll bars always, a resting menu drew no thumb")
	}
	for y := range size.Y {
		for x := range lane {
			if standing.RGBAAt(x, y) != narrow.RGBAAt(x, y) {
				t.Fatalf("pixel (%d,%d) differs from the rows laid out %d px wide; with scroll bars always the rows did not stop short of the %d px lane", x, y, lane, barPx)
			}
		}
	}
}
