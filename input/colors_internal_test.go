package input

import (
	"testing"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/theme/tokens"
)

// A switched-off text field fades its edge toward the surface it stands on at
// the platform's measured disabled coverage, and moves nothing else about the
// box: the interior is that same surface already — control.FieldFill carries
// the reading — so there is no fill of the field's own left to fade.
func TestTheSwitchedOffFieldsEdgeFades(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		restFill, _, restEdge, _, _ := textFieldColors(p, RenderState{})
		offFill, _, offEdge, _, _ := textFieldColors(p, RenderState{Disabled: true})
		if offFill != restFill {
			t.Errorf("a switched-off field fills %v against the resting %v; the interior is the surface and does not move", offFill, restFill)
		}
		if offEdge == restEdge {
			t.Errorf("a switched-off field draws the resting edge %v", offEdge)
		}
		if want := control.Faded(restEdge, p.WindowBackground); offEdge != want {
			t.Errorf("a switched-off field's edge is %v, want the hairline at the platform's disabled coverage, %v", offEdge, want)
		}
	}
}
