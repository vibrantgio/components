package badge_test

import (
	"image/color"
	"testing"

	"github.com/vibrantgio/components/badge"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// The badge's colours, pinned by the platform's own names rather than
// measured. Nothing here is derived: a status is a system colour, the
// foreground on it is the one the platform pairs with a filled control, and
// Neutral is the platform's grey. So what a test can hold is which name
// landed where, and that the five stay five.

// TestEveryStatusIsThePlatformsColourForIt is the table the package doc
// states, read off both recorded appearances. A badge that reached for the
// wrong system colour — systemYellow for Warning, say, where the platform
// paints systemOrange — passes every geometry test in this package.
func TestEveryStatusIsThePlatformsColourForIt(t *testing.T) {
	for _, sc := range goldenSchemes {
		for _, tc := range []struct {
			status badge.Status
			name   string
			want   color.NRGBA
		}{
			{badge.Success, "SystemGreen", sc.p.SystemGreen},
			{badge.Warning, "SystemOrange", sc.p.SystemOrange},
			{badge.Error, "SystemRed", sc.p.SystemRed},
			{badge.Info, "SystemBlue", sc.p.SystemBlue},
			{badge.Neutral, "SystemGray", sc.p.SystemGray},
		} {
			if got := badge.Fill(sc.p, tc.status); got != tc.want {
				t.Errorf("%s: Fill(%v) = %v, want %s %v", sc.name, tc.status, got, tc.name, tc.want)
			}
		}
	}
}

// TestAFilledBadgeReadsInTheForegroundThePlatformPairsWithAFill: the content
// of a worded or counted badge is alternateSelectedControlTextColor and does
// not vary with the status, because the platform's own badge does not.
func TestAFilledBadgeReadsInTheForegroundThePlatformPairsWithAFill(t *testing.T) {
	for _, sc := range goldenSchemes {
		want := sc.p.AlternateSelectedControlText
		if got := badge.Foreground(sc.p); got != want {
			t.Errorf("%s: Foreground = %v, want AlternateSelectedControlText %v", sc.name, got, want)
		}
	}
}

// TestABareSignIsTheSystemColourItself is the glyph utterance's half of the
// same table: standing bare there is no fill to knock white out of, so the
// sign is the system colour. Neutral has none, so it takes the platform's
// secondary label — the strength the platform gives a word that is not the
// subject.
func TestABareSignIsTheSystemColourItself(t *testing.T) {
	for _, sc := range goldenSchemes {
		for _, st := range goldenStatuses {
			want := badge.Fill(sc.p, st.status)
			if st.status == badge.Neutral {
				want = vgcolor.Flatten(sc.p.SecondaryLabel, sc.p.WindowBackground)
			}
			if got := badge.BareForeground(sc.p, st.status, sc.p.WindowBackground); got != want {
				t.Errorf("%s %s: BareForeground = %v, want %v", sc.name, st.label, got, want)
			}
		}
	}
}

// TestTheFiveStayFive is what makes a set of badges a set: no two of the five
// resolve to one colour in either appearance, filled or bare. Hue is the one
// channel that separates them once the shape is the same, so a collision here
// is two statuses a reader cannot tell apart.
func TestTheFiveStayFive(t *testing.T) {
	for _, sc := range goldenSchemes {
		for _, get := range []struct {
			name string
			fn   func(tokens.PlatformColors, badge.Status) color.NRGBA
		}{
			{"Fill", badge.Fill},
			{"BareForeground", func(p tokens.PlatformColors, st badge.Status) color.NRGBA {
				return badge.BareForeground(p, st, p.WindowBackground)
			}},
		} {
			seen := map[color.NRGBA]string{}
			for _, st := range goldenStatuses {
				c := get.fn(sc.p, st.status)
				if prev, dup := seen[c]; dup {
					t.Errorf("%s %s: %s and %s are both %v", sc.name, get.name, prev, st.label, c)
				}
				seen[c] = st.label
			}
		}
	}
}
