package chip

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"

	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/hit"
)

// Intent is what a chip is for, and it is the whole of what one chip differs
// from another by: same anatomy, same silhouette, same height. The four are
// exhaustive — a small control that fits none of them is not a chip.
type Intent uint8

const (
	// Assist offers a contextual action on the content beside it. It is
	// clickable and never selected, and it is the one intent whose ink is the
	// page's full-strength text colour: an assist chip proposes something to
	// do and is read at the weight of what it is proposing.
	//
	// It is the zero value, so a [Props] naming no intent draws one.
	Assist Intent = iota

	// Filter narrows a set, and is the only intent that carries selection:
	// clicking it toggles, and a selected filter fills and grows a leading
	// checkmark. Marking a choice is this intent's job and no button's,
	// whatever a button's emphasis.
	Filter

	// Input is a token the reader entered themselves — a recipient, a tag, a
	// file they picked. It carries the trailing dismiss mark, and its leading
	// slot is the avatar slot: whatever glyph it is given is drawn at
	// [AvatarDp] behind a full-round corner rather than in the cap band, because
	// what leads a token the reader entered is a picture of the thing.
	Input

	// Suggestion is a generated prompt the reader may take up — clickable,
	// usually label-only, never selected.
	Suggestion
)

// Selectable reports whether the intent carries selection. Only [Filter] does;
// the others ignore [RenderState.Selected] and [Props.Selected] entirely, so a
// caller cannot draw a selected assist chip by mistake.
func (i Intent) Selectable() bool { return i == Filter }

// Dismissible reports whether the intent carries the trailing dismiss mark.
// Only [Input] does, and it always does: the mark is the intent's anatomy and
// not an option on it — a token the reader entered is a token they can take
// back.
func (i Intent) Dismissible() bool { return i == Input }

// MarkDp is the square a chip's leading icon and its dismiss mark are drawn
// in, in dp: the cap band of the label they stand beside — baseline to cap
// height, measured off the face that label is set in.
//
// A mark inside a chip is read as part of the label's own line, so it rises no
// higher than the capitals and hangs no lower than the baseline. The platform
// draws its marks in exactly that band: measured offscreen against a
// system-font label at three sizes, its plus, check and cross ink out at 1.11
// to 1.21 times the label's cap height, the excess being the half-stroke a
// line straddling the band leaves outside it. That excess is all the licence a
// mark gets here too — the marks below stroke to the edge of this square and
// nothing widens it.
//
// It does not move with the density: the band belongs to the label's size, and
// the height the chip loses at Compact comes out of its air rather than out of
// its marks. It does move with the label's role, which is why it is a relation
// and not a number.
func MarkDp(style tokens.TextStyle) float32 { return style.FaceMetrics().CapHeight }

// AvatarDp is the square an [Input] chip's leading glyph is drawn in, behind a
// full-round corner. Larger than [MarkDp] because it is a picture of a thing
// rather than a sign for one, and round because that is what separates the two
// at a glance.
const AvatarDp = 24

// DismissHitDp is the side of the pointer target the dismiss mark registers,
// in dp, centred on the mark and free to overhang the chip.
//
// It is WCAG 2.5.8 Target Size (Minimum), the AA criterion, and not the 44 dp
// of [tokens.MinHitTarget]: 44 is this system's floor for a standalone control
// with space around it, and a 44 dp target centred on the mark would reach
// past both ends of the chip carrying it.
const DismissHitDp = 24

// edgeDp is the outline's width — one hair at every density, the width every
// other derived edge in this library is drawn at. It is a width rather than a
// token because no scale in the system carries line weights.
const edgeDp = unit.Dp(1)

// MarkStrokeDp is the line weight a chip's stroked marks are drawn at, in dp:
// the width of the label's own upright stem.
//
// The marks and the words are one utterance, so they are made of ink of one
// weight — which the platform bears out. Measured offscreen at three sizes,
// its plus, check and cross carry a stroke band of 0.82 to 1.02 times their
// label's stem, diagonals included: a diagonal's horizontal run is wider by
// its angle and its band is not.
func MarkStrokeDp(style tokens.TextStyle) float32 { return style.FaceMetrics().Stem }

// Glyph is the painter a chip draws its leading mark with: it fills a
// sizePx×sizePx box at the current origin in colour col. It is the same
// signature components/button gives an icon-only button and the same one
// components/icon's registry hands out, so a named glyph, a clip.Path drawn by
// hand and a picture built for one screen are interchangeable here.
//
// That box is the label's cap band — see [MarkDp] — everywhere but an [Input]
// chip's leading slot, which is the avatar slot at [AvatarDp]. A painter that
// strokes to the edge of the box it is handed lands in the band the words
// occupy, which is what a mark inside a chip is for; one that draws a small
// figure in the middle of it reads as a chip with a hole at its leading end.
//
// A nil Glyph draws no leading mark; the chip loses the mark and the gap after
// it and nothing else. A painter that draws its own picture may ignore col;
// one that draws a sign must honour it, because col is what the chip derived
// against the body actually drawn.
type Glyph func(gtx layout.Context, sizePx int, col color.NRGBA)

