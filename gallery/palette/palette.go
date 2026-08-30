// Package palette draws the palette story: the ramps a theme has to pick
// from, and the colours it picked, each beside the rule that chose it.
//
// Ramps stand above picks, so a pick naming a rung ("Neutral 300") reads
// against the row it came from. Every function is a pure function of the
// [tokens.ColorTokens] handed in, plus the furniture colours and type roles a
// caller states in [Chrome] and [Type]; nothing reads a default palette, so
// the same code draws either scheme and a test can capture it without a
// window.
//
// Rules are read off the colours themselves — a pin compared against its own
// ramp, an ink against the two ends of the tonal axis and its role's deepest
// step — rather than written down statically, so a derivation change updates
// what this package says in the same build.
//
// A base and its ink are one cell because they are one decision: the
// derivation pins the base, then measures both ends of the tonal axis over
// that exact colour and keeps the better one, so the ink cannot be understood
// apart from the fill it was measured against. Surface and Divider stand
// alone because the theme names no ink for either.
//
// Each ramp row ends with the base that role pinned, drawn as a chip because
// two roles (a light scheme's Primary, Secondary, Tertiary) pin a colour that
// is off its own ramp by construction. Where the pin is a rung exactly, the
// chip and the marked cell are the same colour; where the pin is
// indistinguishable from no rung at all, the dot moves onto the chip itself so
// every row still answers where its pinned colour lives. Neutral's chip is
// absent because Neutral pins no solid fill.
//
// Status containers and the two axis ends (white, black) get cells of their
// own because neither is a colour any ramp cell holds: a container is its
// role's hue at a rung's tone with the chroma pulled to the container dial,
// and white/black belong to no ramp.
//
// Deliberately out of scope: interaction-state colours (hover, pressed,
// selected, dragged — a component's own transform of a colour it was given),
// disabled colours (an alpha fraction, not a palette member), and the focus
// ring (Neutral 500, already a cell here). Everything this package draws is a
// colour some widget is painted with at rest.
package palette

