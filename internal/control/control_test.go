package control_test

import (
	"testing"

	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/theme/tokens"
)

// One control family, one set of names: the text field, the checkbox, the
// radio and the picker's field trigger all draw their box from these three,
// and each is the platform's own name for that part of a field.
func TestTheBoxTakesThePlatformsNames(t *testing.T) {
	for _, sc := range []struct {
		name string
		p    tokens.PlatformColors
	}{
		{"light", tokens.PlatformLight},
		{"dark", tokens.PlatformDark},
	} {
		p := sc.p
		if got, want := control.Border(p), p.FieldEdge; got != want {
			t.Errorf("%s: Border = %v, want the platform's field edge %v", sc.name, got, want)
		}
		if got, want := control.Fill(p), p.TextBackground; got != want {
			t.Errorf("%s: Fill = %v, want the platform's text background %v", sc.name, got, want)
		}
		if got, want := control.Placeholder(p), p.PlaceholderText; got != want {
			t.Errorf("%s: Placeholder = %v, want the platform's placeholder text %v", sc.name, got, want)
		}
	}
}

// The prompt has to be visibly not a value, and the platform says how far:
// its placeholder carries a coverage over the field's own fill rather than
// being a second opaque colour.
func TestThePlaceholderCarriesACoverage(t *testing.T) {
	for _, p := range []tokens.PlatformColors{tokens.PlatformLight, tokens.PlatformDark} {
		if a := control.Placeholder(p).A; a == 0 || a == 0xff {
			t.Errorf("Placeholder alpha = %d, want the platform's partial coverage", a)
		}
	}
}