// RenderState holds the explicit visual state a static chip render draws in.
// The zero value is a resting, unselected chip on the window ground, so
// RenderState{} is the default chip.
//
// Intended for golden-image testing and static rendering; production code
// obtains the interaction half from the Gio event system.
type RenderState struct {
	// Ground is the elevation storey of the surface hosting the chip, in the
	// same vocabulary the host names its own fill (tokens.SurfaceAt). An
	// unselected chip carries no colour of its own, so the storey is what its
	// body is painted in and what its ink is floored against. A dialog at
	// tokens.Level2 passes Level2. The zero value is tokens.Level0, the
	// window ground.
	Ground tokens.ElevationLevel

	// Selected is honoured only where [Intent.Selectable] is true. A selected
	// chip drops its outline, fills, and leads with a checkmark.
	Selected bool

	Hovered bool
	Pressed bool
	Focused bool
}

// state is the token vocabulary's name for the interaction the chip is in.
// Press wins over hover, because a pressed control is under the pointer by
// definition and the deeper walk is the one that has something to say.
func (s RenderState) state() tokens.State {
	switch {
	case s.Pressed:
		return tokens.StatePressed
	case s.Hovered:
		return tokens.StateHover
	}
	return tokens.StateNormal
}

// Colors is the set one chip draws with, resolved together by [Resolve].
//
// It is exported because anything drawn behind or beside a chip — a band
// deciding what its own seam must clear, a test measuring a pairing — needs
// the answers the chip drew with, and re-deriving them at the call site is how
// two answers appear.
type Colors struct {
	// Fill is what the chip's body is painted in. On an unselected chip at
	// rest it is the storey the caller named, which is what leaves the outline
	// carrying the whole appearance; under the pointer it is that storey
	// walked, and on a selected chip the secondary container walked.
	Fill color.NRGBA

	// Outline is the resting body's one hair of edge, and Outlined is whether
	// it is drawn. A selected chip has no outline: the fill has arrived and
	// the edge is not needed twice.
	Outline  color.NRGBA
	Outlined bool

	// Label is the ink the words are set in, floored at [tokens.TextFloor]
	// against [Colors.Fill].
	Label color.NRGBA

	// Mark is the colour the leading checkmark, the leading glyph and the
	// dismiss mark are drawn in, floored at [tokens.GraphicFloor] against
	// [Colors.Fill].
	Mark color.NRGBA
}

// Resolve returns the colours a chip of intent i draws with in state s.
//
// Selected and unselected are two derivations, not one with a switch in it:
//
//	unselected  body   the storey itself, walked by the pointer — no colour
//	                   of the chip's own
//	            edge   OutlineVariant while it clears the graphic floor on
//	                   both sides, the floored neutral rung otherwise
//	            ink    OnSurfaceVariant, or the Text pin for Assist, each held
//	                   to its floor against the body actually drawn
//	  selected  body   the secondary container against the storey, walked by
//	                   the pointer and stopped where it stops being a chip
//	            edge   none
//	            ink    InkOn(RoleSecondary, body, TextFloor) for the words,
//	                   OnContainer's own rule against that body for the marks
//
// The walk is the same one every state in this system takes
// ([tokens.ColorTokens.PinnedStateColor]) and it is the whole of the feedback
// grammar here: rest is where the two derivations differ, and hover and press
// follow from whichever rest they started at.
//
// Both inks are resolved against the body ACTUALLY drawn rather than against
// the resting one. A colour derived against a surface the walk has since moved
// is a floor that was met once: measured on the family this replaces, 4.5:1 at
// rest became 2.3:1 pressed. On a selected chip the marks are
// [tokens.ColorTokens.OnContainer]'s own derivation — the role's mark against
// its container at the graphic floor — re-run against the container the chip
// is actually wearing.
//
// The words take the text floor and the marks the graphic one, which is the
// split those two floors are for: WCAG 1.4.3's 4.5:1 is what a run of words
// owes, and 1.4.11's 3:1 is what a shape that must be resolved owes.
func Resolve(c tokens.ColorTokens, i Intent, s RenderState) Colors {
	st := s.state()
	ground := c.SurfaceAt(s.Ground)
	if s.Selected && i.Selectable() {
		fill := walk(c, c.ContainerOn(tokens.RoleSecondary, ground), st, func(fill color.NRGBA) bool {
			// A selected chip carries no outline, so its fill is the whole of
			// what separates it from the page as well as the ground its own
			// words are read on.
			return vgcolor.ContrastRatio(fill, ground) >= tokens.ContainerFloor &&
				vgcolor.ContrastRatio(c.InkOn(tokens.RoleSecondary, fill, tokens.TextFloor), fill) >= tokens.TextFloor
		})
		return Colors{
			Fill:  fill,
			Label: c.InkOn(tokens.RoleSecondary, fill, tokens.TextFloor),
			Mark:  c.MarkOn(tokens.RoleSecondary, fill, tokens.GraphicFloor),
		}
	}
	fill := walk(c, ground, st, func(fill color.NRGBA) bool { return writable(c, fill) })
	pin := c.OnSurfaceVariant()
	if i == Assist {
		pin = c.Text
	}
	return Colors{
		Fill:     fill,
		Outline:  outlineOver(c, ground, fill),
		Outlined: true,
		Label:    neutralInk(c, pin, fill, tokens.TextFloor),
		Mark:     neutralInk(c, pin, fill, tokens.GraphicFloor),
	}
}