import (
	"fmt"
	"image"
	stdcolor "image/color"
	"math"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/gallery/inventory"
	"github.com/vibrantgio/textdraw"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// Chrome is the story's own furniture: the four colours it draws with that
// are not the palette it is describing. Stated by the caller rather than
// derived here, since a heading band belongs to the page the story sits in.
// Exactly four fields — everything else this package draws is a colour of the
// palette itself, handed over as [tokens.ColorTokens].
type Chrome struct {
	// Surface is the ground a section's heading band is filled with.
	Surface stdcolor.NRGBA
	// Divider is the rule under a family's name on the picks board.
	Divider stdcolor.NRGBA
	// Text is the reading ink: section labels, their captions, the ramp names
	// and the step numbers over the grid.
	Text stdcolor.NRGBA
	// Muted is the quiet register: a pick's rule under its name, and the mark
	// standing where a role pins nothing.
	Muted stdcolor.NRGBA
}

// Type is the story's view of the theme's typography: the roles it draws
// directly, as textdraw styles, plus the shaper it draws them with. The
// shaper is a field rather than read off a [tokens.Typography] here, since
// callers may need either the theme's cached shaper or a deterministic one
// for reproducible captures.
type Type struct {
	Shaper *text.Shaper
	Head   textdraw.TextStyle // family names on the picks board
	Label  textdraw.TextStyle // section labels, the pair's "Aa"
	Body   textdraw.TextStyle // a cell's names
	Small  textdraw.TextStyle // rules, step numbers, captions
}

// The furniture measurements the story shares with the pages it stands in.
const (
	// captionGap is what a section's caption stands off the title beside it.
	captionGap unit.Dp = 14
	// hairline is a resting outline: the rule under a heading band, the frame
	// round a swatch.
	hairline unit.Dp = 1
	// innerR is a swatch and chip corner.
	innerR unit.Dp = 8
	// ellipsis is the mark a run of text wears when it was cut short. One
	// mark, so a reader meets one sign for one fact wherever a line stopped
	// early.
	ellipsis = "…"
)

// The section's dimensions.
//
// The heading rows are the column's own furniture at the column's own sizes:
// this story stands among a page of labelled sections, and a heading a few
// points off the ones around it reads as a heading from somewhere else.
// Everything inside them is measured to what it holds.
const (
	// SectionHeadH is a section heading, at the height a page of labelled
	// sections gives its own.
	SectionHeadH unit.Dp = 32

	// RampSteps is how many steps a ramp has. It is the length of the row and
	// the count of the numbers over it, and it is one name rather than a 9
	// written down in three places.
	RampSteps = 9
	// RampLabelW is the column the ramp names stand in, at the left of the
	// grid, and RampGutter what stands between the longest of them and the first
	// swatch. The names are ranged against the grid, so the gutter is a fixed
	// distance rather than whatever a name's own length leaves over.
	RampLabelW unit.Dp = 88
	RampGutter unit.Dp = 14
	// RampRowH is one ramp, RampHeadH the line of step numbers above the
	// columns. A row is a band rather than a tile: nine of them stacked is the
	// whole scale at once, and height spent on any one of them is height the
	// picks below have to be scrolled to.
	RampRowH  unit.Dp = 24
	RampHeadH unit.Dp = 18
	// A cell has no maximum: the nine steps divide whatever the row has left
	// once the names at one end and the chip at the other are reserved, so the
	// row fills its width at every window rather than leaving a ragged edge or
	// a gap ahead of the trailing chip.
	//
	// RampMark is the dot on a rung a pick sits at. It is a dot and not a ring, a
	// label or a heavier frame: a fifth of the cells carry one, and anything with
	// an edge of its own at that count turns a table of colour into a table of
	// marks.
	RampMark unit.Dp = 7

	// RampPinW is the chip at the trailing end of a row holding the base that
	// role pinned, RampPinGap the least air there may ever be between it and
	// step 900, and RampPinInset how far it stands off the row's top and bottom.
	// The gap is wider than any space inside the grid and the inset makes the
	// chip shorter than a cell, which between them are what stop nine steps and
	// a pin from reading as ten steps. The chip is the width of the swatch a
	// pick carries below: the same colour shown twice, so it takes one size.
	//
	// A least and not a fixed distance: the chips are ranged against the grid's
	// trailing edge rather than set one gap past step 900, since nine cells of a
	// whole number of points rarely divide the row exactly and the slack (up to
	// eight points) lands here instead of past the chip.
	RampPinW     unit.Dp = 44
	RampPinGap   unit.Dp = 14
	RampPinInset unit.Dp = 3

	// PickSwatchW and PickSwatchH are one cell's colour. It is wide enough to
	// carry two letters at the size a page sets its specimens in, because most
	// of these swatches carry an ink and an ink shown smaller than the claim
	// next to it is a claim nobody can check — and tall enough that the letters
	// have air above and below rather than reaching the edge, which on a chip
	// this small is the difference between a specimen and a stamp.
	PickSwatchW unit.Dp = 44
	PickSwatchH unit.Dp = 26
	// PickPairH is a base and its ink, which carry three lines: the two names,
	// and a rule for each. PickCellH is a colour that has no ink, which carries
	// two. The difference between either and the lines inside it is the air one
	// cell keeps from the next.
	PickPairH unit.Dp = 62
	PickCellH unit.Dp = 44
	// PickTitleH is the line the names are on and PickRuleH one rule under them.
	PickTitleH unit.Dp = 18
	PickRuleH  unit.Dp = 15
	// PickHeadH is the name over one family of cells and PickHeadGap the air
	// under the line that follows it. PickGroupGap is what stands above the
	// name, and it is the larger of the two by a long way: a heading equidistant
	// between the family it ends and the family it starts belongs to neither.
	PickHeadH    unit.Dp = 20
	PickHeadGap  unit.Dp = 8
	PickGroupGap unit.Dp = 22
	// PickGap is swatch to text.
	PickGap unit.Dp = 10
	// PickColGap is between one column of cells and the next. How narrow a
	// column may be is not written down here — it is measured off the names the
	// board is about to draw, see [Narrowest].
	PickColGap unit.Dp = 24
	// PickMaxCols is as wide as the board spreads. Four families across a
	// window is a row of short lists rather than a board.
	PickMaxCols = 3
)

// What the story says about itself. A ramp is a scale to pick from and a pick
// is a colour taken off one.
//
// RampsHint is ordered so the mark's legend survives truncation before the
// scale clause does: the column headers already print 100–900 and which end
// is nearest the page is visible on sight, so a caption cut to one clause
// keeps the one fact nothing else on screen states — what the dot means.
//
// Step 100 is the palest rung in a light theme and the deepest in a dark one,
// and it is the same step in both (the one nearest the page), which is why a
// component asking for 100 gets a tint on either side of the scheme switch.
//
// A pin indistinguishable from no rung carries its dot on the chip instead of
// a cell; that per-pin distinction (some accents are pinned a hair off their
// own 700) lives on the cell's own rule rather than in the caption.
const (
	RampsLabel = "Palette Ramps"
	RampsHint  = "a dot marks where each pick lives · nine steps a role · 100 nearest the page · each row ends with its role's pinned base, and Neutral pins none"
	PicksLabel = "Palette Picks"
	PicksHint  = "every colour the theme names, and where it came from"
	// HintSep joins one caption clause to the next and is the seam [FitHint]
	// truncates at.
	HintSep = " · "
	// RampPinHead labels the chip column, in the same word the picks' rules use
	// for a pinned colour.
	RampPinHead = "base"
	// RampPinNone marks the one row that pins nothing (Neutral).
	RampPinNone = "—"
)

// The families the cells are read in: page and surfaces (the ground
// everything else stands on) and the inverse pair (also surfaces, borrowed
// from the other side of the scheme) first, then the accents the seed
// rotates, then the status roles it may only tint. Containers have no family
// of their own — each stands under its role, inside Status. The tonal axis
// ends come last: they are what the inks above turned out to be.
const (
	PickPageGroup    = "Page and surfaces"
	PickInverseGroup = "Inverse"
	PickAccentGroup  = "Accents"
	PickStatusGroup  = "Status"
	PickAxisGroup    = "Ink ends"
)

// The role names, said once. They are the ramp rows' labels and the cells'
// names both, and the same role named two ways in one section would read as two
// roles.
const (
	NeutralName   = "Neutral"
	PrimaryName   = "Primary"
	SecondaryName = "Secondary"
	TertiaryName  = "Tertiary"
	ErrorName     = "Error"
	SuccessName   = "Success"
	WarningName   = "Warning"
	InfoName      = "Info"
)

// The rules, as they are written under a pair of names.
const (
	PickGlyph = "Aa"
	// PickPairSep joins a cell's fill name to its ink name.
	PickPairSep = " / "
	// PickMeasured closes an ink's rule when the ink's name already names the
	// role (a rung of that role's own ramp — a dark scheme's inks).
	// PickMeasuredOver closes it when the ink is white or black, which name no
	// role, so the base is named instead.
	PickMeasured     = ", measured over the base"
	PickMeasuredOver = ", measured over %s"
	PickWhite        = "white"
	PickBlack        = "black"
	// PickMeasuredOn is an ink that is neither an axis end nor a rung of its own
	// ramp; no derivation shipping today produces this case.
	PickMeasuredOn = "measured over the base"
	// PickSeed is the light scheme's Primary where the chosen colour fell
	// between rungs: lifted onto the palette's own chroma, pinned at its own
	// depth.
	PickSeed = "the seed, lifted"
	// PickJustOff is a base pinned one unit of lightness off its own 700 rung
	// (the light scheme's accents) — named by that rung, since it is
	// indistinguishable from it on the grid.
	PickJustOff = "pinned just off %s %d"
	// PickSeedNear is PickSeed's rule where the lifted seed lands beside a rung
	// rather than on one.
	PickSeedNear = "the seed, lifted, just off %s %d"
	// PickPinned is a base near no rung at all; no derivation shipping today
	// produces this case for any role but Primary.
	PickPinned = "pinned off the ramp"
	// PickOffRamp is a neutral resolution off the neutral ramp; no derivation
	// shipping today produces this case.
	PickOffRamp = "off the neutral ramp"
	// The side of the scheme the caller is not showing, filled in with that
	// side's own role name: a light window's inverse surface is the dark
	// scheme's Surface exactly, so it is named as that role rather than as a
	// step of the wrong ramp.
	PickOtherLight = "the light scheme's %s"
	PickOtherDark  = "the dark scheme's %s"
	PickOtherSide  = "the other scheme's %s"
	// PickSurfaceRole and PickTextRole are the counterpart roles the inverse
	// pair resolves from.
	PickSurfaceRole = "Surface"
	PickTextRole    = "Text"
	// PickContainerRule is a status container: its role's own rung, kept at
	// that rung's tone and hue with the chroma pulled down to the containers'
	// shared chroma. Naming the rung is what makes it findable on the grid.
	PickContainerRule = "%s %d, held at the container chroma"
	// PickMarkRule is the mark read on a container: a rung of the role's own
	// ramp, chosen against the container rather than against a page.
	PickMarkRule = "%s %d, measured over the container"
	// PickMarkOff is a mark that is not a rung of its own ramp; no derivation
	// shipping today produces this case.
	PickMarkOff = "measured over the container"
	// The two ends of the tonal axis, on no ramp, each named for the end it is
	// and whether the scheme on screen writes any ink in it — read off that
	// scheme's own inks rather than asserted, so the answer turns over with the
	// scheme switch.
	PickAxisLight = "the tonal axis's light end"
	PickAxisDark  = "the tonal axis's dark end"
	PickAxisInk   = "%s, an ink here"
	PickAxisNoInk = "%s, no ink here"
)

// The token names the cells carry: the names in the theme's own source, so
// this package describes a palette a reader can look up.
//
// Container and Mark are names this section builds rather than reads, since
// the theme derives a container from a role on request rather than holding a
// field for one. Mark departs deliberately from the theme's On-something
// naming: an On-colour is text measured against a text contrast floor, while
// a mark is a graphic (icon, leading edge, rule) measured against a lower
// one, so naming it OnErrorContainer would claim it is text when the cell
// draws it as a shape.
const (
	BackgroundPick       = "Background"
	TextPick             = "Text"
	SurfacePick          = "Surface"
	DividerPick          = "Divider"
	InverseSurfacePick   = "InverseSurface"
	OnInverseSurfacePick = "OnInverseSurface"
	ContainerPick        = "Container"
	MarkPick             = "Mark"
	WhitePick            = "White"
	BlackPick            = "Black"
)

// RampRow is one row of the ramps grid: a role's name, the nine steps generated
// for it, and the base the derivation pinned for it — which is a colour of its
// own and not always one of the nine. A row whose role has no pinned base
// carries a transparent one, and a transparent chip is not drawn.
type RampRow struct {
	Name string
	Ramp tokens.Ramp
	Pin  stdcolor.NRGBA
}

// RampRows is the grid, in the order it is read: the seed's own roles first
// (Primary, then Secondary and Tertiary rotated off it — the rows a choice
// moves), then the status roles (anchored hues the seed only tints), then
// Neutral last, since it is the row the seed has least to say about.
func RampRows(c tokens.ColorTokens) []RampRow {
	return []RampRow{
		{PrimaryName, c.Ramps.Primary, c.Primary},
		{SecondaryName, c.Ramps.Secondary, c.Secondary},
		{TertiaryName, c.Ramps.Tertiary, c.Tertiary},
		{ErrorName, c.Ramps.Error, c.Error},
		{SuccessName, c.Ramps.Success, c.Success},
		{WarningName, c.Ramps.Warning, c.Warning},
		{InfoName, c.Ramps.Info, c.Info},
		// Neutral pins no solid fill, so its chip is the one the grid leaves
		// empty rather than a colour invented to fill the column.
		{NeutralName, c.Ramps.Neutral, stdcolor.NRGBA{}},
	}
}

// Claim is one rung a pick took: the row it is on and the step it is.
type Claim struct {
	Role string
	Step int
}

// Claims is every rung the picks below the grid took, which is what the grid
// marks. It is read off the picks rather than tracked separately, so a rule
// and its marker cannot disagree.
func Claims(groups []Group) map[Claim]bool {
	out := map[Claim]bool{}
	for _, g := range groups {
		for _, cell := range g.Cells {
			for _, part := range [2]Part{cell.Base, cell.Ink} {
				if part.Role != "" && part.Step != 0 {
					out[Claim{part.Role, part.Step}] = true
				}
			}
		}
	}
	return out
}

// Part is one colour token in a cell: what the theme calls it, what chose it,
// and — where what chose it was a rung of a ramp — which row and which step,
// so the grid above can mark the rung this pick took. Rung and rule must be
// resolved together, never separately, or a rule and its marker can disagree.
type Part struct {
	Name, Rule string
	Role       string // the ramp row the rule names, empty when it names none
	Step       int    // the rung on that row, 0 when the colour is on none
}

// Cell is one thing the palette decided. It is a base and the ink measured
// over it — one swatch, two names, two rules — or, where the theme names no ink
// for a colour, that colour on its own.
//
// Mark says the second colour is a mark and not an ink: it is drawn as a shape
// over the fill rather than as two letters, because it was chosen against the
// non-text floor and letters would claim a legibility nothing measured.
type Cell struct {
	Base, Ink Part
	Fill, On  stdcolor.NRGBA
	Mark      bool
}

// Paired reports whether this cell carries an ink as well as a fill.
func (c Cell) Paired() bool { return c.Ink.Name != "" }

// Title is the cell's names, in the order their rules are written under them.
func (c Cell) Title() string {
	if !c.Paired() {
		return c.Base.Name
	}
	return c.Base.Name + PickPairSep + c.Ink.Name
}

// Height is the slot this cell takes: three lines for a pair and two for a
// colour standing on its own.
func (c Cell) Height() unit.Dp {
	if c.Paired() {
		return PickPairH
	}
	return PickCellH
}

// Group is one family of cells under its own name.
type Group struct {
	Name  string
	Cells []Cell
}

// Groups is every colour token the theme names, grouped as they are read, with
// each rule resolved against the colours themselves. c is the side of the pair
// being drawn and other is the side it is not — only the inverse pair reads
// other, and it is handed the counterpart rather than deriving one, so the
// caller need not build a palette of its own.
func Groups(c, other tokens.ColorTokens, dark bool) []Group {
	n := c.Ramps.Neutral
	alone := func(base Part, fill stdcolor.NRGBA) Cell {
		return Cell{Base: base, Fill: fill}
	}
	groups := []Group{
		{PickPageGroup, []Cell{
			// The page and the ink it is read in: the one pair in the theme
			// that is two pins rather than a pin and a measurement.
			{
				Base: neutralPart(BackgroundPick, n, c.Background),
				Ink:  neutralPart(TextPick, n, c.Text),
				Fill: c.Background, On: c.Text,
			},
			alone(neutralPart(SurfacePick, n, c.Surface), c.Surface),
			alone(neutralPart(DividerPick, n, c.Divider), c.Divider),
		}},
		{PickInverseGroup, []Cell{{
			Base: inversePart(InverseSurfacePick, c.InverseSurface, other.Surface, PickSurfaceRole, dark),
			Ink:  inversePart(OnInverseSurfacePick, c.OnInverseSurface, other.Text, PickTextRole, dark),
			Fill: c.InverseSurface, On: c.OnInverseSurface,
		}}},
		{PickAccentGroup, []Cell{
			pinnedCell(PrimaryName, c.Ramps.Primary, c.Primary, c.OnPrimary, PickSeedNear, PickSeed),
			pinnedCell(SecondaryName, c.Ramps.Secondary, c.Secondary, c.OnSecondary, PickJustOff, PickPinned),
			pinnedCell(TertiaryName, c.Ramps.Tertiary, c.Tertiary, c.OnTertiary, PickJustOff, PickPinned),
		}},
		// Each status role twice over: the solid fill it puts a label on, and
		// under it the container it fills a band with plus the mark read on
		// that. The container stands under its own role rather than in a
		// family of its own, since in a light scheme its mark is that role's
		// own 700 — the same colour as the fill directly above it.
		{PickStatusGroup, []Cell{
			pinnedCell(ErrorName, c.Ramps.Error, c.Error, c.OnError, PickJustOff, PickPinned),
			containerCell(ErrorName, c, tokens.RoleError, c.Ramps.Error),
			pinnedCell(SuccessName, c.Ramps.Success, c.Success, c.OnSuccess, PickJustOff, PickPinned),
			containerCell(SuccessName, c, tokens.RoleSuccess, c.Ramps.Success),
			pinnedCell(WarningName, c.Ramps.Warning, c.Warning, c.OnWarning, PickJustOff, PickPinned),
			containerCell(WarningName, c, tokens.RoleWarning, c.Ramps.Warning),
			pinnedCell(InfoName, c.Ramps.Info, c.Info, c.OnInfo, PickJustOff, PickPinned),
			containerCell(InfoName, c, tokens.RoleInfo, c.Ramps.Info),
		}},
	}
	// The two colours every ink above was chosen between, shown as colours
	// rather than as letterforms, each told whether this scheme writes
	// anything in it — read off the families already built rather than
	// asserted.
	return append(groups, Group{PickAxisGroup, []Cell{
		alone(axisPart(WhitePick, PickAxisLight, tokens.White, groups), tokens.White),
		alone(axisPart(BlackPick, PickAxisDark, tokens.Black, groups), tokens.Black),
	}})
}

// axisPart is one end of the tonal axis as a cell carries it: which end it is,
// and whether anything above it on the board is written in it.
func axisPart(name, end string, col stdcolor.NRGBA, groups []Group) Part {
	rule := PickAxisNoInk
	if inkedWith(groups, col) {
		rule = PickAxisInk
	}
	return Part{Name: name, Rule: fmt.Sprintf(rule, end)}
}

// inkedWith reports whether any cell of these families is written in col.
func inkedWith(groups []Group, col stdcolor.NRGBA) bool {
	for _, g := range groups {
		for _, cell := range g.Cells {
			if cell.Paired() && cell.On == col {
				return true
			}
		}
	}
	return false
}

// containerCell is one status role's tonal container and the mark read on it:
// one ground, one mark, and the rule each was derived by. The pair is one cell
// for the reason a base and its ink are — the mark is chosen against this exact
// ground and cannot be understood apart from it — and the mark is drawn as a
// disc because that is the kind of thing it was measured to be.
func containerCell(role string, c tokens.ColorTokens, id tokens.Role, r tokens.Ramp) Cell {
	ground, mark := c.StatusContainer(id), c.OnStatusContainer(id)
	return Cell{
		Base: containerPart(role, r, ground),
		Ink:  markPart(role, r, mark),
		Fill: ground, On: mark, Mark: true,
	}
}

// containerPart is a container as a cell carries it: the rung it was realized
// at, and what was done to that rung to get it.
//
// The rung is found by tone (lightness) rather than by colour: a container
// keeps its rung's lightness and hue but gives up chroma, so comparing bytes
// finds nothing and lightness is the one dimension that survives intact.
//
// It claims that rung unconditionally, even where the container is not close
// enough to be mistaken for it — the dot on the grid always marks the step a
// pick's rule names, and marking only the close containers would leave two of
// four rows with word-for-word identical rules dotted differently.
func containerPart(role string, r tokens.Ramp, ground stdcolor.NRGBA) Part {
	step := ToneStep(r, ground)
	return Part{
		Name: role + ContainerPick,
		Rule: fmt.Sprintf(PickContainerRule, role, step),
		Role: role,
		Step: step,
	}
}

// markPart is the mark read on a container: a rung of the role's own ramp,
// chosen against the container rather than against a page, so it claims that
// rung and the grid marks it.
func markPart(role string, r tokens.Ramp, mark stdcolor.NRGBA) Part {
	part := Part{Name: role + MarkPick, Role: role}
	if n := StepIn(r, mark); n != 0 {
		part.Rule, part.Step = fmt.Sprintf(PickMarkRule, role, n), n
		return part
	}
	part.Rule = PickMarkOff
	return part
}

// ToneStep is the rung of r a colour was realized at, read off the lightness
// the two share. See [containerPart] for why lightness is the question.
func ToneStep(r tokens.Ramp, col stdcolor.NRGBA) int {
	tone, _, _ := vgcolor.LabFromNRGBA(col)
	best, at := math.Inf(1), 0
	for i := range r {
		l, _, _ := vgcolor.LabFromNRGBA(r[i])
		if d := math.Abs(l - tone); d < best {
			best, at = d, (i+1)*100
		}
	}
	return at
}

// pinnedCell is one role's cell: the base the derivation pinned and the ink it
// measured over that exact colour, named OnPrimary etc. for the base it has
// to clear. near is what the base's rule says when it landed beside a rung
// rather than on one; off is what it says when it landed beside none.
func pinnedCell(role string, r tokens.Ramp, base, ink stdcolor.NRGBA, near, off string) Cell {
	return Cell{
		Base: BasePart(role, r, base, near, off),
		Ink:  inkPart(role, r, ink),
		Fill: base, On: ink,
	}
}

// StepIn reports which step of r the colour is, or 0 when it is not on the
// ramp at all. It compares bytes rather than measuring a distance: a pin that
// is a rung is that rung exactly, and a pin that is merely near one is a
// different colour.
func StepIn(r tokens.Ramp, col stdcolor.NRGBA) int {
	for i := range r {
		if r[i] == col {
			return (i + 1) * 100
		}
	}
	return 0
}

// BasePart is a pinned base as a cell carries it, in three cases: the rung it
// landed on, the rung it is indistinguishable from and how it came to be beside
// rather than on it (near, which takes the role and the rung), and — where no
// rung is near — how it was pinned and nothing else (off).
func BasePart(role string, r tokens.Ramp, col stdcolor.NRGBA, near, off string) Part {
	if n := StepIn(r, col); n != 0 {
		return Part{Name: role, Rule: fmt.Sprintf("%s %d", role, n), Role: role, Step: n}
	}
	if n := NearestStep(r, col); n != 0 {
		return Part{Name: role, Rule: fmt.Sprintf(near, role, n), Role: role, Step: n}
	}
	return Part{Name: role, Rule: off, Role: role}
}

// NearestStep is the rung of r a colour is indistinguishable from, or 0 when
// it is distinguishable from all nine. "On the ramp" and "the same colour as
// the ramp" are different questions; this answers the second, since every
// light-scheme accent is pinned one unit of lightness off its own 700 rung
// (three parts in 255 — invisible to eye or display).
//
// Distance is measured in OKLab, and [RungTolerance] is set by measurement:
// pins that should match sit up to 0.016 from their rung, and the closest two
// rungs of any ramp this derivation builds are 0.041 apart, so the tolerance
// must clear the first and stay under half the second or a colour could sit
// within reach of two rungs at once.
func NearestStep(r tokens.Ramp, col stdcolor.NRGBA) int {
	best, at := RungTolerance, 0
	for i := range r {
		if d := OKLabDistance(r[i], col); d < best {
			best, at = d, (i+1)*100
		}
	}
	return at
}

// RungTolerance is how far from a rung a colour may sit and still be that
// rung as far as anybody looking is concerned. See [NearestStep] for the two
// measurements it stands between.
const RungTolerance = 0.018

// PinRung is the rung a pinned base claims: the step it is exactly, else the
// one it is indistinguishable from, else 0 — the two questions [BasePart]
// resolves a base's rule by, asked in the order it asks them, so the grid and
// the rule cannot disagree about whether a pin lives on a rung. The one pin
// that claims nothing is a lifted seed whose depth falls between two steps of
// the scale, and its row's chip carries the dot instead — see [rampGrid].
func PinRung(r tokens.Ramp, pin stdcolor.NRGBA) int {
	if n := StepIn(r, pin); n != 0 {
		return n
	}
	return NearestStep(r, pin)
}

// OKLabDistance is how far apart two colours are, perceptually.
func OKLabDistance(a, b stdcolor.NRGBA) float64 {
	l1, a1, b1 := vgcolor.OKLabFromNRGBA(a)
	l2, a2, b2 := vgcolor.OKLabFromNRGBA(b)
	return math.Sqrt((l1-l2)*(l1-l2) + (a1-a2)*(a1-a2) + (b1-b2)*(b1-b2))
}

// neutralPart is a surface, a border or the body ink: all of them resolutions of
// one neutral step.
func neutralPart(name string, n tokens.Ramp, col stdcolor.NRGBA) Part {
	part := BasePart(NeutralName, n, col, PickJustOff, PickOffRamp)
	part.Name = name
	return part
}

// inkPart names what the derivation put over the base and kept: one of the
// two ends of the tonal axis, named for the base it was measured over, or —
// where the base is a dark scheme's — the role's own deepest rung. The axis
// ends are on no ramp, so an ink that is one of them claims no rung and the
// grid marks nothing for it.
func inkPart(role string, r tokens.Ramp, ink stdcolor.NRGBA) Part {
	part := Part{Name: "On" + role, Role: role}
	switch ink {
	case tokens.White:
		part.Rule = PickWhite + fmt.Sprintf(PickMeasuredOver, role)
		return part
	case tokens.Black:
		part.Rule = PickBlack + fmt.Sprintf(PickMeasuredOver, role)
		return part
	}
	if n := StepIn(r, ink); n != 0 {
		part.Rule, part.Step = fmt.Sprintf("%s %d%s", role, n, PickMeasured), n
		return part
	}
	part.Rule = PickMeasuredOn
	return part
}

// inversePart is one member of the inverse pair. It claims no rung: the colour
// is a step of the counterpart scheme's neutral ramp, and that ramp is not one
// of the eight on screen.
func inversePart(name string, col, counterpart stdcolor.NRGBA, role string, dark bool) Part {
	return Part{Name: name, Rule: inverseRule(col, counterpart, role, dark)}
}

// inverseRule names the counterpart role an inverse colour is, in the words of
// the side it came from. The claim is checked rather than asserted: where the
// colour is that side's role byte for byte it is named as it, and where it is
// not the rule falls back to naming the side alone.
func inverseRule(col, counterpart stdcolor.NRGBA, role string, dark bool) string {
	if col != counterpart {
		return fmt.Sprintf(PickOtherSide, role)
	}
	if dark {
		return fmt.Sprintf(PickOtherLight, role)
	}
	return fmt.Sprintf(PickOtherDark, role)
}

// Rows is the story as rows of a page's column: the ramps under their own
// heading, and the picks under theirs.
func Rows(p Chrome, c, other tokens.ColorTokens, ty Type, dark bool) []layout.Widget {
	groups := Groups(c, other, dark)
	return []layout.Widget{
		Heading(p, c, ty, RampsLabel, RampsHint),
		Body(c, rampGrid(p, c, ty, Claims(groups))),
		Heading(p, c, ty, PicksLabel, PicksHint),
		Body(c, pickBoard(p, c, ty, groups)),
	}
}

// Heading labels one section, with what it is at the leading edge and how to
// read it at the trailing one — the two-part label a page of labelled sections
// carries, on the ground that page's headings stand on.
func Heading(p Chrome, c tokens.ColorTokens, ty Type, title, hint string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(SectionHeadH))
		paint.FillShape(gtx.Ops, p.Surface, clip.Rect{Max: size}.Op())
		line := gtx.Dp(hairline)
		paint.FillShape(gtx.Ops, c.Divider,
			clip.Rect(image.Rect(0, size.Y-line, size.X, size.Y)).Op())
		pad := gtx.Dp(inventory.SectionPadX)
		box := image.Rect(pad, 0, max(pad, size.X-pad), size.Y)
		textdraw.FillText(gtx, ty.Shaper, ty.Label, box, 0, 0.5, p.Text, title)
		// The caption takes what the title leaves rather than the whole bar, so
		// a narrow window truncates it instead of running it into the title.
		//
		// Drawn in the ink the title is in, not a rung quieter: the caption's
		// leading clause is the only legend the mark on the grid below has —
		// nothing else on screen says what a dot on a swatch means — so it (and
		// the grid's own step numbers) must read at full strength. The neutral
		// ramp's quiet step reads at two-thirds its neighbours' contrast
		// against a dark ground and a third against a light one, which would
		// make the caption quiet in one scheme and faint in the other.
		if lead := box.Min.X + natural(gtx, ty.Shaper, ty.Label, title) + gtx.Dp(captionGap); lead < box.Max.X {
			if fit := FitHint(gtx, ty, hint, box.Max.X-lead); fit != "" {
				textdraw.FillText(gtx, ty.Shaper, ty.Small,
					image.Rect(lead, 0, box.Max.X, size.Y), 1, 0.5, p.Text, fit)
			}
		}
		return layout.Dimensions{Size: size}
	}
}

