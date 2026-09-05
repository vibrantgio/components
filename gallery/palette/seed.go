// seed.go is the head of the story the rest of this package draws: the one
// colour the ramps, the picks and the bases are all a function of.
//
// It is drawn in the story's own vocabulary and geometry ([PickSwatchW],
// [PickPairH], [PickTitleH], [PickRuleH], [PickGap]) directly, so it reads as
// the first cells of the story rather than a second design.
//
// A palette is not always grown from the colour somebody picked: the
// derivation realizes the light Primary base at the picked colour's own hue
// and depth with the accent chroma dial applied, so a pick under the dial
// comes back more chromatic than it was handed over, and it is the realized
// colour — not the pick — that every accent ramp and base is measured off.
// The row tells the two colours apart as two cells rather than in one
// sentence, since a sentence relating them has a clause boundary between the
// two that a narrow window could cut without marking, misattributing one
// colour's fact to the other. For the same reason every line in this file is
// written as one clause with no comma, no " ·" and no " /", so [FitLine] has
// no unmarked seam to cut on any of them. [SeedGrewFrom] leads every rule
// entitled to make it, since a cut takes words off the tail and a claim at
// the front survives every cut.
//
// The pair is also told apart by size: the two colours are one hue at two
// chromas — measured, the pair a default seed makes stands at 1.00:1
// luminance and four greyscale levels apart, indistinguishable to a
// reduced-chroma reader as a swatch drawn twice. So the colour the palette
// only took in is drawn inside the slot the realized colour fills whole,
// smaller by [SeedHandedInset] — the same device as the ramp grid's
// [RampPinInset] — since size is a channel no display setting removes.
//
// This file does not decide whether the palette on screen is the candidate's
// at all: which token sets a window can be wearing, and whether a candidate's
// alpha needs normalizing before comparison, are facts about the window, not
// the row. [SeedCells] is handed the answer already proven — a seed and the
// colour it was realized as — and a window with nothing proven, or nothing
// picked yet, draws and captions its own cells instead.
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
// drops it a clause at a time — see [FitHint]. The hue and chroma clauses say
// "the accent" rather than "every role": the neutral ramp carries no hue, and
// the four status roles are anchored to fixed hues the seed may only tint.
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

// The names over the cells, in the vocabulary the picks board already uses:
// it calls the realized colour "the seed, lifted", so the two cells are the
// seed and the seed lifted. The board's comma is not carried across, since a
// name is cut by the same [FitLine] a rule is and "Seed, lifted" would cut to
// "Seed" — the other cell's name.
//
// The name line is also where the dark scheme's disclosure lives rather than
// the rule line: a cell's three lines are each cut independently, and this
// row has two facts that must both survive a cut — which colour the palette
// grew from, and that a dark scheme does not draw it — so they stand on
// separate lines, with the shorter name line (larger face) surviving furthest.
//
// SeedName is only ever over a colour somebody picked; a window that cannot
// prove a pick names its cell for the token it is showing instead.
const (
	SeedName       = "Seed"
	SeedLiftedName = "Lifted seed"
	// SeedLiftedNameDark names the same colour in the scheme that does not draw
	// it, in the words the picks board uses for a colour from across the pair:
	// the other side's, pinned there rather than here. Written as one noun
	// phrase rather than a relative clause, since a comma would be a seam
	// [FitLine] cuts at without marking.
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

	// SeedLiftedRule and SeedLiftedRuleDark sit under the realized colour. A
	// light scheme pins this exact colour as its Primary base (the chip at the
	// end of the Primary row below is these very bytes), while a dark scheme
	// pins a re-toned one, so the dark rule discloses that inside its clause.
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
func SeedCells(seed, grown stdcolor.NRGBA, dark bool) []SeedCell {
	if grown == seed {
		return []SeedCell{{Col: grown, Name: SeedName,
			Rule: seedScheme(SeedKeptRule, SeedKeptRuleDark, dark)}}
	}
	return []SeedCell{
		{Col: seed, Name: SeedName, Rule: SeedPickRule, HandedIn: true},
		{Col: grown, Name: seedScheme(SeedLiftedName, SeedLiftedNameDark, dark),
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

// SeedHint is the caption for the cells actually drawn. The legend for the
// two sizes is only true where two cells are drawn, so it is only said there.
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
		// here for the reason it is there: a colour near the tone of the
		// surface it stands on has no boundary of its own.
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