// walk is the state walk and where it stops: [tokens.ColorTokens.PinnedStateColor]
// from rest, held back to the last depth on the way that good still accepts.
//
// The stop is a condition on being a chip at all rather than a second thought
// about the walk. The walk is depth on the neutral ladder and a ramp writes
// with its ends, so between them lies a band of depths no rung reaches the text
// floor against; a body nothing can be written on is not a state to walk to. A
// selected chip adds the second condition, because it carries no outline and a
// walk that took its fill through the ground's own depth would erase the chip
// at the crossing.
//
// The depth is found by measuring the realized tone rather than by solving for
// the boundary, because a tone is realized in 8-bit sRGB and a depth solved
// exactly on the edge rounds to either side of it; halving the interval keeps
// the answer on the side that measured good. A resting body that already fails
// is left alone — that is a defect in the resting appearance and belongs to the
// gates, not to a state walk.
func walk(c tokens.ColorTokens, rest color.NRGBA, st tokens.State, good func(color.NRGBA) bool) color.NRGBA {
	walked := c.PinnedStateColor(rest, st)
	if good(walked) || !good(rest) {
		return walked
	}
	restL, _, _ := vgcolor.LabFromNRGBA(rest)
	walkedL, _, _ := vgcolor.LabFromNRGBA(walked)
	_, chroma, hue := vgcolor.OKLChFromNRGBA(walked)
	// lo is always a depth that measured good, hi one that did not.
	lo, hi := restL, walkedL
	for range 24 {
		mid := (lo + hi) / 2
		if good(vgcolor.NRGBAFromToneChromaHue(mid, chroma, hue)) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return vgcolor.NRGBAFromToneChromaHue(lo, chroma, hue)
}

// writable reports whether a label can be set on fill at all, and it asks with
// the MUTED pin rather than the full-strength one: the muted rung is a rung of
// the neutral ramp, so a depth it reaches is one the ramp reaches, and the
// walk must stop at the same depth for all four intents. A body whose depth
// depended on which intent stood on it would put two chips in one row at two
// different depths under one pointer.
func writable(c tokens.ColorTokens, fill color.NRGBA) bool {
	return vgcolor.ContrastRatio(neutralInk(c, c.OnSurfaceVariant(), fill, tokens.TextFloor), fill) >= tokens.TextFloor
}

// outlineOver is the unselected chip's edge: the boundary token while it holds
// the graphic floor on both sides of the edge, and a floored neutral rung
// otherwise.
//
// [tokens.ColorTokens.OutlineVariant] is floored by construction against
// Surface and Background, which is the pair a neutral boundary is drawn over
// when nobody names a storey. A chip does name one, and the ladder reaches
// past that pair: on the dark scheme's level-3 plane the token measures
// 1.80:1. So it is a pin and not an answer — used while it reads, walked when
// it stops, which is the idiom every other derived colour in this package
// takes.
//
// An edge has two sides and one colour, and the inside of this one moves: the
// body walks a rung and two under the pointer. Both sides are asked for, and
// where no rung can clear both — the deep end of a press on a high storey, at
// which point the two neighbours are further apart than twice the floor — the
// OUTER side keeps the colour. What the outline separates the chip from is the
// page; what is inside it is the chip's own transient state, already under the
// reader's pointer when the two collide.
func outlineOver(c tokens.ColorTokens, ground, fill color.NRGBA) color.NRGBA {
	cands := [...]color.NRGBA{
		c.OutlineVariant(),
		c.MarkOn(tokens.RoleNeutral, ground, tokens.GraphicFloor),
		c.MarkOn(tokens.RoleNeutral, fill, tokens.GraphicFloor),
	}
	clears := func(cand, over color.NRGBA) bool {
		return vgcolor.ContrastRatio(cand, over) >= tokens.GraphicFloor
	}
	for _, cand := range cands {
		if clears(cand, ground) && clears(cand, fill) {
			return cand
		}
	}
	for _, cand := range cands {
		if clears(cand, ground) {
			return cand
		}
	}
	return cands[1]
}

// neutralInk is [tokens.ColorTokens.InkOn]'s rule for a pin no role owns: the
// pin while it clears floor against ground, and otherwise the rung of the
// neutral ramp nearest its mid-value step that does.
//
// InkOn itself asks a role for its pinned base and RoleNeutral has none, so
// the rule is spelled out here rather than reinvented: pin first, walk only
// when the pin stops reading.
func neutralInk(c tokens.ColorTokens, pin, ground color.NRGBA, floor float64) color.NRGBA {
	if vgcolor.ContrastRatio(pin, ground) >= floor {
		return pin
	}
	return c.MarkOn(tokens.RoleNeutral, ground, floor)
}

// Pin is the edge of the box a chip is offered that its body is pinned to.
//
// It is a placement, not a stretch: the chip stays sized to its content — see
// the package doc — and what changes is where in the offered box it is drawn
// and how much of that box the widget reports having used. Only the horizontal
// axis is pinned, because the vertical one is already settled by whatever row
// the chip stands in.
//
// The seam exists because a chip alone can be placed by its container and a
// chip handed on to a container that centres whatever it is given cannot: the
// reserved cap and the drawn chip then part company by half the slack, and the
// only place both widths are known is inside the chip. A pin says it there,
// once.
//
// It costs the container the drawn rect, which is the whole box as far as it
// can tell, so say it only where nothing upstream needs that rect. A container
// that aligns what it is given needs no pin at all.
type Pin uint8

const (
	// PinNone is the zero value and the chip's own habit: the widget reports
	// the chip it drew and no more, so a row of chips is laid out at their own
	// scale and the box around them is the container's business.
	PinNone Pin = iota

	// PinLeading draws the chip at the leading edge of the offered box.
	PinLeading

	// PinTrailing draws the chip at the trailing edge of the offered box.
	PinTrailing
)

// Layout draws w at p's edge of the box the widget was offered — the
// horizontal half of gtx.Constraints.Max — and reports that box rather than
// w's own size, which is what lets a caller upstream find the pinned edge
// where it asked for it. PinNone lays w out untouched, so a chip that pins
// nothing pays nothing.
//
// The whole widget is offset, slop and all, so the pointer target stays
// centred on the chip it was extended around.
func (p Pin) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	if p == PinNone {
		return w(gtx)
	}
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()

	box := dims.Size
	box.X = max(box.X, gtx.Constraints.Max.X)
	off := 0
	if p == PinTrailing {
		off = box.X - dims.Size.X
	}
	o := op.Offset(image.Pt(off, 0)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	o.Pop()
	return layout.Dimensions{Size: box, Baseline: dims.Baseline}
}

// Props configures a [Chip] instance: what it says, which intent it says it
// in, what it stands on, and how an activation is delivered.
//
// There is no emphasis field and there will not be one. Intent is the only
// axis a chip varies on — see the package doc — and there is no Disabled field
// either: a chip is clickable by construction, and something a reader can only
// read is components/badge.
type Props struct {
	// Label is the text the chip carries.
	Label string

	// Intent is what this chip is for. The zero value is [Assist].
	Intent Intent

	// Icon is the mark drawn before the label, in the leading slot: the
	// label's cap band ([MarkDp]) square, or [AvatarDp] behind a full-round
	// corner when the intent is [Input]. A nil Icon draws none.
	//
	// A selected [Filter] chip draws the checkmark here instead — the mark
	// that says it is selected takes the slot rather than standing beside a
	// second one.
	Icon Glyph

	// Selected is the selection a [Filter] chip is seeded with on subscribe.
	// The live chip keeps its own selection from there — a later Selected does
	// not move a running instance; rebuild the subscription to reseed one,
	// which is the idiom components/picker states for the same reason. Every
	// other intent ignores it.
	Selected bool

	// Description is the screen-reader label. Falls back to Label when empty.
	Description string

	// Ground is the elevation storey of the surface hosting the chip, copied
	// straight into [RenderState.Ground] on every frame. A dialog at
	// tokens.Level2 passes Level2. The zero value is tokens.Level0, the window
	// ground. See [RenderState.Ground].
	Ground tokens.ElevationLevel

	// Pin is the edge of the offered box the chip is drawn at. The zero value
	// is [PinNone] and the chip reports itself alone, which is every chip laid
	// out by its own container. Set it where the box is a cap the caller sized
	// and something between the caller and the chip does the placing — see
	// [Pin].
	Pin Pin

	// Clickable, if non-nil, is used instead of an internally-allocated one.
	// The caller then owns &Clickable as the chip's focus tag — usable with
	// key.FocusCmd, key.Filter{Focus: …} and an external Tab cycle — and may
	// detect activation via Clickable.Clicked(gtx). This is what lets a
	// container that drives focus itself avoid a doubled focus ring. When nil
	// the chip allocates and owns its own clickable, which survives every
	// theme emission.
	Clickable *widget.Clickable

	// OnClick is called when the chip's body is activated by click or
	// Space/Enter, whatever the intent. On a [Filter] chip it fires with
	// OnSelect, after the selection has already moved.
	//
	// The gtx argument is the layout.Context active on the frame the
	// activation is processed in, so a consumer may emit
	// mvu.MessageOp{Message: …}.Add(gtx.Ops) from inside it.
	OnClick func(gtx layout.Context)

	// OnSelect is called with the selection the [Filter] chip has just moved
	// to. Other intents never call it.
	OnSelect func(gtx layout.Context, selected bool)

	// Message, if non-nil, is emitted as mvu.MessageOp into gtx.Ops on
	// activation — the MVU path, where OnClick is the FRP one. Both fire when
	// both are set, and they fire from the one place the activation is
	// noticed: the chip polls its clickable once per frame, so a click and a
	// Space both arrive through the same branch and neither can dispatch
	// twice.
	Message any

	// OnDismiss is called when an [Input] chip's dismiss mark is clicked. It
	// reports that the reader asked for this token to go away and nothing
	// more: the chip does not remove itself on the next frame.
	//
	// Other intents never call it, and an Input chip draws its mark whether or
	// not it is set — the mark is the intent's anatomy. What a nil OnDismiss
	// costs is only the dispatch.
	OnDismiss func(gtx layout.Context)

	// DismissMessage, if non-nil, is emitted as mvu.MessageOp into gtx.Ops on
	// dismissal — the MVU path, where OnDismiss is the FRP one. Both fire when
	// both are set, from the one place the click is noticed.
	DismissMessage any

	// Shaper is an explicit per-instance override of the text shaper. Leave it
	// nil in normal use: the chip then shapes with the theme's shaper
	// (tokens.Typography.Shaper()), built once for the process and shared by
	// every component reading that typography. Set it only when this chip must
	// shape with a different one — a golden test pinning its faces.
	Shaper *text.Shaper
}

// resolvedTokens is the concrete per-emission snapshot the widget closure
// draws from: the whole theme flattened to the values one frame needs.
type resolvedTokens struct {
	color   tokens.ColorTokens
	label   tokens.TextStyle // the LabelLarge role
	spacing tokens.SpacingScale
	radius  tokens.RadiusScale
	density tokens.Density
	shaper  *text.Shaper
}

// Chip returns an rx.Observable[layout.Widget] emitting a new widget whenever
// the theme changes. It is the live face of [Render]: the same anatomy, drawn
// from the theme rather than from tokens handed in, with the four things the
// pure path cannot carry — the pointer areas, the keyboard, the [Filter]
// chip's own selection, and the dispatch.
//
// The pointer target is extended to the density's [tokens.Density.MinHitTarget]
// (44 dp, WCAG 2.5.5) on both axes, centred on the drawn chip, exactly as
// components/button extends its own: the chip draws at the density's chip
// height and what the pointer may land on does not shrink with it. The widget
// still reports the chip's size, so a row of chips is laid out at their own
// scale and the slop overhangs the air around them — unless [Props.Pin] asks
// for the box instead, in which case the slop travels with the chip it was
// centred on.
//
// An [Input] chip's dismiss mark registers a second, smaller target
// ([DismissHitDp]) over the body's, and the body keeps walking while the
// pointer is on it: the mark is part of the chip, so a reader reaching for it
// must not see the chip go cold.
//
// Keyboard activation is gioui.org/widget.Clickable's: the chip is focusable,
// Space and Enter activate it, and gtx.Focused drives [RenderState.Focused] —
// so a focused chip wears the ring the package doc describes. Both integration
// paths are supported and both are read off the one poll of the clickable:
//   - FRP: set Props.OnClick, Props.OnSelect, Props.OnDismiss.
//   - MVU: set Props.Message and Props.DismissMessage.
//
// Widget state — hover, press, focus, and a filter's selection — lives in the
// rx.Defer scope and survives every theme emission for the life of the
// subscription.
func Chip(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	// Flatten the nested theme observables into one concrete snapshot. The
	// typography emission carries both the LabelLarge role the chip is set in
	// and the theme's cached shaper (ADR-003: the theme owns the typeface).
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Color, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.ColorTokens, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					color:   n.First,
					label:   typ.LabelLarge,
					spacing: n.Third,
					radius:  n.Fourth,
					density: n.Fifth,
					shaper:  typ.Shaper(),
				}
			},
		)
	})

	return rx.Defer(func() rx.Observable[layout.Widget] {
		// Allocated once per subscription, so hover, press, focus and the
		// filter's selection survive every theme emission. ownClick is used
		// only when the caller supplies no clickable.
		var ownClick, dismiss widget.Clickable
		selected := props.Selected && props.Intent.Selectable()

		return rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			desc := props.Description
			if desc == "" {
				desc = props.Label
			}

			return func(gtx layout.Context) layout.Dimensions {
				click := props.Clickable
				if click == nil {
					click = &ownClick
				}

				// Both clickables are drained every frame — a queued click
				// left behind would fire on a later frame against a token the
				// caller has already taken away — and both are drained to one
				// event, because a double click on a mark is one dismissal.
				dismissed := false
				if props.Intent.Dismissible() {
					for dismiss.Clicked(gtx) {
						dismissed = true
					}
				}
				activated := false
				for click.Clicked(gtx) {
					activated = true
				}

				if dismissed {
					// The mark wins the frame. Its pointer area lies over the
					// body's and Gio delivers to both, so the exclusivity the
					// anatomy implies — the mark is a hole in the chip, not a
					// second thing on top of it — is the component's to
					// enforce, once, here.
					if props.OnDismiss != nil {
						props.OnDismiss(gtx)
					}
					if props.DismissMessage != nil {
						mvu.MessageOp{Message: props.DismissMessage}.Add(gtx.Ops)
					}
				} else if activated {
					// One poll, one dispatch: Clicked reports a pointer click
					// and a Space or Enter alike, so both paths leave from
					// here.
					if props.Intent.Selectable() {
						selected = !selected
						if props.OnSelect != nil {
							props.OnSelect(gtx, selected)
						}
					}
					if props.OnClick != nil {
						props.OnClick(gtx)
					}
					if props.Message != nil {
						mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
					}
				}

				s := RenderState{
					Ground:   props.Ground,
					Selected: selected,
					// The dismiss mark's area lies over the body's and takes
					// the pointer from it, so the body reads both: a chip
					// whose mark is under the finger is a chip under the
					// finger.
					Hovered: click.Hovered() || dismiss.Hovered(),
					Pressed: click.Pressed() || dismiss.Pressed(),
					Focused: gtx.Focused(click),
				}

				return props.Pin.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), click.Layout,
						func(gtx layout.Context) layout.Dimensions {
							return draw(gtx, shaper, props.Label, props.Intent, props.Icon,
								tok, s, desc, &dismiss)
						})
				})
			}
		})
	})
}