// FitHint is a section's caption cut to the room it has: whole clauses
// dropped from the tail (never a mid-word cut), and nothing marks the cut —
// clauses are written tail-droppable on purpose, in an order that keeps the
// load-bearing ones first. With room for not even the leading clause, the
// caption is dropped whole rather than shown as a fragment.
func FitHint(gtx layout.Context, ty Type, hint string, room int) string {
	if natural(gtx, ty.Shaper, ty.Small, hint) <= room {
		return hint
	}
	clauses := strings.Split(hint, HintSep)
	heads := make([]string, 0, len(clauses))
	for n := len(clauses) - 1; n > 0; n-- {
		heads = append(heads, strings.Join(clauses[:n], HintSep))
	}
	return longestHead(gtx, ty.Shaper, ty.Small, heads, "", room)
}

// FitLine is one line of the picks board cut to the room its column has,
// never at a mid-word boundary. It tries a clause cut first (at the comma
// inside a rule, the slash between a cell's two names, or [HintSep]), which
// leaves a shorter true sentence rather than an interrupted one — e.g. "Success
// 300, held at the container chroma" cuts to "Success 300". Failing that it
// falls back to a word boundary with a trailing ellipsis, since a mid-sentence
// cut must say that it stopped. With room for not even the first word, the
// line is handed to the shaper whole and wears whatever truncation it gives:
// a dropped line would leave a cell with a colour and no name at all.
func FitLine(gtx layout.Context, shaper *text.Shaper, style textdraw.TextStyle, line string, room int) string {
	if room <= 0 || natural(gtx, shaper, style, line) <= room {
		return line
	}
	if cut := longestHead(gtx, shaper, style, LineHeads(line, true), "", room); cut != "" {
		return cut
	}
	if cut := longestHead(gtx, shaper, style, LineHeads(line, false), ellipsis, room); cut != "" {
		return cut
	}
	return line
}

