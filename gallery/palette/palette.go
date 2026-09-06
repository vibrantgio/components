// Package palette draws the palette story: the ramps a theme has to pick
// from, and the colours it picked, each beside the rule that chose it.
//
// Ramps stand above picks, so a pick naming a step ("Neutral 300") reads
// against the row it came from. Every function is a pure function of the
// [tokens.ColorTokens] handed in, plus the furniture colours and type roles a
// caller states in [Chrome] and [Type]; nothing reads a default palette, so
// the same code draws either scheme and a test can capture it without a
// window.
//
// Rules are read off the colours themselves — a pin compared against its own
// ramp, a foreground against the two ends of the tonal axis and its role's
// deepest step — rather than written down statically, so a derivation change
// updates what this package says in the same build.
//
// A base and its foreground are one cell because they are one decision: the
// derivation pins the base, then measures both ends of the tonal axis over
// that exact colour and keeps the better one, so the foreground cannot be
// understood apart from the fill it was measured against. Surface and Divider
// stand alone because the theme names no foreground for either.
//
// Each ramp row ends with the base that role pinned, drawn as a chip because
// two roles (a light scheme's Primary, Secondary, Tertiary) pin a colour that
// is off its own ramp by construction. Where the pin is a step exactly, the
// chip and the marked cell are the same colour; where the pin is
// indistinguishable from no step at all, the dot moves onto the chip itself so
// every row still answers where its pinned colour lives. Neutral's chip is
// absent because Neutral pins no solid fill.
//
// Status containers and the two axis ends (white, black) get cells of their
// own because neither is a colour any ramp cell holds: a container is its
// role's hue at a step's tone with the chroma pulled to the container dial,
// and white/black belong to no ramp.
//
// The reserved highlighter stands in a family of its own for the same reason
// carried one step further: its hue is reserved outside the role table, so no
// row of the grid runs at it, no step of one is its to claim, and no seed
// rotates it.
//
// Deliberately out of scope: interaction-state colours (hover, pressed,
// selected, dragged — a component's own transform of a colour it was given),
// disabled colours (an alpha fraction, not a palette member), and the focus
// ring (Neutral 500, already a cell here). Everything this package draws is a
// colour some component is painted with at rest.
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
	// Surface is the fill a section's heading band carries.
	Surface stdcolor.NRGBA
	// Divider is the rule under a family's name on the picks board.
	Divider stdcolor.NRGBA
	// Text is the reading foreground: section labels, their captions, the ramp
	// names and the step numbers over the grid.
	Text stdcolor.NRGBA
	// Muted is the least pronounced of the four: a pick's rule under its name,
	// and the mark standing where a role pins nothing.
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
	// RampMark is the dot on the step a pick sits at. It is a dot and not a
	// ring, a label or a heavier frame: a fifth of the cells carry one, and
	// anything with an edge of its own at that count turns a table of colour
	// into a table of marks.
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
	// of these swatches carry a foreground, and a foreground shown smaller
	// than the claim next to it is a claim nobody can check — and tall enough
	// that the letters have air above and below rather than reaching the edge,
	// which on a chip this small is the difference between a specimen and a
	// stamp.
	PickSwatchW unit.Dp = 44
	PickSwatchH unit.Dp = 26
	// PickPairH is a base and its foreground, which carry three lines: the two
	// names, and a rule for each. PickCellH is a colour that has no
	// foreground, which carries two. The difference between either and the
	// lines inside it is the air one cell keeps from the next.
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
// Step 100 is the palest step in a light theme and the deepest in a dark one,
// and it is the same step in both (the one nearest the page), which is why a
// component asking for 100 gets a tint on either side of the scheme switch.
//
// A pin indistinguishable from no step carries its dot on the chip instead of
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

