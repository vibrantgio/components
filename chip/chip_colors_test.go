package chip_test

import (
	"testing"

	"github.com/vibrantgio/theme/tokens"

	"github.com/vibrantgio/components/chip"
)

// chipSchemes is the platform's two recorded sets, which is every colour a
// chip can be handed.
var chipSchemes = []struct {
	name string
	p    tokens.PlatformColors
}{
	{"light", tokens.PlatformLight},
	{"dark", tokens.PlatformDark},
}

var chipPurposes = []struct {
	name string
	i    chip.Purpose
}{
	{"assist", chip.Assist},
	{"filter", chip.Filter},
	{"input", chip.Input},
	{"suggestion", chip.Suggestion},
}

// TestResolveNamesThePlatformColors pins the whole table a chip draws from
// against the platform's own field names, which is the only claim there is to
// make now that nothing is measured: a resting chip is the platform's small
// control, a selected filter is its emphasized selection, and a held chip
// wears the press coverage over whichever body it started from.
func TestResolveNamesThePlatformColors(t *testing.T) {
	for _, sc := range chipSchemes {
		p := sc.p
		t.Run(sc.name, func(t *testing.T) {
			rest := chip.Resolve(p, chip.Filter, chip.RenderState{})
			want := chip.Colors{
				Fill:     p.Control,
				Outline:  p.Separator,
				Outlined: true,
				Label:    p.ControlText,
				Mark:     p.ControlText,
				Dismiss:  p.SecondaryLabel,
			}
			if rest != want {
				t.Errorf("a resting chip resolved %+v, want %+v", rest, want)
			}
			picked := chip.Resolve(p, chip.Filter, chip.RenderState{Selected: true})
			wantPicked := chip.Colors{
				Fill: p.SelectedContentBackground,
				// The rim's colour stands and is not drawn: a selected chip
				// has the fill instead.
				Outline:  p.Separator,
				Outlined: false,
				Label:    p.AlternateSelectedControlText,
				Mark:     p.AlternateSelectedControlText,
				Dismiss:  p.SecondaryLabel,
			}
			if picked != wantPicked {
				t.Errorf("a selected filter chip resolved %+v, want %+v", picked, wantPicked)
			}
			for _, tc := range []struct {
				name string
				s    chip.RenderState
			}{
				{"unselected", chip.RenderState{Pressed: true}},
				{"selected", chip.RenderState{Selected: true, Pressed: true}},
			} {
				held := chip.Resolve(p, chip.Filter, tc.s)
				if held.Overlay != p.PressOverlay {
					t.Errorf("a held %s chip lays %v over its body, want PressOverlay %v",
						tc.name, held.Overlay, p.PressOverlay)
				}
				tc.s.Pressed = false
				if got, want := held.Fill, chip.Resolve(p, chip.Filter, tc.s).Fill; got != want {
					t.Errorf("a held %s chip changed its body to %v from %v; the press is an overlay",
						tc.name, got, want)
				}
			}
		})
	}
}

// TestHoverMovesNoColour is the measured platform fact stated as a claim on
// the component: a push-button-shaped control does not change colour under the
// pointer on macOS 26, so hover resolves to the resting colours exactly.
func TestHoverMovesNoColour(t *testing.T) {
	for _, sc := range chipSchemes {
		for _, in := range chipPurposes {
			for _, selected := range []bool{false, true} {
				rest := chip.Resolve(sc.p, in.i, chip.RenderState{Selected: selected})
				hovered := chip.Resolve(sc.p, in.i, chip.RenderState{Selected: selected, Hovered: true})
				if rest != hovered {
					t.Errorf("%s %s (selected %v): hover resolved %+v, want the resting %+v",
						sc.name, in.name, selected, hovered, rest)
				}
			}
		}
	}
}

// TestThePurposesAreOneColour is the structure rule: the four purposes differ
// in behaviour and in what stands in their slots, never in colour, so a chip
// that answered a colour of its own per purpose would be drawing a difference
// the platform does not draw.
func TestThePurposesAreOneColour(t *testing.T) {
	for _, sc := range chipSchemes {
		want := chip.Resolve(sc.p, chip.Assist, chip.RenderState{})
		for _, in := range chipPurposes {
			if got := chip.Resolve(sc.p, in.i, chip.RenderState{}); got != want {
				t.Errorf("%s: a resting %s chip resolved %+v, want the one answer %+v",
					sc.name, in.name, got, want)
			}
		}
	}
}

// TestOnlyFilterCanBeSelected is the structure rule where a lookup could
// silently break it: three of the four purposes must resolve identically
// whether or not the caller set Selected, because they have no selection to
// draw.
func TestOnlyFilterCanBeSelected(t *testing.T) {
	for _, sc := range chipSchemes {
		for _, in := range chipPurposes {
			if in.i.Selectable() {
				continue
			}
			plain := chip.Resolve(sc.p, in.i, chip.RenderState{})
			marked := chip.Resolve(sc.p, in.i, chip.RenderState{Selected: true})
			if plain != marked {
				t.Errorf("%s %s: Selected changed the colours to %+v from %+v; only a filter chip is selectable",
					sc.name, in.name, marked, plain)
			}
		}
	}
}