// LineHeads is every head this line can be cut down to, longest first: the
// head at each of its clause boundaries when clauses is set, and the head at
// each of its word boundaries when it is not. A boundary is a space; a clause
// boundary is a space preceded by the line's own punctuation, which is
// trimmed off the head so it never ends in a dangling separator.
func LineHeads(line string, clauses bool) []string {
	var heads []string
	for i := len(line) - 1; i > 0; i-- {
		if line[i] != ' ' {
			continue
		}
		if clauses && !strings.HasSuffix(line[:i], ",") &&
			!strings.HasSuffix(line[:i], " ·") && !strings.HasSuffix(line[:i], " /") {
			continue
		}
		if head := strings.TrimRight(line[:i], " ,·/"); head != "" {
			heads = append(heads, head)
		}
	}
	return heads
}

// longestHead is the first of these heads that fits the room with the tail on
// the end of it, and "" when none of them does. The heads arrive longest first,
// so the first that fits is the most of the line that could be kept.
func longestHead(gtx layout.Context, shaper *text.Shaper, style textdraw.TextStyle, heads []string, tail string, room int) string {
	for _, head := range heads {
		if natural(gtx, shaper, style, head+tail) <= room {
			return head + tail
		}
	}
	return ""
}

// Body lays one section's content out on the page's own ground, inside the
// margin every other body in that column keeps.
//
// The content is drawn before the ground is painted and replayed over it: the
// height is the content's own (a grid and a board differ, and neither is
// fixed), so the ground cannot be filled until the content is measured.
func Body(c tokens.ColorTokens, body func(gtx layout.Context, width int) int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		padX, padY := gtx.Dp(inventory.SectionPadX), gtx.Dp(inventory.SectionPadY)
		width := max(0, gtx.Constraints.Max.X-2*padX)
		macro := op.Record(gtx.Ops)
		h := 0
		at(gtx, image.Pt(padX, padY), func(gtx layout.Context) { h = body(gtx, width) })
		content := macro.Stop()
		size := image.Pt(gtx.Constraints.Max.X, h+2*padY)
		paint.FillShape(gtx.Ops, c.Background, clip.Rect{Max: size}.Op())
		content.Add(gtx.Ops)
		return layout.Dimensions{Size: size}
	}
}