// The families the cells are read in: page and surfaces (what everything else
// stands on) and the inverse pair (also surfaces, borrowed from the other side
// of the scheme) first, then the reserved highlighter, then the accents the
// seed rotates, then the status roles it may only tint. Containers have no
// family of their own — each stands under its role, inside Status. The tonal
// axis ends come last: they are what the foregrounds above turned out to be.
//
// Reserved holds the one colour that is in no role: it stands where the CSS
// export puts it, after the inverse pair and ahead of the accents, so a reader
// meets the theme's colours in one order wherever they are listed.
const (
	PickPageGroup     = "Page and surfaces"
	PickInverseGroup  = "Inverse"
	PickReservedGroup = "Reserved"
	PickAccentGroup   = "Accents"
	PickStatusGroup   = "Status"
	PickAxisGroup     = "Axis ends"
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
	// PickPairSep joins a cell's fill name to its foreground name.
	PickPairSep = " / "
	// PickMeasured closes a foreground's rule when its name already names the
	// role (a step of that role's own ramp — a dark scheme's foregrounds).
	// PickMeasuredOver closes it when the foreground is white or black, which
	// name no role, so the base is named instead.
	PickMeasured     = ", measured over the base"
	PickMeasuredOver = ", measured over %s"
	PickWhite        = "white"
	PickBlack        = "black"
	// PickMeasuredOn is a foreground that is neither an axis end nor a step of
	// its own ramp; no derivation shipping today produces this case.
	PickMeasuredOn = "measured over the base"
	// PickSeed is the light scheme's Primary where the chosen colour fell
	// between steps: lifted onto the palette's own chroma, pinned at its own
	// depth.
	PickSeed = "the seed, lifted"
	// PickJustOff is a base pinned one unit of lightness off its own 700 step
	// (the light scheme's accents) — named by that step, since it is
	// indistinguishable from it on the grid.
	PickJustOff = "pinned just off %s %d"
	// PickSeedNear is PickSeed's rule where the lifted seed lands beside a
	// step rather than on one.
	PickSeedNear = "the seed, lifted, just off %s %d"
	// PickPinned is a base near no step at all; no derivation shipping today
	// produces this case for any role but Primary.
	PickPinned = "pinned off the ramp"
	// PickOffRamp is a neutral resolution off the neutral ramp; no derivation
	// shipping today produces this case.
	PickOffRamp = "off the neutral ramp"
	// PickContentPin is the content plane's rule where the neutral band
	// offers nothing above its own 100 stop: the pin stands one band step
	// under the tonal axis so that the first raise on it has a whole step to
	// take. Where the band climbs away from its 100 stop the pin IS that
	// stop and is named as one.
	PickContentPin = "one band step under white"
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
	// PickContainerRule is a status container: its role's own step's depth, at
	// the role's anchor hue and the containers' shared chroma. Naming the step
	// is what makes the depth findable on the grid; the hue is not that
	// step's, because a tinted fill carries its role's hue at every depth it
	// is drawn at (theme's containers.go).
	PickContainerRule = "%s %d's depth, at the container chroma"
	// PickMarkRule is the mark read on a container: a step of the role's own
	// ramp, chosen against the container rather than against a page.
	PickMarkRule = "%s %d, measured over the container"
	// PickMarkOff is a mark that is not a step of its own ramp; no derivation
	// shipping today produces this case.
	PickMarkOff = "measured over the container"
	// PickHighlightRule is the reserved highlighter: a hue no status may use,
	// deepened off its fixed step until it clears the container floor over
	// what it is marking. The separation is measured off this scheme's own
	// status fills rather than asserted; the derivation holds at least 37.60°
	// of it over the seed sweep, at one strength and at the same colour under
	// every seed, which is why the line names no seed, no role and no step
	// (theme/tokens/highlight.go).
	//
	// The floor is named against the page and not against "the surface":
	// the cell shows the field, which is the fill resolved against the
	// Background pin, and it clears the floor there and not everywhere the
	// word "surface" would cover — 1.32:1 and 1.90:1 against the page, but
	// 1.21:1 over the light scheme's deeper Surface token, where a fill is a
	// walked colour this cell does not carry.
	//
	// Two clauses and no longer, like every other rule here: the board cuts a
	// long line at its first comma, so the reservation stands ahead of it and
	// the whole line still fits the widest column the board is dealt.
	PickHighlightRule = "a reserved hue %.0f° off every status, floored at %.2f over the page"
	// The two ends of the tonal axis, on no ramp, each named for the end it is
	// and whether the scheme on screen writes any foreground in it — read off
	// that scheme's own foregrounds rather than asserted, so the answer turns
	// over with the scheme switch.
	PickAxisLight        = "the tonal axis's light end"
	PickAxisDark         = "the tonal axis's dark end"
	PickAxisForeground   = "%s, a foreground here"
	PickAxisNoForeground = "%s, no foreground here"
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
	HighlightPick        = "Highlight"
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

// Claim is one cell of the grid a pick took: the row it is on and the step it
// is.
type Claim struct {
	Role string
	Step int
}

// Claims is every step the picks below the grid took, which is what the grid
// marks. It is read off the picks rather than tracked separately, so a rule
// and its marker cannot disagree.
func Claims(groups []Group) map[Claim]bool {
	out := map[Claim]bool{}
	for _, g := range groups {
		for _, cell := range g.Cells {
			for _, part := range [2]Part{cell.Base, cell.Foreground} {
				if part.Role != "" && part.Step != 0 {
					out[Claim{part.Role, part.Step}] = true
				}
			}
		}
	}
	return out
}

// Part is one colour token in a cell: what the theme calls it, what chose it,
// and — where what chose it was a step of a ramp — which row and which step,
// so the grid above can mark the step this pick took. Step and rule must be
// resolved together, never separately, or a rule and its marker can disagree.
type Part struct {
	Name, Rule string
	Role       string // the ramp row the rule names, empty when it names none
	Step       int    // the step on that row, 0 when the colour is on none
}

// Cell is one thing the palette decided. It is a base and the foreground
// measured over it — one swatch, two names, two rules — or, where the theme
// names no foreground for a colour, that colour on its own.
//
// Mark says the second colour is a mark and not a foreground: it is drawn as a
// shape over the fill rather than as two letters, because it was chosen
// against the non-text floor and letters would claim a legibility nothing
// measured.
type Cell struct {
	Base, Foreground Part
	Fill, On         stdcolor.NRGBA
	Mark             bool
}

// Paired reports whether this cell carries a foreground as well as a fill.
func (c Cell) Paired() bool { return c.Foreground.Name != "" }

// Title is the cell's names, in the order their rules are written under them.
func (c Cell) Title() string {
	if !c.Paired() {
		return c.Base.Name
	}
	return c.Base.Name + PickPairSep + c.Foreground.Name
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
			// The page and the foreground it is read in: the one pair in the
			// theme that is two pins rather than a pin and a measurement.
			{
				Base:       backgroundPart(n, c.Background),
				Foreground: neutralPart(TextPick, n, c.Text),
				Fill:       c.Background, On: c.Text,
			},
			alone(neutralPart(SurfacePick, n, c.Surface), c.Surface),
			alone(neutralPart(DividerPick, n, c.Divider), c.Divider),
		}},
		{PickInverseGroup, []Cell{{
			Base:       inversePart(InverseSurfacePick, c.InverseSurface, other.Surface, PickSurfaceRole, dark),
			Foreground: inversePart(OnInverseSurfacePick, c.OnInverseSurface, other.Text, PickTextRole, dark),
			Fill:       c.InverseSurface, On: c.OnInverseSurface,
		}}},
		// The reserved highlighter, resolved against the page the content it
		// marks stands on. It carries no foreground: the theme names none for
		// it, and what a highlight owes the surface it marks is a findable
		// edge rather than a legibility.
		{PickReservedGroup, []Cell{
			alone(highlightPart(c), c.Highlight),
		}},
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
	// The two colours every foreground above was chosen between, shown as
	// colours rather than as letterforms, each told whether this scheme writes
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
	rule := PickAxisNoForeground
	if writtenIn(groups, col) {
		rule = PickAxisForeground
	}
	return Part{Name: name, Rule: fmt.Sprintf(rule, end)}
}

