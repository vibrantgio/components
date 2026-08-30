// seed.go is the head of the story the rest of this package draws: the one
// colour the ramps, the picks and the bases are all a function of. Until this
// row a palette section shows every derivation and never the input.
//
// It is drawn in the story's own vocabulary — the same heading band, the same
// body ground, cells the size of a picks-board cell with the same swatch, the
// same hairline frame and the same name-over-rules block — so it reads as the
// first cells of the story rather than as a second design. It takes the
// story's own geometry ([PickSwatchW], [PickPairH], [PickTitleH], [PickRuleH],
// [PickGap]) directly: a swatch above the board and a swatch on it are the
// same object shown twice, and a second set of numbers for it would be two
// answers to one question.
//
// Honesty is the whole of the difficulty, and it shaped the layout.
//
// # Why two colours and not one sentence
//
// A palette is not always grown from the colour somebody picked. The
// derivation realizes the light Primary base at the picked colour's own hue
// and depth with the accent chroma dial applied, so a pick under the dial
// comes back more chromatic than it was handed over, and it is the realized
// colour — not the pick — that every accent ramp and base is measured off.
//
// So the row has two colours to tell apart, and it tells them apart as two
// cells rather than inside one sentence. A sentence relating them has its only
// clause boundary between the two, so a narrow window cuts it to a line naming
// one colour and claiming the other's fact, with nothing marking the cut —
// which is the one claim this file exists never to make. Two cells, each with
// its own swatch and its own name, and no line carrying a relation a cut can
// invert.
//
// Every line here is written as one clause for the same reason. [FitLine] cuts
// a line at its commas and at " ·" and " /" and marks nothing when it does,
// and falls back to a word boundary with an ellipsis; a line with no clause
// seam in it can therefore only ever be cut the marked way. That is a
// structural guard rather than an editorial one, and it is what the strings
// below are written to: they say what they say without a comma, which is why
// they lean on "and" where a comma would read better.
//
// The claim leads. [SeedGrewFrom] is the sentence the section is named after,
// and every rule entitled to make it opens with it — a cut takes words off the
// tail, so a claim at the front survives every cut a reader is shown a mark
// for.
//
// # Why the pair is told apart by size as well
//
// The two colours are one hue at two chromas, which is a difference a
// reduced-chroma reader does not have: measured, the pair a default seed makes
// stands at 1.00:1 luminance and four greyscale levels apart, which is one
// swatch drawn twice. So the colour the palette only took in is drawn inside
// the slot the realized colour fills whole, smaller by [SeedHandedInset]. It
// is the grid's own device — [RampPinInset], which stops a pinned chip reading
// as a tenth step — turned on the one distinction this row exists to draw.
// Size is a channel no display setting takes away.
//
// # What this package does not decide
//
// Whether the palette on screen is the candidate's at all. A seed derives a
// fixed set of token sets, and a window is that seed's only if it wears one of
// them byte for byte — but which sets a window can be wearing, and whether a
// candidate's alpha has to be normalized before the comparison, are facts
// about the window rather than about the row: one window knows its pick
// first-hand and derives the pair it is drawn from, another infers a seed
// through the token sets an application theme can hand it. Handing those in as
// a parameter would be handing in the whole of the check, so the check stays
// with the window and [SeedCells] is given the answer: a seed and the colour
// it was realized as, both proven.
//
// For the same reason the cells a window draws where nothing is proven — or
// where nothing is picked yet — are the window's own, and so is the caption
// over them. What is here is the part that is one row wherever it is drawn.
package palette

import (
	"fmt"
	"image"
	stdcolor "image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/vibrantgio/textdraw"
	"github.com/vibrantgio/theme/tokens"
)

// SeedSectionRows is how many rows [SeedRows] returns: a heading and a body,
// the shape every section of this story has.
const SeedSectionRows = 2

// SeedHandedInset is how far inside its slot the swatch of a colour the
// palette only took in is drawn, leaving the full slot to the colour the
// palette realized. Four rather than the grid's three, because the shape it
// insets is the larger one and the ring it leaves has to be a ring at 1x.
const SeedHandedInset unit.Dp = 4

// What the section says about itself, as [HintSep] clauses so a narrow window
// drops it a clause at a time — see [FitHint]. A caption is a legend and not a
// claim under a swatch, and each of these stands alone.
//
// The derivation clauses say the hue and the chroma come from different places
// because they do, and the two cells under the caption are the proof. They say
// "the accent" rather than "every role" for the same reason: the neutral ramp
// carries no hue at all, and the four status roles are anchored to fixed hues
// the seed may only tint. They stop there rather than going on about ramps and
// bases, which are two bands further down under captions of their own.
const (
	SeedLabel = "Palette Seed"
	// SeedHintPair is the legend for the two sizes, and leads the caption
	// wherever there are two cells to size — the ramps' caption leads with the
	// legend for its dot the same way.
	SeedHintPair   = "the smaller swatch is the colour picked"
	SeedHintHue    = "the seed sets the accent hue"
	SeedHintChroma = "the palette's own dial sets its chroma"
	SeedHintStatus = "it only tints the status hues"
)

