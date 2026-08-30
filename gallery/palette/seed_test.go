package palette

import (
	stdcolor "image/color"
	"strings"
	"testing"

	"github.com/vibrantgio/theme/tokens"
)

// seedLines is every name and rule the shared row can put on screen, and
// seedClauses every caption clause it can put in a band. The guards below are
// run over these two sets rather than over whatever a case happens to produce,
// so a string added here without a home in one of them is a string nothing
// checks.
var (
	seedNames = []string{SeedName, SeedLiftedName, SeedLiftedNameDark}
	seedRules = []string{
		SeedGrewFrom, SeedPickRule,
		SeedLiftedRule, SeedLiftedRuleDark,
		SeedKeptRule, SeedKeptRuleDark,
	}
	seedClauses = []string{SeedHintPair, SeedHintHue, SeedHintChroma, SeedHintStatus}
)

// TestSeedLinesCarryNoUnmarkedSeam is the guard the whole row rests on.
// [FitLine] has two ways to shorten a line: at a clause seam — a comma, " ·"
// or " /" — with nothing at all marking the cut, and at a word boundary with
// an ellipsis. A line with a seam in it is a line a narrow window can shorten
// into a different claim without saying that it did, and there is no wording
// of a comma a reader can be relied on to supply back.
//
// So no name and no rule here carries one, and every cut a reader is ever
// shown ends in an ellipsis.
func TestSeedLinesCarryNoUnmarkedSeam(t *testing.T) {
	for _, line := range append(append([]string{}, seedNames...), seedRules...) {
		if heads := LineHeads(line, true); len(heads) > 0 {
			t.Errorf("%q can be cut to %q with nothing marking the cut", line, heads[0])
		}
	}
	// The caption is the one thing written as clauses, and is cut at the
	// separator its own list is strung on — so no clause may carry it.
	for _, clause := range seedClauses {
		if strings.Contains(clause, HintSep) {
			t.Errorf("the caption clause %q carries the separator its own list is strung on", clause)
		}
	}
}

// TestSeedClaimLeadsEveryRuleThatMakesIt: a cut takes words off the tail, so a
// claim that leads is a claim that survives every cut a reader is shown a mark
// for. Every rule entitled to say which colour the palette grew from opens
// with the words, and the one rule that is not entitled to say it does not say
// it anywhere.
func TestSeedClaimLeadsEveryRuleThatMakesIt(t *testing.T) {
	for _, rule := range []string{SeedLiftedRule, SeedLiftedRuleDark, SeedKeptRule, SeedKeptRuleDark} {
		if !strings.HasPrefix(rule, SeedGrewFrom) {
			t.Errorf("rule %q makes the claim somewhere other than the front, where a cut can take it off", rule)
		}
	}
	if strings.Contains(SeedPickRule, SeedGrewFrom) {
		t.Errorf("the pick's rule %q claims what the palette grew from, and the pick is not it", SeedPickRule)
	}
	// A dark scheme draws a re-toned accent, so the colour named as the one the
	// palette grew from is a colour the ramps below it draw nowhere. Both dark
	// rules disclose that, and inside the clause that makes the claim.
	for _, rule := range []string{SeedLiftedRuleDark, SeedKeptRuleDark} {
		if !strings.Contains(rule, "re-toned") {
			t.Errorf("dark rule %q does not disclose that this scheme re-tones the colour", rule)
		}
	}
	// And no rule names a colour value: values belong on the cell's own value
	// line, where a truncation cannot attach one colour's value to another
	// colour's claim.
	for _, rule := range seedRules {
		if strings.Contains(rule, "#") {
			t.Errorf("rule %q names a colour value", rule)
		}
	}
}