// rampGrid draws the eight ramps as a table: the step numbers standing over
// the columns once, and under them a row per role — its name at the leading
// edge and its nine steps beside it, each in its own cell. A rung one of the
// picks took carries a mark, joining the picks' rules ("Error 700") to the
// grid's row and column.
func rampGrid(p Chrome, c tokens.ColorTokens, ty Type, claims map[Claim]bool) func(gtx layout.Context, width int) int {
	rows := RampRows(c)
	return func(gtx layout.Context, width int) int {
		head, rowH := gtx.Dp(RampHeadH), gtx.Dp(RampRowH)
		total := head + len(rows)*rowH
		labelW := min(gtx.Dp(RampLabelW), width)
		// The chips are reserved out of the width before the cells are measured,
		// and given up only when reserving them would leave the nine steps under
		// a point each. A grid too narrow to hold both is a grid too narrow to
		// read, and the steps are the thing the chips are read against.
		pinW, pinGap := gtx.Dp(RampPinW), gtx.Dp(RampPinGap)
		if width-labelW-pinGap-pinW < RampSteps {
			pinW, pinGap = 0, 0
		}
		cellW := max(0, width-labelW-pinGap-pinW) / RampSteps
		if cellW <= 0 {
			return total
		}
		// Ranged against the trailing edge, matching the heading bar's caption
		// and the picks board below, rather than left wherever nine whole-point
		// cells happen to run out. The reserved width above guarantees the gap
		// before the chip is never less than [RampPinGap] and never more than
		// eight points past it.
		pinX := width - pinW
		// The numbers are set in the ink the names are, not a rung quieter. They
		// are the table's only legend — every cell under them is a colour with
		// no other way of saying which step it is — and a legend drawn fainter
		// than the thing it explains is a legend somebody has to go looking for.
		for n := range RampSteps {
			box := image.Rect(labelW+n*cellW, 0, labelW+(n+1)*cellW, head)
			textdraw.FillText(gtx, ty.Shaper, ty.Small, box, 0.5, 0.5, p.Text,
				strconv.Itoa((n+1)*100))
		}
		// A word rather than a tenth number, because the chips under it are not
		// a step and the header row is where a reader finds out.
		if pinW > 0 {
			textdraw.FillText(gtx, ty.Shaper, ty.Small,
				image.Rect(pinX, 0, pinX+pinW, head), 0.5, 0.5, p.Text, RampPinHead)
		}
		gutter := gtx.Dp(RampGutter)
		line := gtx.Dp(hairline)
		for i, r := range rows {
			y := head + i*rowH
			// Ranged against the grid rather than off the margin: set flush left
			// the eight names end wherever their own length puts them, and half
			// of them float a word away from the row they belong to.
			textdraw.FillText(gtx, ty.Shaper, ty.Small,
				image.Rect(0, y, max(0, labelW-gutter), y+rowH), 1, 0.5, p.Text, r.Name)
			for n := range RampSteps {
				// A point off the row top and bottom, so two ramps a rung apart
				// are two rows rather than one block of colour.
				cell := image.Rect(labelW+n*cellW, y+1, labelW+(n+1)*cellW, y+rowH-1)
				// A fill with the colour inset into it, not a stroke over it: a
				// one-point centred stroke antialiases to half strength, which
				// draws no visible boundary between a step and a page it is
				// within a shade of (the case at step 100 of every ramp — the
				// ground, near enough). [EdgeIn] is the one frame colour used
				// for the whole section.
				step := r.Ramp.Step((n + 1) * 100)
				paint.FillShape(gtx.Ops, EdgeIn(c), clip.Rect(cell).Op())
				if in := cell.Inset(line); !in.Empty() {
					paint.FillShape(gtx.Ops, step, clip.Rect(in).Op())
				}
				if claims[Claim{r.Name, (n + 1) * 100}] {
					markRung(gtx, cell, step)
				}
			}
			if pinW > 0 {
				slot := image.Rect(pinX, y+gtx.Dp(RampPinInset), pinX+pinW, y+rowH-gtx.Dp(RampPinInset))
				if r.Pin.A != 0 {
					markPin(gtx, c, slot, r.Pin)
					// A pin that claims no rung has no cell to dot, so the dot
					// lands on the chip instead.
					if PinRung(r.Ramp, r.Pin) == 0 {
						markRung(gtx, slot, r.Pin)
					}
				} else {
					textdraw.FillText(gtx, ty.Shaper, ty.Small, slot, 0.5, 0.5, p.Muted, RampPinNone)
				}
			}
		}
		return total
	}
}

