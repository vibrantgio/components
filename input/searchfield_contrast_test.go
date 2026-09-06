// The search field's two marks, measured rather than eyeballed: the looking
// glass that names the control and the clear mark that empties it, both drawn
// in the control family's prompt foreground, against the fill they are drawn
// on at every level a field can be handed.
//
// A mark is a graphic and not text, so what it answers to is WCAG 1.4.11 —
// the same floor the resting border clears in
// TestControlBorderClearsTheGraphicFloor, and the reason that sweep is not
// enough on its own: the border is measured against the surfaces either side
// of it, and these two are drawn inside the field, on its own interior.
package input

import (
	"testing"

	themecolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/internal/control"
)

// TestSearchMarksClearTheGraphicFloor asserts the looking glass and the clear
// mark stay visible against the field's own fill on every level, in both
// schemes: a field standing on a raised host fills one step above that host,
// so a mark that clears the floor over the window's own surface can still
// disappear inside a field on a level-3 dialog.
//
// Both marks take one colour, so one measurement answers for both — and the
// test names them separately anyway, because the day they stop sharing it is
// the day this has to fail twice.
func TestSearchMarksClearTheGraphicFloor(t *testing.T) {
	for _, sc := range []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := sc.colors
			for _, level := range controlLevels {
				fill := controlFill(c, level.level)
				for _, m := range []string{"the looking glass", "the clear mark"} {
					mark := control.Placeholder(c)
					got := themecolor.ContrastRatio(mark, fill)
					t.Logf("%s %s %s against the %s field's fill %s: %.2f:1",
						level.name, m, hex(mark), level.name, hex(fill), got)
					if got < graphicFloor {
						t.Errorf("%s %s %s against the %s field's fill %s = %.2f:1, want at least %.1f:1",
							level.name, m, hex(mark), level.name, hex(fill), got, graphicFloor)
					}
				}
			}
		})
	}
}