// writtenIn reports whether any cell of these families is written in col.
func writtenIn(groups []Group, col stdcolor.NRGBA) bool {
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
// one fill, one mark, and the rule each was derived by. The pair is one cell
// for the reason a base and its foreground are — the mark is chosen against
// this exact fill and cannot be understood apart from it — and the mark is
// drawn as a disc because that is the kind of thing it was measured to be.
func containerCell(role string, c tokens.ColorTokens, id tokens.Role, r tokens.Ramp) Cell {
	fill, mark := c.StatusContainer(id), c.OnStatusContainer(id)
	return Cell{
		Base:       containerPart(role, r, fill),
		Foreground: markPart(role, r, mark),
		Fill:       fill, On: mark, Mark: true,
	}
}

// containerPart is a container as a cell carries it: the step it was realized
// at, and what was done to that step to get it.
//
// The step is found by tone (lightness) rather than by colour: a container
// keeps its step's lightness, takes its hue from the ramp's pale tint depth
// and gives up chroma, so comparing bytes finds nothing and lightness is the
// one dimension that survives intact.
//
// It claims that step unconditionally, even where the container is not close
// enough to be mistaken for it — the dot on the grid always marks the step a
// pick's rule names, and marking only the close containers would leave two of
// four rows with word-for-word identical rules dotted differently.
func containerPart(role string, r tokens.Ramp, fill stdcolor.NRGBA) Part {
	step := ToneStep(r, fill)
	return Part{
		Name: role + ContainerPick,
		Rule: fmt.Sprintf(PickContainerRule, role, step),
		Role: role,
		Step: step,
	}
}

// markPart is the mark read on a container: a step of the role's own ramp,
// chosen against the container rather than against a page, so it claims that
// step and the grid marks it.
func markPart(role string, r tokens.Ramp, mark stdcolor.NRGBA) Part {
	part := Part{Name: role + MarkPick, Role: role}
	if n := StepIn(r, mark); n != 0 {
		part.Rule, part.Step = fmt.Sprintf(PickMarkRule, role, n), n
		return part
	}
	part.Rule = PickMarkOff
	return part
}

// ToneStep is the step of r a colour was realized at, read off the lightness
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

// pinnedCell is one role's cell: the base the derivation pinned and the
// foreground it measured over that exact colour, named OnPrimary etc. for the
// base it has to clear. near is what the base's rule says when it landed
// beside a step rather than on one; off is what it says when it landed beside
// none.
func pinnedCell(role string, r tokens.Ramp, base, foreground stdcolor.NRGBA, near, off string) Cell {
	return Cell{
		Base:       BasePart(role, r, base, near, off),
		Foreground: foregroundPart(role, r, foreground),
		Fill:       base, On: foreground,
	}
}

// StepIn reports which step of r the colour is, or 0 when it is not on the
// ramp at all. It compares bytes rather than measuring a distance: a pin that
// is a step is that step exactly, and a pin that is merely near one is a
// different colour.
func StepIn(r tokens.Ramp, col stdcolor.NRGBA) int {
	for i := range r {
		if r[i] == col {
			return (i + 1) * 100
		}
	}
	return 0
}

// BasePart is a pinned base as a cell carries it, in three cases: the step it
// landed on, the step it is indistinguishable from and how it came to be
// beside rather than on it (near, which takes the role and the step), and —
// where no step is near — how it was pinned and nothing else (off).
func BasePart(role string, r tokens.Ramp, col stdcolor.NRGBA, near, off string) Part {
	if n := StepIn(r, col); n != 0 {
		return Part{Name: role, Rule: fmt.Sprintf("%s %d", role, n), Role: role, Step: n}
	}
	if n := NearestStep(r, col); n != 0 {
		return Part{Name: role, Rule: fmt.Sprintf(near, role, n), Role: role, Step: n}
	}
	return Part{Name: role, Rule: off, Role: role}
}

// NearestStep is the step of r a colour is indistinguishable from, or 0 when
// it is distinguishable from all nine. "On the ramp" and "the same colour as
// the ramp" are different questions; this answers the second, since every
// light-scheme accent is pinned one unit of lightness off its own 700 step
// (three parts in 255 — invisible to eye or display).
//
// Distance is measured in OKLab, and [StepTolerance] is set by measurement:
// pins that should match sit up to 0.0158 from their step, and the closest two
// steps of any ramp this derivation builds are 0.0330 apart, so the tolerance
// must clear the first and stay under half the second or a colour could sit
// close enough to two steps at once.
func NearestStep(r tokens.Ramp, col stdcolor.NRGBA) int {
	best, at := StepTolerance, 0
	for i := range r {
		if d := OKLabDistance(r[i], col); d < best {
			best, at = d, (i+1)*100
		}
	}
	return at
}

// StepTolerance is how far from a step a colour may sit and still be that step
// as far as anybody looking is concerned. See [NearestStep] for the two
// measurements it stands between.
const StepTolerance = 0.016

// PinStep is the step a pinned base claims: the step it is exactly, else the
// one it is indistinguishable from, else 0 — the two questions [BasePart]
// resolves a base's rule by, asked in the order it asks them, so the grid and
// the rule cannot disagree about whether a pin lives on a step. The one pin
// that claims nothing is a lifted seed whose depth falls between two steps of
// the scale, and its row's chip carries the dot instead — see [rampGrid].
func PinStep(r tokens.Ramp, pin stdcolor.NRGBA) int {
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

// neutralPart is a surface, a border or the body foreground: all of them
// resolutions of one neutral step.
func neutralPart(name string, n tokens.Ramp, col stdcolor.NRGBA) Part {
	part := BasePart(NeutralName, n, col, PickJustOff, PickOffRamp)
	part.Name = name
	return part
}

// backgroundPart names where the content plane's pin came from: the neutral
// step it landed on, or — where the band has nothing above its 100 stop and
// the pin stepped down off the axis to keep headroom for its first raise —
// that rule, and no step, since the pin is on no ramp step.
func backgroundPart(n tokens.Ramp, col stdcolor.NRGBA) Part {
	if StepIn(n, col) != 0 {
		return neutralPart(BackgroundPick, n, col)
	}
	return Part{Name: BackgroundPick, Role: NeutralName, Rule: PickContentPin}
}

// foregroundPart names what the derivation put over the base and kept: one of
// the two ends of the tonal axis, named for the base it was measured over, or
// — where the base is a dark scheme's — the role's own deepest step. The axis ends are
// on no ramp, so a foreground that is one of them claims no step and the grid
// marks nothing for it.
func foregroundPart(role string, r tokens.Ramp, foreground stdcolor.NRGBA) Part {
	part := Part{Name: "On" + role, Role: role}
	switch foreground {
	case tokens.White:
		part.Rule = PickWhite + fmt.Sprintf(PickMeasuredOver, role)
		return part
	case tokens.Black:
		part.Rule = PickBlack + fmt.Sprintf(PickMeasuredOver, role)
		return part
	}
	if n := StepIn(r, foreground); n != 0 {
		part.Rule, part.Step = fmt.Sprintf("%s %d%s", role, n, PickMeasured), n
		return part
	}
	part.Rule = PickMeasuredOn
	return part
}

// inversePart is one member of the inverse pair. It claims no step: the colour
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

// highlightPart is the reserved highlighter as a cell carries it. It claims no
// step: the fill takes a neutral step's depth but sits at a hue no ramp on the
// grid runs at, so there is no row for the grid to mark it on.
func highlightPart(c tokens.ColorTokens) Part {
	return Part{
		Name: HighlightPick,
		Rule: fmt.Sprintf(PickHighlightRule, math.Floor(statusHueGap(c)), tokens.ContainerFloor),
	}
}

// statusHueGap is how far the highlighter stands from the nearest status
// colour of this palette, in OKLCh degrees — the reservation as this scheme
// realized it, read off the eight status fills the board draws below rather
// than asserted. Both the pinned fill and the container are asked, since a
// container keeps its role's hue and either could be the nearest.
//
// Reported unrounded and floored where it is printed, so the rule never
// claims a degree the palette does not hold.
func statusHueGap(c tokens.ColorTokens) float64 {
	_, _, hue := vgcolor.OKLChFromNRGBA(c.Highlight)
	gap := 180.0
	for _, col := range [8]stdcolor.NRGBA{
		c.Error, c.StatusContainer(tokens.RoleError),
		c.Success, c.StatusContainer(tokens.RoleSuccess),
		c.Warning, c.StatusContainer(tokens.RoleWarning),
		c.Info, c.StatusContainer(tokens.RoleInfo),
	} {
		_, _, h := vgcolor.OKLChFromNRGBA(col)
		gap = min(gap, hueApart(hue, h))
	}
	return gap
}

// hueApart is the shorter way round the hue circle between two angles in
// degrees.
func hueApart(a, b float64) float64 {
	d := math.Abs(a - b)
	return min(d, 360-d)
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
// carries, on the surface that page's headings stand on.
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
		// Drawn in the same foreground the title is in, not one step down: the
		// caption's leading clause is the only legend the mark on the grid
		// below has — nothing else on screen says what a dot on a swatch means
		// — so it (and the grid's own step numbers) must read at full
		// strength. The neutral ramp's step below it reads at two-thirds its
		// neighbours' contrast against a dark surface and a third against a
		// light one, which would leave the caption merely less pronounced in
		// one scheme and faint in the other.
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

// Body lays one section's content out on the page's own fill, inside the
// margin every other body in that column keeps.
//
// The content is drawn before the fill is painted and replayed over it: the
// height is the content's own (a grid and a board differ, and neither is
// fixed), so the fill cannot be painted until the content is measured.
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
// edge and its nine steps beside it, each in its own cell. A step one of the
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
		// The numbers are set in the same foreground the names are, not one
		// step down. They are the table's only legend — every cell under them
		// is a colour with no other way of saying which step it is — and a
		// legend drawn fainter than the thing it explains is a legend somebody
		// has to go looking for.
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
				// A point off the row top and bottom, so two ramps a step
				// apart are two rows rather than one block of colour.
				cell := image.Rect(labelW+n*cellW, y+1, labelW+(n+1)*cellW, y+rowH-1)
				// A fill with the colour inset into it, not a stroke over it:
				// a one-point centred stroke antialiases to half strength,
				// which draws no visible boundary between a step and a page it
				// is within a shade of (the case at step 100 of every ramp —
				// the page, near enough). [EdgeIn] is the one frame colour
				// used for the whole section.
				step := r.Ramp.Step((n + 1) * 100)
				paint.FillShape(gtx.Ops, EdgeIn(c), clip.Rect(cell).Op())
				if in := cell.Inset(line); !in.Empty() {
					paint.FillShape(gtx.Ops, step, clip.Rect(in).Op())
				}
				if claims[Claim{r.Name, (n + 1) * 100}] {
					markStep(gtx, cell, step)
				}
			}
			if pinW > 0 {
				slot := image.Rect(pinX, y+gtx.Dp(RampPinInset), pinX+pinW, y+rowH-gtx.Dp(RampPinInset))
				if r.Pin.A != 0 {
					markPin(gtx, c, slot, r.Pin)
					// A pin that claims no step has no cell to dot, so the dot
					// lands on the chip instead.
					if PinStep(r.Ramp, r.Pin) == 0 {
						markStep(gtx, slot, r.Pin)
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
// shows a boundary against a pale surface.
func markPin(gtx layout.Context, c tokens.ColorTokens, box image.Rectangle, pin stdcolor.NRGBA) {
	if box.Empty() {
		return
	}
	radius := gtx.Dp(innerR) / 2
	fillRRect(gtx, box, radius, pin)
	strokeRRect(gtx, box, radius, gtx.Dp(hairline), EdgeIn(c))
}

// markStep puts the dot on the fill a pick lives on: the cell of a step it
// took, or — for a pin that claims no step — the chip at the end of its row.
// Its colour is measured over that fill rather than taken from the page, since
// the mark can land on any of seventy-two steps or on an arbitrary pinned chip
// and a page-chosen colour would be invisible on some of them.
func markStep(gtx layout.Context, cell image.Rectangle, step stdcolor.NRGBA) {
	d := min(gtx.Dp(RampMark), min(cell.Dx(), cell.Dy())/2)
	if d <= 0 {
		return
	}
	mid := image.Pt((cell.Min.X+cell.Max.X)/2, (cell.Min.Y+cell.Max.Y)/2)
	dot := image.Rect(mid.X-d/2, mid.Y-d/2, mid.X-d/2+d, mid.Y-d/2+d)
	fillRRect(gtx, dot, d/2, MarkForegroundOn(step))
}

// MarkForegroundOn is the colour a mark takes over one step: whichever end of the
// tonal axis reads better on it, measured.
//
// Measured, and not decided by asking whether the step is dark. Half-way up
// the luminance scale is the answer to "which side of a scheme is this", and
// it is the wrong question here: a step at a luminance of a third is called
// dark by that test and carries white at under three to one, while black on
// the same step reads at nearly eight. That band is where the mid steps of a
// saturated hue live — a light red, a mid amber — and this section puts marks
// on them the moment a status role's container names its mark. So the two
// candidates are tried and the better one kept, which is what the derivation
// itself does when it picks an on-colour, and the mark is doing the same job
// on the same fill.
func MarkForegroundOn(step stdcolor.NRGBA) stdcolor.NRGBA {
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
// One colour for the whole section, not the better of two candidates measured
// per fill, so no row's edge changes polarity partway along its own length. It
// is strongest exactly where an edge is needed: step 100 of every ramp is the
// page, near enough, and that is where the inverse of the page is furthest
// from the fill. Walking toward the far end of the ramp the edge fades, but a
// fill that far from the page is bounded by its own colour instead.
//
// Measured over every fill this section draws (seventy-two steps, the pinned
// bases, the board's cells, over four seeds): a light scheme's edge runs from
// 1.00:1 to 15.91:1, a dark scheme's from 1.00:1 to 17.14:1, with the page's
// own swatch (the case the edge exists for) at 14.72:1 and 14.49:1. The soft
// end is the InverseSurface cell, framed in itself at 1.00:1; the 900 steps of
// every ramp sit at 1.16:1 (light) and 1.05:1 (dark), reading against the page
// itself at 17.10:1 and 15.22:1. The least-bounded swatch overall measures
// 3.98:1 by the better of the two axis-end readings — above the floor a
// graphic owes the surface it stands on, and the point past which a fill's own
// contrast against the page has already taken over the boundary's job.
func EdgeIn(c tokens.ColorTokens) stdcolor.NRGBA { return c.InverseSurface }

// pickBoard draws every colour the theme names, in families, across as many
// columns as the window is wide enough for. A board rather than a single
// list, so families short enough to read side by side can be compared at a
// glance. Column count is derived from the window width rather than fixed: a
// column is worth having only where it still holds what its cells have to
// say.
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

// drawFamily draws the name over one family of cells and the line under it,
// and answers how much of the column the pair took. The line under the name
// is what makes it out-rank the cells below, since a heading only a point or
// two larger than its cells' names would read as the same level, not a level
// of its own.
func drawFamily(gtx layout.Context, p Chrome, ty Type, name string, x, y, w int) int {
	head := gtx.Dp(PickHeadH)
	textdraw.FillText(gtx, ty.Shaper, ty.Head, image.Rect(x, y, x+w, y+head), 0, 0.5, p.Text,
		FitLine(gtx, ty.Shaper, ty.Head, name, w))
	line := gtx.Dp(hairline)
	paint.FillShape(gtx.Ops, p.Divider, clip.Rect(image.Rect(x, y+head, x+w, y+head+line)).Op())
	return head + line + gtx.Dp(PickHeadGap)
}

// drawCell draws one cell in the slot it was given: the colour, with the
// foreground written on it where there is one, and beside them the names over
// their rules. The rules are one step down from the names, since a reader
// scanning for a token name has to be able to skip the halves that are not
// names.
func drawCell(gtx layout.Context, p Chrome, c tokens.ColorTokens, ty Type, cell Cell, r image.Rectangle) {
	if r.Dx() <= 0 {
		return
	}
	sw, sh := min(gtx.Dp(PickSwatchW), r.Dx()), min(gtx.Dp(PickSwatchH), r.Dy())
	top := r.Min.Y + (r.Dy()-sh)/2
	box := image.Rect(r.Min.X, top, r.Min.X+sw, top+sh)
	radius := gtx.Dp(innerR) / 2
	fillRRect(gtx, box, radius, cell.Fill)
	// The one frame the grid above uses — see [EdgeIn] — needed here since the
	// Background swatch, among others, has no boundary of its own against the
	// page it is the background of.
	strokeRRect(gtx, box, radius, gtx.Dp(hairline), EdgeIn(c))
	switch {
	case cell.Mark:
		// In the colour the derivation chose, over the fill it chose it
		// against: a mark drawn in any other colour would be an unverifiable
		// claim.
		markGlyph(gtx, box, cell.On)
	case cell.Paired():
		textdraw.FillText(gtx, ty.Shaper, ty.Label, box, 0.5, 0.5, cell.On, PickGlyph)
	}
	lines := box.Max.X + gtx.Dp(PickGap)
	if lines >= r.Max.X {
		return
	}
	// Every line below is cut to this room at its own boundaries rather than
	// handed over long — see [FitLine] — since the shaper's fallback is to
	// break a word, and a broken token name cannot be looked up.
	room := r.Max.X - lines
	// The lines stand in a block shorter than the slot rather than spread to
	// the slot's full height, which is where the air between cells comes from
	// and what keeps each rule visually paired with its own name.
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
			FitLine(gtx, ty.Shaper, ty.Small, cell.Foreground.Rule, room))
	}
}

// markGlyph draws a status role's mark on its own container: a square rather
// than a disc, since the ramp grid's dot already uses a disc for an unrelated
// meaning (a pick's step) and the two must not be confused. Sized as a share
// of the swatch rather than a fixed number, since a mark is measured against
// the non-text contrast floor and must be legible at graphic size, not text
// size.
func markGlyph(gtx layout.Context, box image.Rectangle, mark stdcolor.NRGBA) {
	d := min(box.Dx(), box.Dy()) / 2
	if d <= 0 {
		return
	}
	mid := image.Pt((box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2)
	fillRRect(gtx, image.Rect(mid.X-d/2, mid.Y-d/2, mid.X-d/2+d, mid.Y-d/2+d), gtx.Dp(hairline), mark)
}

// Narrowest is the narrowest a column is worth having: a swatch, the air
// beside it, and the longest name the board is about to draw, whole.
//
// Measured off those names rather than a fixed constant, since the width is a
// fact about the current token vocabulary and must track it as roles are
// renamed or added — a fixed number silently goes stale and starts cutting
// names mid-word.
//
// Rules under the names are excluded from this measurement: a rule is longer
// than its name by design, so sizing columns to it would cost a column at
// widths where every name and most rules still fit. Rules are instead what
// [FitLine] cuts at their own clauses; names are never cut.
func Narrowest(gtx layout.Context, ty Type, groups []Group) int {
	lead, narrowest := gtx.Dp(PickSwatchW)+gtx.Dp(PickGap), 0
	for _, g := range groups {
		// A family's name stands across the column, over the swatches, so it
		// asks for the column width rather than a text-sized slot.
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

// Pack deals the families into n columns so that the tallest column is as
// short as it can be, each family whole and none of them out of order: whole
// because a family split across a column boundary reads as two families, and
// in order because the board is read column by column and an out-of-order
// deal would give the same board different reading orders at different window
// widths.
//
// The search is over runs of boundaries in the reading order rather than over
// column assignments, since the number of boundary runs is combinatorially
// far smaller than the number of assignments for the same family count.
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
// empty, for a board with fewer families than columns. Ties go to the first
// arrangement found; the walk starts each boundary as far along the run as it
// can go, so a tie fills the leftmost column first.
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
// whole type stack, every role a surface reads in.
const typeSection = "foundations-type"

// sectionTitleSep is the seam an inventory section's title is written with:
// what the section is, then how to read it. The story's own bands are built
// from exactly that pair — a label at the leading edge and a caption at the
// trailing one — so a borrowed title splits into a band with nothing reworded.
const sectionTitleSep = " — "

// TypeScaleRows is the inventory's type stack as two rows in the story's own
// bands: the heading band the story's own sections wear, over the inventory
// section's own body. The inventory's own title words are kept, split at the
// em dash its titles are already written with, so nothing is reworded here. A
// title with no separator lands as the whole label with no caption.
func TypeScaleRows(inv *inventory.Inventory, p Chrome, c tokens.ColorTokens, ty Type) []layout.Widget {
	for _, s := range inv.Foundations(c) {
		if s.Name != typeSection {
			continue
		}
		label, hint, _ := strings.Cut(s.Title, sectionTitleSep)
		return []layout.Widget{
			Heading(p, c, ty, label, hint),
			Body(c, scaleBody(s)),
		}
	}
	return nil
}

// scaleBody adapts an inventory section's body to the story body's shape: the
// story measures its content and reports the height, while a section body is
// laid out in a slot of the height the section states — bounded, since the
// type stack measures nothing of its own.
func scaleBody(s inventory.Section) func(gtx layout.Context, width int) int {
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