// markPin draws the base a role pinned, at the end of that role's row: a
// rounded chip with the row cells' own frame, so a pale pinned colour still
// shows a boundary against a pale ground.
func markPin(gtx layout.Context, c tokens.ColorTokens, box image.Rectangle, pin stdcolor.NRGBA) {
	if box.Empty() {
		return
	}
	radius := gtx.Dp(innerR) / 2
	fillRRect(gtx, box, radius, pin)
	strokeRRect(gtx, box, radius, gtx.Dp(hairline), EdgeIn(c))
}

// markRung puts the dot on the ground a pick lives on: the cell of a rung it
// took, or — for a pin that claims no rung — the chip at the end of its row.
// Its ink is measured over that ground rather than taken from the page, since
// the mark can land on any of seventy-two rungs or on an arbitrary pinned
// chip and a page-chosen ink would be invisible on some of them.
func markRung(gtx layout.Context, cell image.Rectangle, step stdcolor.NRGBA) {
	d := min(gtx.Dp(RampMark), min(cell.Dx(), cell.Dy())/2)
	if d <= 0 {
		return
	}
	mid := image.Pt((cell.Min.X+cell.Max.X)/2, (cell.Min.Y+cell.Max.Y)/2)
	dot := image.Rect(mid.X-d/2, mid.Y-d/2, mid.X-d/2+d, mid.Y-d/2+d)
	fillRRect(gtx, dot, d/2, MarkInkOn(step))
}

// MarkInkOn is the ink a mark takes over one step: whichever end of the tonal
// axis reads better on it, measured.
//
// Measured, and not decided by asking whether the step is dark. Half-way up the
// luminance scale is the answer to "which side of a scheme is this", and it is
// the wrong question here: a step at a luminance of a third is called dark by
// that test and carries white at under three to one, while black on the same
// step reads at nearly eight. That band is where the mid rungs of a saturated
// hue live — a light red, a mid amber — and this section puts marks on them the
// moment a status role's container names its mark. So the two candidates are
// tried and the better one kept, which is what the derivation itself does when
// it picks an on-colour, and the mark is doing the same job on the same ground.
func MarkInkOn(step stdcolor.NRGBA) stdcolor.NRGBA {
	if vgcolor.ContrastRatio(tokens.White, step) > vgcolor.ContrastRatio(tokens.Black, step) {
		return tokens.White
	}
	return tokens.Black
}

// EdgeIn is the frame every swatch of this section wears: the inverse of the
// page, which is the theme's own [tokens.ColorTokens.InverseSurface] — the
// counterpart scheme's surface, near-black under a light page and near-white
// under a dark one.
//
// One colour for the whole section, and not the better of two candidates
// measured over each fill. Measured per swatch, the frame took the deep
// candidate over the pale half of a ramp and the pale one over the deep half,
// so every row turned its edge over somewhere along its own length — at a
// different column per row, since the eight rows are eight hues carrying their
// lightness differently — and a table whose boundaries change polarity in the
// middle of a scale reads as two tables butted together rather than one ramp
// of nine steps. The flip was louder than the edges it was buying.
//
// So one voice, and the one that is strongest where an edge is actually needed.
// An edge is needed where a fill comes near the tone of the page it stands on:
// step 100 of every ramp is the ground, near enough, and the page, the surface
// and the divider are three of the colours the board below has to show. That is
// exactly where the inverse of the page is furthest from the fill. Walking
// toward the ink end the edge fades, and it may — a fill that far from the page
// is bounded by its own colour, which is the same boundary read from the other
// side. The page and its inverse are 14.72:1 apart in a light scheme and 14.49
// in a dark one, and for a fill between them the two readings multiply out to
// that: the edge cannot go soft without the fill having already taken the job
// over. Past the inverse — the deepest rungs of a light scheme, the palest of a
// dark one — the fill's own reading is larger still. The least-bounded swatch
// in the section measures 3.98:1 by the better of the two readings, which is
// the crossover and is above the floor a graphic owes its ground.
//
// The range, measured over every fill this section draws — seventy-two rungs,
// the pinned bases and the board's own cells, over four seeds. A light scheme's
// edge runs from 1.00:1 to 15.91:1, a dark scheme's from 1.00:1 to 17.14:1,
// with the page's own swatch — the case the edge exists for — at 14.72:1 and
// 14.49:1. The soft end is the InverseSurface cell of the picks board, which is
// this exact colour and so is framed in itself at 1.00:1; behind it come the
// 900 rungs of every ramp, at 1.16:1 in a light scheme and 1.05:1 in a dark
// one, which are the fills that read against the page at 17.10:1 and 15.22:1.
// Getting on for half the fills in the section carry an edge under 3:1 and that
// is the design: the soft end is named here and left alone rather than
// engineered away, because engineering it away is what put the polarity flip
// in the grid.
func EdgeIn(c tokens.ColorTokens) stdcolor.NRGBA { return c.InverseSurface }