// TestSeedCellsNameWhatTheyShow checks the two-cell rule against the palette
// the cells are standing on rather than against itself: the colour under the
// pick's rule is the colour handed over byte for byte, and the colour under
// the claim is the base the light scheme actually pins.
func TestSeedCellsNameWhatTheyShow(t *testing.T) {
	// A seed the accent dial moves, which is the two-cell case, and one it
	// leaves alone, which is the one-cell case. Both are read off the
	// derivation rather than asserted, so a dial that stops moving the first
	// fails here instead of quietly turning this into one test twice.
	moved := tokens.DefaultSeed
	movedLight, movedDark := tokens.FromSeed(moved)
	if movedLight.Primary == moved {
		t.Fatalf("the seed %s is no longer moved by the accent dial; this test needs one that is", SeedHex(moved))
	}
	kept := stdcolor.NRGBA{R: 0xff, A: 0xff}
	keptLight, keptDark := tokens.FromSeed(kept)
	if keptLight.Primary != kept {
		t.Fatalf("the seed %s is moved by the accent dial; this test needs one it leaves alone", SeedHex(kept))
	}

	for _, tc := range []struct {
		name  string
		seed  stdcolor.NRGBA
		grown stdcolor.NRGBA
		dark  bool
		cells []SeedCell
	}{
		{"light, a pick the dial moved", moved, movedLight.Primary, false, []SeedCell{
			{Col: moved, Name: SeedName, Rule: SeedPickRule, HandedIn: true},
			{Col: movedLight.Primary, Name: SeedLiftedName, Rule: SeedLiftedRule},
		}},
		{"dark, a pick the dial moved", moved, movedLight.Primary, true, []SeedCell{
			{Col: moved, Name: SeedName, Rule: SeedPickRule, HandedIn: true},
			{Col: movedLight.Primary, Name: SeedLiftedNameDark, Rule: SeedLiftedRuleDark},
		}},
		{"light, the pick itself", kept, keptLight.Primary, false, []SeedCell{
			{Col: kept, Name: SeedName, Rule: SeedKeptRule},
		}},
		{"dark, the pick itself", kept, keptLight.Primary, true, []SeedCell{
			{Col: kept, Name: SeedName, Rule: SeedKeptRuleDark},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SeedCells(tc.seed, tc.grown, tc.dark, SeedLiftedNameDark)
			if len(got) != len(tc.cells) {
				t.Fatalf("the row draws %d cells, want %d", len(got), len(tc.cells))
			}
			for i, want := range tc.cells {
				if got[i] != want {
					t.Errorf("cell %d is {%s %q %q handedIn=%v}, want {%s %q %q handedIn=%v}",
						i, SeedHex(got[i].Col), got[i].Name, got[i].Rule, got[i].HandedIn,
						SeedHex(want.Col), want.Name, want.Rule, want.HandedIn)
				}
			}
		})
	}

	// The one thing the row may never do: show a colour that is not the pick
	// under the rule that says it is, or claim the palette grew from a colour
	// the palette does not pin.
	for _, tc := range []struct {
		seed stdcolor.NRGBA
		pair [2]tokens.ColorTokens
		dark bool
	}{
		{moved, [2]tokens.ColorTokens{movedLight, movedDark}, false},
		{moved, [2]tokens.ColorTokens{movedLight, movedDark}, true},
		{kept, [2]tokens.ColorTokens{keptLight, keptDark}, false},
		{kept, [2]tokens.ColorTokens{keptLight, keptDark}, true},
	} {
		lifted := tc.pair[0].Primary
		said := false
		for _, cell := range SeedCells(tc.seed, lifted, tc.dark, SeedLiftedNameDark) {
			if cell.Rule == SeedPickRule && cell.Col != tc.seed {
				t.Errorf("a cell showing %s is captioned %q, and the colour picked is %s",
					SeedHex(cell.Col), cell.Rule, SeedHex(tc.seed))
			}
			if strings.HasPrefix(cell.Rule, SeedGrewFrom) {
				said = true
				if cell.Col != lifted {
					t.Errorf("the cell claiming %q shows %s, and the light scheme pins %s",
						SeedGrewFrom, SeedHex(cell.Col), SeedHex(lifted))
				}
			}
		}
		if !said {
			t.Errorf("no cell opens with %q, so nothing on screen says which colour the palette grew from",
				SeedGrewFrom)
		}
	}
}

// TestSeedCaptionSizesItselfToWhatIsDrawn: the legend for the two sizes is
// only true where two cells are drawn, so it is only said there.
func TestSeedCaptionSizesItselfToWhatIsDrawn(t *testing.T) {
	moved := tokens.DefaultSeed
	light, _ := tokens.FromSeed(moved)
	pair := SeedHint(SeedCells(moved, light.Primary, false, SeedLiftedNameDark))
	if !strings.HasPrefix(pair, SeedHintPair) {
		t.Errorf("the two-cell caption is %q, want it to lead with the legend for the sizes", pair)
	}
	one := SeedHint(SeedCells(moved, moved, false, SeedLiftedNameDark))
	if strings.Contains(one, SeedHintPair) {
		t.Errorf("the one-cell caption is %q, want no legend for a size distinction it does not draw", one)
	}
	if !strings.HasPrefix(one, SeedHintHue) {
		t.Errorf("the one-cell caption is %q, want the derivation it does describe", one)
	}
}

// TestSeedCellHeightFollowsWhatItDraws: a cell with a colour writes its value
// out and takes the three-line slot the picks board gives a pair; a cell that
// is words takes the two-line one.
func TestSeedCellHeightFollowsWhatItDraws(t *testing.T) {
	if got := (SeedCell{Col: tokens.DefaultSeed}).Height(); got != PickPairH {
		t.Errorf("a cell with a colour is %v tall, want %v", got, PickPairH)
	}
	if got := (SeedCell{WordsOnly: true}).Height(); got != PickCellH {
		t.Errorf("a cell of words is %v tall, want %v", got, PickCellH)
	}
}