// The names over the cells, in the vocabulary the picks board already uses: it
// calls the realized colour "the seed, lifted", so the two cells are the seed
// and the seed lifted. The board's comma is not carried across — a name is cut
// by the same [FitLine] a rule is, and "Seed, lifted" cuts to "Seed", which is
// the other cell's name over the other cell's colour.
//
// The name is also where the dark scheme's disclosure lives, and for a reason
// a rule cannot serve. A cell draws three lines and each is cut on its own, so
// a fact on the name line and a fact on the rule line are shed at two
// different widths while two facts on one line are shed together. This row has
// two facts that must both survive — which colour the palette grew from, and
// that a dark scheme does not draw it — so they stand on two lines: the rule
// opens with the claim, the name carries the scheme. The name is the shorter
// line in the larger face, so it is the one that survives furthest.
//
// SeedName is only ever over a colour somebody picked. A window that cannot
// prove a pick names its cell for the token it is showing instead, in its own
// words.
const (
	SeedName       = "Seed"
	SeedLiftedName = "Lifted seed"
	// SeedLiftedNameDark names the same colour in the scheme that does not draw
	// it, in the words the picks board names a colour from across the pair in:
	// the other side's, and pinned there rather than here.
	//
	// It is a participle and not a relative clause. The draft read "Lifted seed
	// the light scheme pins", which is grammatical with the "that" dropped and
	// which a fresh-eyes pass read as a run-on with a clause missing its
	// punctuation — and punctuation is the one repair not available here, since
	// a comma in a name is a seam [FitLine] cuts at without marking. Recast as
	// one noun phrase it needs no seam and reads as one thing.
	//
	// It is the one line of the row [SeedCells] takes as a parameter rather
	// than reading from here, so that rewording a disclosure is a deliberate
	// change to what a window draws rather than one that arrives with a
	// version bump.
	SeedLiftedNameDark = "Lifted seed pinned in the light scheme"
)

// The rules under the names. Every one of them is one clause — no comma, no
// " ·", no " /" — so [FitLine] has no unmarked cut to make on any of them and
// every cut a reader is shown ends in an ellipsis.
const (
	// SeedGrewFrom is the claim the section is named after. Every rule that can
	// prove it opens with it, so no cut can take it off.
	SeedGrewFrom = "the colour this palette grew from"

	// SeedPickRule sits under the colour handed to the derivation, in the cell
	// whose neighbour carries what the derivation made of it. It is the one
	// rule here that may not open with SeedGrewFrom: this is the colour the
	// palette did not grow from.
	SeedPickRule = "the colour picked"

	// SeedLiftedRule and SeedLiftedRuleDark sit under the realized colour. They
	// differ because the fact differs: a light scheme pins this exact colour as
	// its Primary base — the chip at the end of the Primary row below is these
	// very bytes — while a dark scheme pins a re-toned one, so in dark the
	// swatch is a colour the palette on screen draws nowhere. The dark one says
	// so inside the clause that names it.
	SeedLiftedRule     = SeedGrewFrom + " and pins as its Primary base"
	SeedLiftedRuleDark = SeedGrewFrom + " before this scheme re-toned it"

	// SeedKeptRule and SeedKeptRuleDark are the one-cell case: the dial left
	// the pick alone, so the colour picked and the colour the palette grew from
	// are one colour and there is nothing to tell apart.
	SeedKeptRule = SeedGrewFrom + " and the colour picked and its Primary base"
	// The dark one names no scheme where the light one names none either: it is
	// already the longest line the row draws, and the room a narrow window
	// leaves is the budget every line here is written to.
	SeedKeptRuleDark = SeedGrewFrom + " and the colour picked before it was re-toned"
)

// SeedCell is one thing the row shows: a colour, what to call it, what it is,
// and whether the palette only took it in — which is drawn, as the smaller
// swatch. The value under the name is written out from the colour itself.
//
// WordsOnly is the cell with no colour at all, which is a row standing where a
// seed is not there to show: the name and the rule, and no swatch to stand for
// one.
type SeedCell struct {
	Col        stdcolor.NRGBA
	Name, Rule string
	HandedIn   bool
	WordsOnly  bool
}

// Height is the slot this cell takes: three lines where there is a colour to
// write out, two where the cell is words.
func (c SeedCell) Height() unit.Dp {
	if c.WordsOnly {
		return PickCellH
	}
	return PickPairH
}

// SeedCells is the two-cell rule, for a palette that has been proven to be
// seed's: grown is the colour seed was realized as, which is the light
// scheme's pinned base. Where the dial left the pick alone the two are one
// colour and the row draws one cell that says both things about it; where it
// did not, they are two cells and the pick is the one drawn smaller.
//
// liftedDark is the name over the realized colour in a dark scheme. The
// story's own is [SeedLiftedNameDark]; it is a parameter because that line is
// a disclosure a window may already be drawing in different words, and one
// wording silently replacing another is a change to what is on screen rather
// than to how it gets there.
func SeedCells(seed, grown stdcolor.NRGBA, dark bool, liftedDark string) []SeedCell {
	if grown == seed {
		return []SeedCell{{Col: grown, Name: SeedName,
			Rule: seedScheme(SeedKeptRule, SeedKeptRuleDark, dark)}}
	}
	return []SeedCell{
		{Col: seed, Name: SeedName, Rule: SeedPickRule, HandedIn: true},
		{Col: grown, Name: seedScheme(SeedLiftedName, liftedDark, dark),
			Rule: seedScheme(SeedLiftedRule, SeedLiftedRuleDark, dark)},
	}
}

