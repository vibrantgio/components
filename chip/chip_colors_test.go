package chip_test

import (
	"testing"

	vgcolor "github.com/vibrantgio/theme/color"
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
//
// Every name that carries a coverage comes back resolved against the fill it
// lands on — the rim and the ring against the surface the chip stands on, the
// words and the marks against the body — because the platform composites in
// encoded sRGB and Gio's rasterizer would not.
func TestResolveNamesThePlatformColors(t *testing.T) {
	for _, sc := range chipSchemes {
		p := sc.p
		t.Run(sc.name, func(t *testing.T) {
			on := p.WindowBackground
			rest := chip.Resolve(p, chip.Filter, chip.RenderState{})
			want := chip.Colors{
				Fill:     p.PushButtonFill,
				Outline:  vgcolor.Flatten(p.Separator, on),
				Outlined: true,
				Ring:     vgcolor.Flatten(p.KeyboardFocusIndicator, on),
				Label:    vgcolor.Flatten(p.ControlText, p.PushButtonFill),
				Mark:     vgcolor.Flatten(p.ControlText, p.PushButtonFill),
				Dismiss:  vgcolor.Flatten(p.SecondaryLabel, p.PushButtonFill),
			}
			if rest != want {
				t.Errorf("a resting chip resolved %+v, want %+v", rest, want)
			}
			picked := chip.Resolve(p, chip.Filter, chip.RenderState{Selected: true})
			wantPicked := chip.Colors{
				Fill: p.SelectedContentBackground,
				// The rim's colour stands and is not drawn: a selected chip
				// has the fill instead.
				Outline:  vgcolor.Flatten(p.Separator, on),
				Outlined: false,
				Ring:     vgcolor.Flatten(p.KeyboardFocusIndicator, on),
				Label:    vgcolor.Flatten(p.AlternateSelectedControlText, p.SelectedContentBackground),
				Mark:     vgcolor.Flatten(p.AlternateSelectedControlText, p.SelectedContentBackground),
				Dismiss:  vgcolor.Flatten(p.SecondaryLabel, p.SelectedContentBackground),
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
				tc.s.Pressed = false
				resting := chip.Resolve(p, chip.Filter, tc.s).Fill
				if want := vgcolor.Flatten(p.PressOverlay, resting); held.Fill != want {
					t.Errorf("a held %s chip's body is %v, want the press overlay over its resting body, %v",
						tc.name, held.Fill, want)
				}
				if held.Fill.A != 0xff {
					t.Errorf("a held %s chip's body alpha is %d, want an opaque fill", tc.name, held.Fill.A)
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