// Render produces a layout.Widget drawing the chip in an explicit visual
// state, without event processing: the leading mark, the label and — for
// [Input] — the dismiss mark on one row, inside the body the intent and the
// state resolve to.
//
// icon may be nil, in which case the chip leads with its label; a selected
// [Filter] chip leads with the checkmark whether or not one was given.
// labelStyle is the whole text style the label is set in; pass
// tokens.DefaultTypography.LabelLarge, which is the role a chip is set in at
// either density.
//
// The chip is sized to its content, clamped to the constraints it is handed,
// and asks for the pointer cursor. Registering pointer areas — the body's 44 dp
// floor and the dismiss mark's own — is the live path's job; see [Chip].
func Render(
	shaper *text.Shaper,
	label string,
	i Intent,
	icon Glyph,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	labelStyle tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, label: labelStyle, spacing: sp, radius: rad, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return draw(gtx, shaper, label, i, icon, tok, s, label, nil)
	}
}

// draw paints one chip: the body the intent resolves to, the leading mark, the
// label, and the dismiss mark the [Input] intent carries.
func draw(
	gtx layout.Context,
	shaper *text.Shaper,
	label string,
	i Intent,
	icon Glyph,
	tok resolvedTokens,
	s RenderState,
	desc string,
	dismiss *widget.Clickable,
) layout.Dimensions {
	selected := s.Selected && i.Selectable()
	col := Resolve(tok.color, i, s)

	padH := gtx.Dp(unit.Dp(tok.density.PaddingX))
	gap := gtx.Dp(unit.Dp(tok.spacing.S2))

	// The marks are capped at the body's INNER height — the chip's own height
	// less the edge on both sides — so a mark never lies on the outline it
	// stands inside. It binds at Compact, where the density's chip height and
	// the avatar slot are the same number; the cap band is well under both.
	band := max(gtx.Dp(edgeDp), 1)
	chipH := gtx.Dp(unit.Dp(tok.density.ChipHeight()))
	iconPx := min(gtx.Dp(unit.Dp(MarkDp(tok.label))), chipH-2*band)

	// The leading slot: the checkmark on a selected filter, the avatar on an
	// input chip that was given a glyph, the icon otherwise.
	lead, avatar := 0, false
	switch {
	case selected:
		lead = iconPx
	case icon == nil:
	case i == Input:
		lead, avatar = min(gtx.Dp(unit.Dp(AvatarDp)), chipH-2*band), true
	default:
		lead = iconPx
	}
	trail := 0
	if i.Dismissible() {
		trail = iconPx
	}
	leadGap, trailGap := 0, 0
	if lead > 0 && label != "" {
		leadGap = gap
	}
	if trail > 0 {
		trailGap = gap
	}

	// Record the label's material and its layout to learn its size before
	// anything is painted. typeset.Layout rather than widget.Label.Layout
	// because the role's line height has to be the height of the label box and
	// Gio alone reports the glyph ink instead — see theme/typeset.
	labelDims := layout.Dimensions{}
	var labelCall op.CallOp
	if label != "" {
		mColor := op.Record(gtx.Ops)
		paint.ColorOp{Color: col.Label}.Add(gtx.Ops)
		material := mColor.Stop()

		labelGtx := gtx
		labelGtx.Constraints.Min = image.Point{}
		if w := gtx.Constraints.Max.X - 2*padH - lead - leadGap - trailGap - trail; w > 0 {
			labelGtx.Constraints.Max.X = w
		}
		mLabel := op.Record(gtx.Ops)
		labelDims = typeset.Layout(labelGtx, shaper,
			typeset.Label(tok.label, 1), typeset.Font(tok.label, font.Normal),
			unit.Sp(tok.label.Size), label, material)
		labelCall = mLabel.Stop()
	}

	// Sized to content, not to the width it was given: a chip is something
	// content sprouted, and one that stretched would be a banner.
	//
	// The height is the density's chip height outright — not a floor under
	// max(content, ControlHeight + padding), which is the rule for controls on
	// the ladder the chip has just left. A chip is shorter than a button by
	// construction and its label's line box fits inside that height at both
	// densities, so the only thing that can push it taller is a caller's own
	// oversized style.
	h := max(chipH, labelDims.Size.Y)
	h = min(h, gtx.Constraints.Max.Y)

	// The leading padding is the text padding, EXCEPT in front of the avatar,
	// where it is the clearance the avatar already has above and below itself.
	// A round picture set in a square well is what an address field draws, and
	// the text padding in front of it leaves four times that clearance on one
	// side of the picture and none on the other three — a hole at the leading
	// end. Stated as the relation rather than as a number so it holds at both
	// densities and wherever the avatar is capped by the body's inner height.
	leadPad := padH
	if avatar {
		leadPad = (h - lead) / 2
	}

	w := leadPad + lead + leadGap + labelDims.Size.X + trailGap + trail + padH
	w = min(w, gtx.Constraints.Max.X)
	size := image.Pt(w, h)
	box := image.Rectangle{Max: size}

	// The edge, as nested fills — the shape in the edge's colour, the body
	// inset by one hair inside it — and not as a stroke on the shape's path. A
	// stroke is centred on its path, so half a hair of it would fall outside
	// the box the widget reports and every pixel of it would be a blend of the
	// two colours rather than either.
	//
	// A focused chip's edge IS the focus ring: the ring replaces the outline
	// rather than being drawn inside it. Drawn inside, the two make a
	// three-line sandwich — hairline, a pixel of body, then the ring — which
	// reads as a dirty halo, the same "a band beside a boundary reads as part
	// of that boundary" that holds components/button's ring clear of its edge.
	// Nothing else moves: the chip measures the same box focused as at rest,
	// and the label does not shift.
	radius := min(gtx.Dp(unit.Dp(tok.radius.Lg)), h/2)
	edgeInk, edged := col.Outline, col.Outlined
	if s.Focused {
		band, edgeInk, edged = gtx.Dp(focus.Width), focus.Ring(tok.color), true
	}
	inner, innerRad := box, radius
	if edged {
		paint.FillShape(gtx.Ops, edgeInk, rrect(gtx.Ops, box, radius))
		if in := box.Inset(band); in.Dx() > 0 && in.Dy() > 0 {
			inner, innerRad = in, max(radius-band, 0)
		}
	}
	paint.FillShape(gtx.Ops, col.Fill, rrect(gtx.Ops, inner, innerRad))

	// One row, leading edge to trailing: mark, label, dismiss mark. The row is
	// laid from the leading padding rather than centred in the box, because
	// the anatomy is read from its leading edge and a chip clamped narrower
	// than its content must lose its trailing end and not both.

	// Where the cap band sits in the chip's own coordinates: the label's
	// baseline, less the band's height. typeset reports the baseline from the
	// bottom of the box it laid the label out in, and that report is the only
	// place this is knowable — the line box is taller than the ink and the
	// leading it adds is not split evenly around the ink. Centring the mark in
	// the chip instead lands it a pixel low, because the band the capitals
	// occupy is not centred on the line box that holds them.
	//
	// A label-less chip has no band to sit on, so its mark keeps the chip's own
	// middle.
	markY := (h - iconPx) / 2
	if label != "" {
		baseline := (h-labelDims.Size.Y)/2 + labelDims.Size.Y - labelDims.Baseline
		markY = baseline - iconPx
	}

	x := leadPad
	if lead > 0 {
		// The avatar is a picture rather than a mark on the line: it is taller
		// than the band the words occupy and it is centred on the chip.
		leadY := markY
		if avatar {
			leadY = (h - lead) / 2
		}
		lo := op.Offset(image.Pt(x, leadY)).Push(gtx.Ops)
		switch {
		case selected:
			drawCheck(gtx, lead, stroke(gtx, tok.label), col.Mark)
		case avatar:
			// The avatar slot is corner-full, and the clip is the chip's to
			// apply: a painter handed a square box would otherwise put square
			// corners in a round slot.
			cl := clip.RRect{Rect: image.Rectangle{Max: image.Pt(lead, lead)},
				NW: lead / 2, NE: lead / 2, SE: lead / 2, SW: lead / 2}.Push(gtx.Ops)
			icon(gtx, lead, col.Mark)
			cl.Pop()
		default:
			icon(gtx, lead, col.Mark)
		}
		lo.Pop()
		x += lead + leadGap
	}
	if label != "" {
		lo := op.Offset(image.Pt(x, (h-labelDims.Size.Y)/2)).Push(gtx.Ops)
		labelCall.Add(gtx.Ops)
		lo.Pop()
		x += labelDims.Size.X
	}

	// The chip's own semantic node, scoped to the box it drew. A semantic op
	// attaches to the innermost clip area around it, so a chip that emitted
	// its label without an area of its own would write that label onto
	// whatever area encloses it.
	//
	// A filter chip is a checkbox to a screen reader and says which way it is
	// set; every other intent is a button, because that is what activating it
	// does. The dismiss mark stays outside this area on purpose: a pointer
	// area is clipped by the areas above it, and the mark's target is
	// deliberately larger than the mark.
	sem := clip.Rect{Max: size}.Push(gtx.Ops)
	if i.Selectable() {
		semantic.ClassOp(semantic.CheckBox).Add(gtx.Ops)
		semantic.SelectedOp(selected).Add(gtx.Ops)
	} else {
		semantic.ClassOp(semantic.Button).Add(gtx.Ops)
	}
	semantic.LabelOp(label).Add(gtx.Ops)
	semantic.DescriptionOp(desc).Add(gtx.Ops)
	semantic.EnabledOp(true).Add(gtx.Ops)
	sem.Pop()

	pointer.CursorPointer.Add(gtx.Ops)

	if trail > 0 {
		origin := image.Pt(x+trailGap, markY)
		mo := op.Offset(origin).Push(gtx.Ops)
		drawCross(gtx, trail, stroke(gtx, tok.label), col.Mark)
		mo.Pop()
		registerDismissTarget(gtx, desc, origin, trail, dismiss)
	}
	return layout.Dimensions{Size: size}
}