// seedScheme chooses the wording for the side of the pair on screen. It is not
// called pick: in this story a pick is a colour the theme names, and the word
// is spoken for.
func seedScheme(light, dark string, isDark bool) string {
	if isDark {
		return dark
	}
	return light
}

// SeedHint is the caption for the cells actually drawn. The legend for the two
// sizes is only true where two cells are drawn, so it is only said there.
//
// A window with cells of its own to draw — a state rather than a derivation —
// captions them itself: this is the caption for the derivation, and a
// derivation nobody is looking at is not described.
func SeedHint(cells []SeedCell) string {
	clauses := []string{SeedHintHue, SeedHintChroma, SeedHintStatus}
	if len(cells) > 1 {
		clauses = append([]string{SeedHintPair}, clauses...)
	}
	return strings.Join(clauses, HintSep)
}

// SeedRows is the head of the palette story as rows of a page's column: the
// band, and under it the cells the caller has to show.
func SeedRows(p Chrome, c tokens.ColorTokens, ty Type, cells []SeedCell, hint string) []layout.Widget {
	return []layout.Widget{
		Heading(p, c, ty, SeedLabel, hint),
		Body(c, seedBody(p, c, ty, cells)),
	}
}

// seedBody draws the cells stacked: each colour, and beside it the name over
// its value over its rule. The geometry is a paired picks cell's, taken from
// the same constants, so the head of the story lines up with the story.
func seedBody(p Chrome, c tokens.ColorTokens, ty Type, cells []SeedCell) func(gtx layout.Context, width int) int {
	return func(gtx layout.Context, width int) int {
		y := 0
		for _, cell := range cells {
			h := gtx.Dp(cell.Height())
			if width > 0 {
				drawSeedCell(gtx, p, c, ty, cell, image.Rect(0, y, width, y+h))
			}
			y += h
		}
		return y
	}
}

// drawSeedCell draws one cell in the slot it was given, the way the picks
// board draws a paired one: the colour, and a block of lines beside it.
//
// The colour sits in a slot the width of a picks swatch whichever cell this
// is, and the lines start off the slot rather than off the colour, so two
// cells keep one text column however their swatches are drawn. A cell that is
// words takes no slot at all — an empty box where a swatch belongs reads as a
// swatch that failed to draw.
func drawSeedCell(gtx layout.Context, p Chrome, c tokens.ColorTokens, ty Type, cell SeedCell, r image.Rectangle) {
	if r.Dx() <= 0 {
		return
	}
	lines := r.Min.X
	body := []string{cell.Rule}
	if !cell.WordsOnly {
		sw, sh := min(gtx.Dp(PickSwatchW), r.Dx()), min(gtx.Dp(PickSwatchH), r.Dy())
		top := r.Min.Y + (r.Dy()-sh)/2
		slot := image.Rect(r.Min.X, top, r.Min.X+sw, top+sh)
		box := slot
		// The second channel: a colour the palette only took in is drawn inside
		// the slot the colour it realized fills whole.
		if cell.HandedIn {
			if in := slot.Inset(gtx.Dp(SeedHandedInset)); !in.Empty() {
				box = in
			}
		}
		radius := gtx.Dp(innerR) / 2
		fillRRect(gtx, box, radius, cell.Col)
		// The same frame every swatch in this story wears — see [EdgeIn] — and
		// here for the reason it is there: a colour near the tone of the ground
		// it stands on has no boundary of its own.
		strokeRRect(gtx, box, radius, gtx.Dp(hairline), EdgeIn(c))
		lines = slot.Max.X + gtx.Dp(PickGap)
		body = []string{SeedHex(cell.Col), cell.Rule}
	}
	if lines >= r.Max.X {
		return
	}
	room := r.Max.X - lines
	title, rule := gtx.Dp(PickTitleH), gtx.Dp(PickRuleH)
	block := min(title+len(body)*rule, r.Dy())
	y := r.Min.Y + (r.Dy()-block)/2
	textdraw.FillText(gtx, ty.Shaper, ty.Body,
		image.Rect(lines, y, r.Max.X, y+title), 0, 0.5, p.Text,
		FitLine(gtx, ty.Shaper, ty.Body, cell.Name, room))
	y += title
	for _, line := range body {
		textdraw.FillText(gtx, ty.Shaper, ty.Small,
			image.Rect(lines, y, r.Max.X, y+rule), 0, 0.5, p.Muted,
			FitLine(gtx, ty.Shaper, ty.Small, line, room))
		y += rule
	}
}

// SeedHex writes a colour the way a stylesheet writes one: uppercase, and with
// no alpha, since every colour this row draws is opaque.
func SeedHex(c stdcolor.NRGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