// pickBoard draws every colour the theme names, in families, across as many
// columns as the window is wide enough for.
//
// A board rather than a list: eleven cells down one column is a column taller
// than the window, and the families are short enough that side by side they can
// all be read at once — which is the comparison worth having, since the question
// a reader brings here is usually about two roles rather than one.
//
// How many columns is the window's answer and not a number the board keeps: a
// third column is worth having only where three of them still hold what the
// cells have to say, and where they do not the board spreads over two.
func pickBoard(p Chrome, c tokens.ColorTokens, ty Type, groups []Group) func(gtx layout.Context, width int) int {
	return func(gtx layout.Context, width int) int {
		gap := gtx.Dp(PickColGap)
		cols := Pack(groups, Columns(width, gap, Narrowest(gtx, ty, groups)))
		colW := (width - (len(cols)-1)*gap) / len(cols)
		if colW <= 0 {
			return 0
		}
		tallest := 0
		for i, col := range cols {
			x, y := i*(colW+gap), 0
			for gi, g := range col {
				if gi > 0 {
					y += gtx.Dp(PickGroupGap)
				}
				y += drawFamily(gtx, p, ty, g.Name, x, y, colW)
				for _, cell := range g.Cells {
					h := gtx.Dp(cell.Height())
					drawCell(gtx, p, c, ty, cell, image.Rect(x, y, x+colW, y+h))
					y += h
				}
			}
			tallest = max(tallest, y)
		}
		return tallest
	}
}

// drawFamily draws the name over one family of cells and the line under it, and
// answers how much of the column the pair took.
//
// The line is what makes the name out-rank the cells below it. The names in
// those cells are set at the size a name is read at, and a family heading a
// point or two larger than them is not a level of its own — it is the same level
// slightly emphasised, which is worse than no level at all. A rule across the
// column is unmistakably a break, costs one point of height, and binds the name
// to what is under it rather than leaving it floating between two families.
func drawFamily(gtx layout.Context, p Chrome, ty Type, name string, x, y, w int) int {
	head := gtx.Dp(PickHeadH)
	textdraw.FillText(gtx, ty.Shaper, ty.Head, image.Rect(x, y, x+w, y+head), 0, 0.5, p.Text,
		FitLine(gtx, ty.Shaper, ty.Head, name, w))
	line := gtx.Dp(hairline)
	paint.FillShape(gtx.Ops, p.Divider, clip.Rect(image.Rect(x, y+head, x+w, y+head+line)).Op())
	return head + line + gtx.Dp(PickHeadGap)
}

// drawCell draws one cell in the slot it was given: the colour, with the ink
// written on it where there is one, and beside them the names over their rules.
//
// The rules are a rung quieter than the names. They are two different kinds of
// thing — one is what the theme calls this colour, the other is where it came
// from — and a reader scanning for a token name has to be able to skip the
// halves that are not names.
func drawCell(gtx layout.Context, p Chrome, c tokens.ColorTokens, ty Type, cell Cell, r image.Rectangle) {
	if r.Dx() <= 0 {
		return
	}
	sw, sh := min(gtx.Dp(PickSwatchW), r.Dx()), min(gtx.Dp(PickSwatchH), r.Dy())
	top := r.Min.Y + (r.Dy()-sh)/2
	box := image.Rect(r.Min.X, top, r.Min.X+sw, top+sh)
	radius := gtx.Dp(innerR) / 2
	fillRRect(gtx, box, radius, cell.Fill)
	// The one frame the grid above uses — see [EdgeIn] — and here for the reason
	// it is there: a Background swatch on the page it is the background of has
	// no boundary of its own, and without one it reads as a swatch that failed
	// to draw. This board is where that case is certain rather than possible,
	// since the page, the surface and the divider are three of the colours it
	// has to show, and it is where the edge is at its strongest.
	strokeRRect(gtx, box, radius, gtx.Dp(hairline), EdgeIn(c))
	switch {
	case cell.Mark:
		// In the colour the derivation chose, over the ground it chose it
		// against: this cell is the specimen of that pairing, and a mark drawn
		// in anything else would be a claim nobody could check by looking.
		markGlyph(gtx, box, cell.On)
	case cell.Paired():
		textdraw.FillText(gtx, ty.Shaper, ty.Label, box, 0.5, 0.5, cell.On, PickGlyph)
	}
	lines := box.Max.X + gtx.Dp(PickGap)
	if lines >= r.Max.X {
		return
	}
	// What the words have to fit in, which is what is left of the slot once the
	// swatch and the air beside it have taken theirs. Every line drawn below is
	// cut to it at its own boundaries rather than handed over long — see
	// [FitLine] — because the shaper's answer to a line that does not fit is to
	// break a word, and a broken token name is a name a reader cannot look up.
	room := r.Max.X - lines
	// The lines stand in a block shorter than the slot, which is where the air
	// between one cell and the next comes from. Set at the slot's full height
	// they came out on an even pitch from the top of the column to the bottom,
	// and an even pitch is what a list of thirty lines looks like rather than
	// eleven cells: nothing in the spacing said which rule belonged to which
	// name.
	title, rule := gtx.Dp(PickTitleH), gtx.Dp(PickRuleH)
	rules := 1
	if cell.Paired() {
		rules = 2
	}
	block := min(title+rules*rule, r.Dy())
	y := r.Min.Y + (r.Dy()-block)/2
	textdraw.FillText(gtx, ty.Shaper, ty.Body,
		image.Rect(lines, y, r.Max.X, y+title), 0, 0.5, p.Text,
		FitLine(gtx, ty.Shaper, ty.Body, cell.Title(), room))
	y += title
	textdraw.FillText(gtx, ty.Shaper, ty.Small,
		image.Rect(lines, y, r.Max.X, y+rule), 0, 0.5, p.Muted,
		FitLine(gtx, ty.Shaper, ty.Small, cell.Base.Rule, room))
	if cell.Paired() {
		y += rule
		textdraw.FillText(gtx, ty.Shaper, ty.Small,
			image.Rect(lines, y, r.Max.X, y+rule), 0, 0.5, p.Muted,
			FitLine(gtx, ty.Shaper, ty.Small, cell.Ink.Rule, room))
	}
}