// rrect is the chip's silhouette: a rounded rectangle at one radius on all
// four corners.
func rrect(ops *op.Ops, box image.Rectangle, r int) clip.Op {
	return clip.RRect{Rect: box, NW: r, NE: r, SE: r, SW: r}.Op(ops)
}

// stroke is the mark weight in pixels, never under one: a zero or unset metric
// would erase a mark and a sub-pixel width would leave it a smear, and neither
// is better than the thinnest stroke that draws.
func stroke(gtx layout.Context, style tokens.TextStyle) float32 {
	if w := MarkStrokeDp(style) * gtx.Metric.PxPerDp; w >= 1 {
		return w
	}
	return 1
}

// drawCheck strokes the selection mark in a size×size box at the current
// origin: a check whose foot sits below its own midline, so the shorter arm
// reads as an arm and not as a serif.
//
// The path runs to the edges of the box and the stroke straddles them, so half
// a stroke of ink lies outside the cap band on each side. That is the whole of
// the optical licence the platform's own marks take — see [MarkDp] — and it is
// spent by the shape rather than granted as a larger box.
func drawCheck(gtx layout.Context, size int, w float32, c color.NRGBA) {
	x0, x1 := float32(0), float32(size)
	y0, y1 := float32(0), float32(size)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(x0, (y0+y1)/2))
	p.LineTo(f32.Pt(x0+(x1-x0)*0.36, y1))
	p.LineTo(f32.Pt(x1, y0))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: w}.Op())
}