// markGlyph draws a status role's mark on its own container: a square, which is
// the plainest thing that is a graphic and not a letter.
//
// A square and not a disc, though a disc is plainer still, because the grid
// above already spends a disc on something else — a dot there says a pick came
// off this rung — and one shape carrying two unrelated meanings in one section
// is a legend a reader has to hold two entries for. The square is nearly the
// same weight of ink and cannot be confused with the marker.
//
// The size is a share of the swatch rather than a number of its own. A mark
// measured against the non-text floor is legible at the size a graphic is drawn
// at and not at the size text is, and a mark small enough to pass for a full
// stop would understate a contrast the derivation actually achieved.
func markGlyph(gtx layout.Context, box image.Rectangle, mark stdcolor.NRGBA) {
	d := min(box.Dx(), box.Dy()) / 2
	if d <= 0 {
		return
	}
	mid := image.Pt((box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2)
	fillRRect(gtx, image.Rect(mid.X-d/2, mid.Y-d/2, mid.X-d/2+d, mid.Y-d/2+d), gtx.Dp(hairline), mark)
}

// Narrowest is the narrowest a column is worth having: a swatch, the air beside
// it, and the longest name the board is about to draw, whole.
//
// Measured off those names rather than written down as a number, because the
// number is a fact about the token vocabulary and not about this layout. It was
// written down, at two hundred and fifty points, and the vocabulary grew past
// it: "InverseSurface / OnInverseSurface" wants two hundred and seventeen and
// the widest family name a hundred and eighteen, so a window at nine hundred
// took the three columns the constant said it could afford and cut two of them
// mid-name. A board is worth spreading over another column only when the extra
// column can still say what a cell says, so the names decide, and they decide
// again whenever a role is renamed or the theme names a new one.
//
// The rules under the names are not in this measurement. A rule is a sentence
// about a name and it is longer than the name by design — sizing columns to the
// longest of them would cost the board a column at widths where every name and
// most rules fit — so the rules are what [FitLine] cuts at their own clauses,
// and the names are what never has to be cut at all.
func Narrowest(gtx layout.Context, ty Type, groups []Group) int {
	lead, narrowest := gtx.Dp(PickSwatchW)+gtx.Dp(PickGap), 0
	for _, g := range groups {
		// A family's name stands across the column, over the swatches rather
		// than beside them, so it asks for the column and not for the text.
		narrowest = max(narrowest, natural(gtx, ty.Shaper, ty.Head, g.Name))
		for _, cell := range g.Cells {
			narrowest = max(narrowest, lead+natural(gtx, ty.Shaper, ty.Body, cell.Title()))
		}
	}
	return narrowest
}

// Columns is how many columns of cells fit in width px, gap px apart, at no
// less than narrowest px each — at least one, however narrow the window is, and
// never more than the board is worth spreading over.
func Columns(width, gap, narrowest int) int {
	if width <= 0 || narrowest <= 0 {
		return 1
	}
	return max(1, min(PickMaxCols, (width+gap)/(narrowest+gap)))
}

// Load is how tall one family stands, in dp: its name, the line under it, the
// air below that, and a slot per cell. It is what the columns are balanced by,
// and it is arithmetic on the constants rather than a measurement, so the deal
// is the same deal before a frame is drawn.
func Load(g Group) int {
	h := int(PickHeadH) + int(PickHeadGap)
	for _, c := range g.Cells {
		h += int(c.Height())
	}
	return h
}

// Pack deals the families into n columns so that the tallest column is as short
// as it can be, each family whole and none of them out of order.
//
// Whole, because a family cut across a column boundary is two families as far as
// anybody reading is concerned. In order, because the board is read down one
// column and then down the next, and a deal free to put the fourth family in the
// first column would give the same board two different reading orders at two
// window widths — a reader dragging the window watches a family change
// neighbours.
//
// So a deal is not an assignment of families to columns, it is a run of
// boundaries in the reading order, and the search is over where the boundaries
// fall. Which is the difference between a handful of arrangements and a number
// that grows by a factor of three every time the board gains a family: this
// section stood at four families and grew to six the moment the containers and
// the ends of the axis joined it, and enumerating assignments would have gone
// from eighty-one arrangements a frame to seven hundred and twenty-nine, to
// find what twenty-eight runs of boundaries answer.
func Pack(groups []Group, n int) [][]Group {
	if n < 1 {
		n = 1
	}
	cols, cuts := make([][]Group, n), bestCuts(groups, n)
	at := 0
	for i, g := range groups {
		for at < len(cuts) && i >= cuts[at] {
			at++
		}
		cols[at] = append(cols[at], g)
	}
	return cols
}

// bestCuts is where the column boundaries fall in the evenest in-order deal:
// n-1 indices into groups, never going backwards, cut[j] naming the first
// family of column j+1. A boundary at the end of the run leaves its column
// empty, which is what a board with fewer families than columns comes to.
//
// Ties go to the first arrangement found, and the walk starts each boundary as
// far along the run as it can go, so the first arrangement found — and the one
// a tie keeps — is the one that fills the leftmost column first.
func bestCuts(groups []Group, n int) []int {
	cuts, best, tallest := make([]int, n-1), make([]int, n-1), -1
	var walk func(j, from int)
	walk = func(j, from int) {
		if j == len(cuts) {
			if got := cutsTallest(groups, cuts); tallest < 0 || got < tallest {
				tallest = got
				copy(best, cuts)
			}
			return
		}
		for cut := len(groups); cut >= from; cut-- {
			cuts[j] = cut
			walk(j+1, cut)
		}
	}
	walk(0, 0)
	return best
}

// cutsTallest is the height of the tallest column one run of boundaries deals.
func cutsTallest(groups []Group, cuts []int) int {
	tallest, load, at := 0, 0, 0
	for i, g := range groups {
		for at < len(cuts) && i >= cuts[at] {
			at, load = at+1, 0
		}
		load += Load(g)
		tallest = max(tallest, load)
	}
	return tallest
}

// typeSection is the inventory section a page borrows to close the story: the
// whole type ladder, every role a surface reads in.
const typeSection = "foundations-type"

// sectionTitleSep is the seam an inventory section's title is written with:
// what the section is, then how to read it. The story's own bands are built
// from exactly that pair — a label at the leading edge and a caption at the
// trailing one — so a borrowed title splits into a band with nothing reworded.
const sectionTitleSep = " — "

// TypeLadderRows is the inventory's type ladder as two rows in the story's own
// bands: the heading band the story's own sections wear, over the inventory
// section's own body.
//
// The ladder rides with the story rather than standing on a page of its own
// because a theme is a palette and a typeface: the type roles are generated
// from the same theme the ramps are, so whatever answers "what is this theme"
// has to answer both halves of it. It wears the band its neighbours wear —
// a column with two sections banded one way and a third banded another is
// inconsistent with itself, in a column whose whole subject is that a theme is
// coherent — and the inventory's own words are kept, split at the em dash its
// titles are already written with, so nothing is reworded here and a title
// reworded upstream arrives reworded.
//
// A section whose title carries no separator lands with the whole title as the
// label and no caption, which is what the band does with an empty hint anyway.
func TypeLadderRows(inv *inventory.Inventory, p Chrome, c tokens.ColorTokens, ty Type) []layout.Widget {
	for _, s := range inv.Foundations(c) {
		if s.Name != typeSection {
			continue
		}
		label, hint, _ := strings.Cut(s.Title, sectionTitleSep)
		return []layout.Widget{
			Heading(p, c, ty, label, hint),
			Body(c, ladderBody(s)),
		}
	}
	return nil
}

// ladderBody adapts an inventory section's body to the story body's shape: the
// story measures its content and reports the height, while a section body is
// laid out in a slot of the height the section states. So the slot is stated
// here — bounded, because the type ladder measures nothing of its own and an
// unbounded one would take the column with it — and handed back as the height
// the band wraps.
func ladderBody(s inventory.Section) func(gtx layout.Context, width int) int {
	return func(gtx layout.Context, width int) int {
		h := gtx.Dp(s.Height)
		gtx.Constraints = layout.Constraints{Max: image.Pt(width, h)}
		s.Body(gtx)
		return h
	}
}

// natural is how wide a string wants to be, unconstrained by the room it is
// about to be given.
func natural(gtx layout.Context, shaper *text.Shaper, style textdraw.TextStyle, str string) int {
	gtx.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	return textdraw.MeasureText(gtx, shaper, style, str).X
}

// fillRRect paints a rounded rectangle.
func fillRRect(gtx layout.Context, r image.Rectangle, radius int, c stdcolor.NRGBA) {
	defer clip.UniformRRect(r, radius).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// strokeRRect outlines a rounded rectangle, inset by half the stroke width so
// the whole line lands inside the rectangle rather than half outside it.
func strokeRRect(gtx layout.Context, r image.Rectangle, radius, width int, c stdcolor.NRGBA) {
	if width <= 0 {
		return
	}
	half := float32(width) / 2
	inner := image.Rect(r.Min.X+width/2, r.Min.Y+width/2, r.Max.X-width/2, r.Max.Y-width/2)
	path := clip.UniformRRect(inner, max(0, radius-width/2)).Path(gtx.Ops)
	defer clip.Stroke{Path: path, Width: half * 2}.Op().Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// at offsets the operations w records to origin, leaving the caller's
// coordinate system untouched.
func at(gtx layout.Context, origin image.Point, w func(gtx layout.Context)) {
	defer op.Offset(origin).Push(gtx.Ops).Pop()
	w(gtx)
}