// drawCross strokes the dismiss mark in a size×size box at the current origin,
// to the same band and the same licence as [drawCheck].
func drawCross(gtx layout.Context, size int, w float32, c color.NRGBA) {
	x0, y0 := float32(0), float32(0)
	x1, y1 := float32(size), float32(size)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(x0, y0))
	p.LineTo(f32.Pt(x1, y1))
	p.MoveTo(f32.Pt(x1, y0))
	p.LineTo(f32.Pt(x0, y1))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: w}.Op())
}

// registerDismissTarget puts the clickable's pointer area over the dismiss
// mark, grown to [DismissHitDp] on each axis and centred on it.
//
// The chip's own reported size is unaffected: a caller laying chips out spaces
// the chips it can see, not the slop behind them. The area is registered after
// the body's, so it takes the pointer where the two overlap — which is why the
// body reads this clickable's hover as its own.
func registerDismissTarget(gtx layout.Context, desc string, origin image.Point, mark int, dismiss *widget.Clickable) {
	if dismiss == nil {
		return
	}
	target := max(gtx.Dp(unit.Dp(DismissHitDp)), mark)
	off := op.Offset(image.Pt(origin.X-(target-mark)/2, origin.Y-(target-mark)/2)).Push(gtx.Ops)
	dismiss.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.ClassOp(semantic.Button).Add(gtx.Ops)
		// The chip's own words name the target: what the mark removes is this
		// token, and a reader reaching the mark should be told which one
		// rather than a word this package invented for it.
		semantic.LabelOp(desc).Add(gtx.Ops)
		semantic.EnabledOp(true).Add(gtx.Ops)
		pointer.CursorPointer.Add(gtx.Ops)
		return layout.Dimensions{Size: image.Pt(target, target)}
	})
	off.Pop()
}
