package picker

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/key"
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
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/components/internal/focus"
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/components/list"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// The form trigger's mark is the platform's pop-up mark, and both of this
// library's pop-up triggers draw one drawing at one size: see
// [control.DrawMark], which holds the measurement and the geometry.

// edgeDp is the hairline the open menu's plane is drawn with, and so the
// distance from the plane's outer edge to its inner one. The trigger draws no
// edge at all — see [drawTrigger].
const edgeDp = unit.Dp(1)

// dismissReach is how far the outside-press absorber reaches beyond the box
// the open field was offered, on every side. A component cannot see the
// window from inside its own layout, so the reach is simply larger than any
// display and whatever the field stands in clips it back.
const dismissReach = unit.Dp(8192)

// Drop is the side an open [Field] PREFERS to spend its overflow on.
//
// An open menu does not drop. It stands OVER the trigger with the row the
// picker is holding on the trigger's own label, which is what this platform's
// pop-up button does: the choice the field is showing does not move when the
// menu opens over it, and the rest of the catalogue is laid above and below it
// — see [alignOverTrigger].
//
// Drop answers what is left after that: which way the menu is pushed when the
// room the caller reports ([FieldProps.AvailableRoom]) cannot hold it where it
// wants to stand. A field near the foot of a dialog has little room below, so
// its menu is pushed up; one near the top is pushed down. The push is worked
// out from the room itself, and Drop settles it only where BOTH ends are
// short — a menu taller than the whole room is capped to the room and scrolls
// inside it, and Drop says which end of the room it is anchored to while the
// rows move under it. With no room reported nothing says the menu cannot stand
// where it wants to, and it does.
//
// Either way the open field reports its TRIGGER and nothing else. The menu is
// a floating surface deferred to the end of the frame, and a floating surface
// asks the container its anchor stands in for no room: an open field is placed
// exactly where a closed one is, and where the menu lands changes what it
// covers, never the box the field reports.
//
// The trigger's mark says nothing about it. The platform's pop-up mark is a
// pair of chevrons pointing opposite ways — it says the choice can move
// either way and cannot say a direction — so a field whose menu is pushed
// upwards draws the same trigger as one whose menu is pushed down. See
// [control.DrawMark].
type Drop uint8

const (
	// DropDown is the zero value: a menu that cannot stand where it wants to
	// takes the room below the trigger first.
	DropDown Drop = iota

	// DropUp takes the room above the trigger first.
	DropUp
)

// FieldState holds the explicit visual state a static field render draws in.
// All fields default to their zero values: closed, unfocused, enabled.
//
// Intended for golden-image testing; production code obtains state from the
// Gio event system via [Field].
type FieldState struct {
	Open     bool
	Focused  bool
	Disabled bool
	Selected int
	Options  []string

	// Hovered lays the platform's hover overlay over the trigger's fill;
	// Pressed lays its press overlay there instead, a press winning over a
	// hover. A trigger whose menu stands (Open) takes the pressed drawing
	// too, for as long as it stands. See [drawTrigger].
	Hovered bool
	Pressed bool

	// Surface is the opaque fill the trigger stands on. A switched-off
	// trigger fades toward it at the platform's measured coverage, and the
	// coverages drawn over the faded fill are flattened onto the answer. The
	// zero value — no colour — is the window's own plane. See
	// [FieldProps.Surface].
	Surface color.NRGBA

	// Drop is the side the open menu takes first when the room cannot hold
	// it where it wants to stand — over the trigger. The zero value is
	// [DropDown]. See [Drop].
	Drop Drop

	// MaxHeight is the tallest the caller would like the open menu's plane to
	// be; above it the rows scroll inside the cap. It is a preference the
	// available room may TIGHTEN and can never loosen: the menu is capped to
	// the smaller of this and what AvailableRoom leaves on the side the menu
	// floats on. The zero value states no preference, and the room alone
	// decides. See [MenuProps.MaxHeight].
	MaxHeight unit.Dp

	// AvailableRoom, if non-nil, reports the room the open menu has to stand
	// in: the pixels between the trigger's top edge and the top of the
	// container the field was laid out in, and between its bottom edge and
	// that container's bottom. It is asked on every frame the menu stands,
	// before the menu is laid out. The menu stands over the trigger, so the
	// room it has is both of those plus the trigger's own height.
	//
	// The zero value is nil, and a menu with no room reported is bounded by
	// the window alone — what a floating surface is bounded by when nobody
	// says otherwise. See [FieldProps.AvailableRoom].
	AvailableRoom func(gtx layout.Context) (above, below int)

	// Placeholder is the wording the trigger shows in place of a value while
	// the field holds none — Selected naming no option. See
	// [FieldProps.Placeholder].
	Placeholder string

	// NoOptions is the wording the trigger shows when there is nothing to
	// pick at all. See [FieldProps.NoOptions].
	NoOptions string
}

// FieldProps configures a [Field] instance.
type FieldProps struct {
	// Description is the screen-reader label.
	Description string

	// Options is the list of selectable items.
	Options []string

	// Selected is the initial selected index established on subscribe.
	Selected int

	// Drop is the side the open menu PREFERS to spend its overflow on, copied
	// straight into [FieldState.Drop] on every frame. The zero value is
	// [DropDown]. The menu itself stands over the trigger with the held row on
	// the trigger's label; Drop settles only where a menu the reported room
	// cannot hold is anchored. Either way the field reports its trigger alone
	// and an open one is placed where a closed one is. See [Drop].
	Drop Drop

	// MaxHeight is the tallest the caller would like the open menu's plane to
	// be; above it the rows scroll inside the cap and the selected row is kept
	// in view. It is a PREFERENCE: the available room may tighten it and can
	// never loosen it, so a caller that states 320 dp where only 120 dp stands
	// above the trigger gets 120. The zero value states no preference, and a
	// field that has been told its room is capped by the room alone.
	//
	// A caller that reports no room and holds a catalogue rather than a
	// handful wants a number here: uncapped and unbounded, the far end of the
	// list is drawn past the bottom of the window, where nothing operates it.
	// See [MenuProps.MaxHeight] for what the cap trades.
	MaxHeight unit.Dp

	// AvailableRoom, if non-nil, is asked on every frame the menu stands for
	// the room it has to stand in — the pixels above the trigger's top edge
	// and below its bottom edge, inside the container that laid the field out
	// — and it settles both questions a floating surface has: where the menu
	// lands, and how tall it may be. The menu stands over the trigger, so the
	// room it has is those two plus the trigger's own height; a menu that
	// cannot stand there whole is pushed into the room, and one taller than
	// the whole room is capped to it and scrolls inside that cap.
	//
	// The container is asked because a component cannot see past its own box:
	// Gio gives a component its constraints and no readback of the transform or
	// of the clip its ancestors imposed, so where the trigger stands inside
	// the dialog, the column or the window around it is knowable only where
	// that box was laid out. The container measures it; it is not a number
	// anybody picks.
	//
	// Nil leaves the menu bounded by the window alone.
	AvailableRoom func(gtx layout.Context) (above, below int)

	// Placeholder is what the trigger says while the field holds no value:
	// the prompt that stands where the chosen option will, drawn in the
	// prompt's own foreground so an unanswered field cannot read as an answered
	// one.
	//
	// The wording is the caller's because it names the caller's subject —
	// "Choose model…" belongs to the app that has models. The empty string
	// draws an empty trigger, which is what a field with no prompt to give
	// has always drawn.
	Placeholder string

	// NoOptions is what the trigger says when there is nothing to pick at
	// all, which is a different sentence from Placeholder: one asks the
	// reader to choose and the other reports that there is no choice to
	// make. A field offered an empty option list draws it, in the same
	// prompt foreground, and opens no menu.
	NoOptions string

	// Disabled, if non-nil, disables the field when it emits true.
	Disabled rx.Observable[bool]

	// OnSelect is called with the newly selected index on every selection.
	// This is the FRP path. The gtx argument is the layout.Context active on
	// the frame the selection is processed in, so a consumer may emit
	// mvu.MessageOp{Message: …}.Add(gtx.Ops) from inside it.
	OnSelect func(gtx layout.Context, index int)

	// Message, if non-nil, is emitted as mvu.MessageOp into the frame's ops on
	// every selection — the MVU path, where OnSelect is the FRP one.
	Message any

	// Surface is the opaque fill the field stands on. The platform's control
	// text, its overlays and the coverage a switched-off control fades by all
	// carry a coverage rather than a colour, so what they land as depends on
	// what is beneath: a caller that put the field on a card, a selected row
	// or a coloured fill says so here. The zero value — no colour — is the
	// window's own plane.
	Surface color.NRGBA

	// Shaper is an explicit per-instance override of the text shaper. Leave it
	// nil in normal use: the field then shapes its text with the theme's
	// shaper (Typography.Shaper()), which is built once for the process and
	// shared by every component reading that typography — the cache lives
	// behind the Typography value, so it survives the copy this component's
	// map function makes of it. Set it only when this field must shape with a
	// different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays the layout
	// tree out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// Field returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes: the form variant's picker —
// the flat trigger bar and, while it is open, the [Menu] it floats against
// itself, beneath by default and above under [DropUp]. Interaction state
// (open/closed, selected index, focus) lives in the rx.Defer scope and
// persists across emissions.
//
// A trigger that must stand under a floating menu instead — one placed against
// the window by patterns/popover — is [Toolbar]; see the package doc.
//
// Both integration paths are supported:
//   - FRP: set FieldProps.OnSelect.
//   - MVU: set FieldProps.Message; the field emits mvu.MessageOp on selection.
//
// # Leaving without choosing
//
// The open menu is a transient overlay and owns the ways out of one: a press
// landing anywhere but on the field, a scroll that carries the trigger, and
// Escape. All three are the field's while the menu stands and none is the
// field's while it is closed, so a dialog hosting the field keeps its own
// Escape until the moment there is a menu to spend it on. Opening the menu
// also gives the trigger the keyboard, which is what a key is bound to and
// what the ring the trigger then wears is saying. See dismissed.
//
// Keyboard reach through the open menu is [Menu]'s, and its doc states what
// each of the two arrangements — per-row tags uncapped, the list's own tag
// under a cap — reaches.
func Field(th rx.Observable[theme.Theme], props FieldProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}

	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Platform, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					body:     typ.BodyLarge,
					spacing:  n.Third,
					radius:   n.Fourth,
					density:  n.Fifth,
					shaper:   typ.Shaper(),
				}
			},
		)
	})

	inputs := rx.CombineLatest2(resolved, disabled)

	return rx.Defer(func() rx.Observable[layout.Widget] {
		var trigger widget.Clickable
		optClicks := make([]widget.Clickable, len(props.Options))
		var open bool
		selected := props.Selected
		rows := list.NewState()
		rows.Select(selected)
		rows.Reveal(selected)
		// The absorber that notices a press landing anywhere but on this
		// field, which is one of the two ways an open menu leaves.
		var outside int

		return rx.Map(inputs, func(next rx.Tuple2[resolvedTokens, bool]) layout.Widget {
			tok, dis := next.First, next.Second

			// Props.Shaper is an explicit override; the theme's shaper is
			// the default.
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}

			return func(gtx layout.Context) layout.Dimensions {
				if dis {
					gtx = gtx.Disabled()
				}

				// Dismissal first, so a press that leaves the menu cannot
				// also be read as a press that opens it.
				if open && dismissed(gtx, &outside, &trigger, rows.Focus()) {
					open = false
				}
				for trigger.Clicked(gtx) {
					open = !open
					if open {
						// An open menu holds the keyboard, which is both
						// what the platform does and what makes Escape
						// operable: a key filter is bound to a focus, and
						// a pointer press moves focus nowhere on its own.
						// The ring the trigger then wears is the truth —
						// the control is the one the keys are going to.
						gtx.Execute(key.FocusCmd{Tag: &trigger})
						// A menu that opens showing the top of a catalogue
						// hides the answer the field is already holding.
						rows.Reveal(selected)
					}
				}
				for i := range optClicks {
					for optClicks[i].Clicked(gtx) {
						selected = i
						open = false
						dispatch(gtx, props.OnSelect, props.Message, i)
					}
				}
				if rows.Selected() != selected {
					rows.Select(selected)
				}

				foc := !dis && gtx.Focused(&trigger)

				dims := layoutFieldLive(gtx, shaper, &trigger, optClicks, rows, &outside, tok, props.Description, FieldState{
					Open:          open,
					Focused:       foc,
					Disabled:      dis,
					Selected:      selected,
					Options:       props.Options,
					Drop:          props.Drop,
					MaxHeight:     props.MaxHeight,
					AvailableRoom: props.AvailableRoom,
					Placeholder:   props.Placeholder,
					NoOptions:     props.NoOptions,
					Surface:       props.Surface,
				})
				if moved := rows.Selected(); open && moved >= 0 && moved != selected {
					selected = moved
					dispatch(gtx, props.OnSelect, props.Message, moved)
				}
				return dims
			}
		})
	})
}

// dismissed reports whether this frame carried one of the three events that
// close an open menu without choosing from it, and drains them all either way.
//
// A PRESS LANDING ELSEWHERE. The absorber under the open field catches it,
// and catching is the whole of what it does: while a menu stands, the next
// press anywhere is spent on putting it away, not on whatever it landed on.
// That is what a transient overlay is — the press that dismisses it is not
// also a press on the dialog behind it — and it is why the absorber is
// registered UNDER the trigger and the rows, which answer presses inside
// their own bounds first.
//
// A SCROLL LANDING ELSEWHERE. The menu is a floating surface and goes with
// its trigger: a trigger carried out of view by the scroller it stands in
// would leave the menu standing over a window the trigger has left. The same
// absorber watches the wheel for that, and declares NO scroll range, so it
// takes none of the distance: the scroller still moves and the menu leaves
// on the same frame. A capped menu's own rows are hit first and consume the
// wheel there, so scrolling inside the menu is not scrolling past it.
//
// ESCAPE. Bound to the tags the open field can hold the keyboard through —
// the trigger, and the list's own tag once a capped menu has it — and drained
// here, in the field's own layout, which runs while a surrounding dialog is
// laying its content out and therefore before that dialog asks for the same
// key. A key event is delivered once, to the first handler that asks for it,
// so the field asking takes it: Escape puts the menu away and the dialog
// around it stays open. Closed, the field registers no filter at all and the
// dialog's Escape is its own again.
func dismissed(gtx layout.Context, outside *int, keys ...event.Tag) bool {
	leave := false
	for {
		e, ok := gtx.Event(pointer.Filter{Target: outside, Kinds: pointer.Press | pointer.Scroll})
		if !ok {
			break
		}
		if pe, ok := e.(pointer.Event); ok && (pe.Kind == pointer.Press || pe.Kind == pointer.Scroll) {
			leave = true
		}
	}
	for _, tag := range keys {
		for {
			e, ok := gtx.Event(key.Filter{Focus: tag, Name: key.NameEscape})
			if !ok {
				break
			}
			if ke, ok := e.(key.Event); ok && ke.State == key.Press {
				leave = true
			}
		}
	}
	return leave
}

// RenderField produces a layout.Widget for the form variant's picker in an
// explicit visual state, without any event processing or rx machinery.
// Intended for golden-image testing and static demonstrations; production code
// should use [Field], which reads both of the parameters below off the theme.
//
// body is the BodyLarge role's whole text style — typeface, weight, size and
// line height all reach the shaper — and d is the density the trigger and the
// option rows draw at. Pass tokens.DefaultTypography.BodyLarge and
// tokens.Comfortable for the default desktop look.
func RenderField(
	shaper *text.Shaper,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s FieldState,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, body: body, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawField(gtx, shaper, tok, s)
	}
}

// layoutFieldLive lays out the interactive field with Clickable hit areas.
func layoutFieldLive(gtx layout.Context, shaper *text.Shaper, trigger *widget.Clickable, optClicks []widget.Clickable, rows *list.State, outside *int, tok resolvedTokens, desc string, s FieldState) layout.Dimensions {
	// The side and the cap are settled before anything is drawn, because the
	// menu is laid out against the side it lands on and the cap is the height
	// that side leaves.
	offsetY, capPx := fitMenu(gtx, shaper, tok, s)
	marked := s
	// The trigger's own pointer state, read off the clickable that covers
	// exactly the drawn bar. The menu's rows carry theirs separately.
	marked.Hovered = trigger.Hovered()
	marked.Pressed = trigger.Pressed()

	// The trigger's pointer area is the drawn bar: a control's target is the
	// control. The menu's rows are their own targets — see layoutMenuLive.
	triggerMacro := op.Record(gtx.Ops)
	triggerDims := trigger.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.Button.Add(gtx.Ops)
		if desc != "" {
			semantic.DescriptionOp(desc).Add(gtx.Ops)
		}
		return drawTrigger(gtx, shaper, tok, marked)
	})
	triggerCall := triggerMacro.Stop()

	if !s.Open || len(s.Options) == 0 {
		triggerCall.Add(gtx.Ops)
		return triggerDims
	}

	menuMacro := op.Record(gtx.Ops)
	menuDims := layoutMenuLive(gtx, shaper, optClicks, rows, tok, MenuState{
		Options:  s.Options,
		Selected: s.Selected,
		Hovered:  hoveredRow(optClicks),
	}, capPx)
	menuCall := menuMacro.Stop()

	// The outside-press absorber, registered before the trigger and the rows
	// so that both win for presses inside their own bounds. A field cannot
	// see the window from inside its own box, so the area simply reaches
	// further than any display in every direction and is clipped by whatever
	// the field is standing in.
	m := gtx.Dp(dismissReach)
	area := clip.Rect{
		Min: image.Pt(-m, -m),
		Max: image.Pt(gtx.Constraints.Max.X+m, gtx.Constraints.Max.Y+m),
	}.Push(gtx.Ops)
	event.Op(gtx.Ops, outside)
	area.Pop()

	return floatMenu(gtx, offsetY, tok, triggerCall, triggerDims, menuCall, menuDims)
}

// alignOverTrigger is where the menu's plane starts, measured from the
// trigger's own top edge, for the menu to stand OVER the trigger with the row
// the picker is holding on the trigger's label: the rows above it are laid out
// above the trigger, the rows below it below, and the held row covers the
// trigger itself.
//
// That is this platform's pop-up button. The control shows a value; pressing
// it opens the catalogue around that value rather than beside it, so the thing
// the reader was looking at does not move and the pointer is already on it.
//
// The held row's own box is centred on the trigger's — both are set in the
// same role at the same size, so their labels meet when their boxes do, and
// the trigger's own label is centred in the trigger the same way
// ([drawTrigger]). A picker holding nothing has no row to align and puts the
// top of the menu on the top of the trigger.
//
// heights are the rows' own heights in order, triggerH the trigger's.
func alignOverTrigger(heights []int, selected, triggerH int) int {
	if selected < 0 || selected >= len(heights) {
		return 0
	}
	before := 0
	for _, h := range heights[:selected] {
		before += h
	}
	return (triggerH-heights[selected])/2 - before
}

// fitMenu settles the two questions the available room answers before an open
// menu is laid out: where the menu's plane starts against the trigger's top
// edge, and how tall that plane may be. A closed field, or one with nothing to
// pick, is asked nothing.
//
// A MENU THAT DRAWS WHOLE stands over the trigger, on the held row — see
// [alignOverTrigger]. The room the caller reports is what may move it: the
// menu is pushed the least distance that brings it wholly inside the room, so
// a field near the top of its container has its menu pushed down and one near
// the foot has it pushed up. Nothing is moved that already fits.
//
// A CAPPED MENU is a viewport, and the rows move under it: there is no row to
// pin to the trigger, because which row stands where is the scroller's answer
// and it changes as the reader scrolls. So a capped plane is placed by its
// room instead — [FieldState.Drop] says which end of the room it takes,
// DropUp the room above the trigger and DropDown the room below — and it
// scrolls inside that cap, which is what the Placement rule asks of a surface
// taller than the room it wins. With no room reported there is no room to take
// an end of, so the cap is anchored to the trigger itself: DropDown begins at
// the trigger's top edge and DropUp ends at its foot.
//
// [FieldState.MaxHeight] tightens the cap and can never loosen it: a menu the
// caller capped and the room capped again takes the smaller of the two. A cap
// no smaller than the menu's own height is no cap at all, so a menu with room
// to spare stacks its rows plainly and draws whole.
//
// A cap ends on a row's edge — see [wholeRows].
func fitMenu(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) (offsetY, capPx int) {
	if !s.Open || len(s.Options) == 0 {
		return 0, capPixels(gtx, s.MaxHeight)
	}

	// What the menu would draw uncapped, row by row, so a cap can end on a
	// row's edge and the held row can be found.
	heights := rowHeights(gtx, shaper, tok, MenuState{Options: s.Options, Selected: s.Selected})
	uncapped := 0
	for _, h := range heights {
		uncapped += h
	}
	triggerH := gtx.Dp(unit.Dp(tok.density.ControlHeight))

	// The room, in the trigger's own coordinates: the trigger's top edge is
	// zero, so the room runs from minus what stands above it to the trigger's
	// height plus what stands below. A field told nothing is bounded by the
	// window, and the trigger's own edges are all this arithmetic knows.
	top, bottom := 0, triggerH
	if s.AvailableRoom != nil {
		above, below := s.AvailableRoom(gtx)
		top, bottom = -above, triggerH+below
	}

	height := uncapped
	if prefer := capPixels(gtx, s.MaxHeight); prefer > 0 && prefer < height {
		height = wholeRows(heights, prefer)
	}
	if s.AvailableRoom != nil && height > bottom-top {
		height = wholeRows(heights, bottom-top)
	}

	if height >= uncapped {
		want := alignOverTrigger(heights, s.Selected, triggerH)
		if s.AvailableRoom != nil {
			if want+uncapped > bottom {
				want = bottom - uncapped
			}
			if want < top {
				want = top
			}
		}
		return want, 0
	}

	if s.Drop == DropUp {
		if s.AvailableRoom != nil {
			return top, height
		}
		return triggerH - height, height
	}
	if s.AvailableRoom != nil {
		return bottom - height, height
	}
	return 0, height
}

// wholeRows is the tallest plane no taller than limit that ends on a row's
// edge. A cap taken raw stops through the letters of whatever row the room ran
// out inside, which reads as a drawing fault rather than as more rows below. A
// limit too small for even one row keeps the limit, since the limit is the
// bound; and a limit that leaves nothing is still a bound and not the absence
// of one, so the plane is capped to a sliver rather than uncapped.
func wholeRows(heights []int, limit int) int {
	whole := 0
	for _, h := range heights {
		if whole+h > limit {
			break
		}
		whole += h
	}
	if whole > 0 {
		return whole
	}
	if limit < 1 {
		return 1
	}
	return limit
}

// floatMenu draws the recorded trigger where the caller put it, floats the
// recorded menu against it at the offset fitMenu settled — over the trigger,
// on the held row, unless the room pushed it — and reports the TRIGGER, which
// is the whole of what the field measures whether its menu stands or not.
//
// The menu and its plane go through op.Defer, so the open plane paints and
// hit-tests above every sibling the window lays out after the field's slot,
// bounded by the window rather than by whatever clipped the trigger.
// patterns/popover's package doc states the idiom and what deferral keeps and
// drops. A floating surface takes no room from the container its anchor
// stands in, so the box the field reports is the trigger's either way and an
// open field is placed exactly where a closed one is.
//
// The plane is the FIELD's to draw and not the menu's: [Menu] handed to a
// pattern is circled by that pattern's own surface and would wear two corners
// and two shadows. What the field adds around the recorded rows is the corner,
// the shadow the level is told by, and the edge — see [planeShape].
func floatMenu(gtx layout.Context, offsetY int, tok resolvedTokens, trigger op.CallOp, triggerDims layout.Dimensions, menu op.CallOp, menuDims layout.Dimensions) layout.Dimensions {
	trigger.Add(gtx.Ops)
	floating := op.Record(gtx.Ops)
	menuOff := op.Offset(image.Pt(0, offsetY)).Push(gtx.Ops)
	planeShape(gtx, menuDims.Size, tok.platform, menu)
	menuOff.Pop()
	op.Defer(gtx.Ops, floating.Stop())
	return triggerDims
}

// planeRadius is the corner the open menu's plane is cut to.
//
// NOT MEASURED: no stored capture in reference/macos holds an open menu, so
// there is no reading of a menu's own corner to take. It is the corner the one
// pill the reference does measure carries — patterns/sidebar's selected row,
// cornered at 8 — and the capture that would settle a menu's own is on the
// reference's list. The rows' own pill takes the same corner, so the plane and
// what stands inside it are cut alike.
const planeRadius = unit.Dp(8)

// planeShape draws the open menu's plane and the rows on it: the shadow that
// says the surface is floating, the rows clipped to the plane's own corner,
// and the edge that says where the plane ends.
//
// THE SHADOW is what tells this level. The Level entry gives a floating
// surface the window background under the platform's shadow, and the window
// background is what the rows themselves fill: nothing but the shadow tells
// the menu's fill from the window's behind it, which is how this platform
// tells them apart — see [control.DrawFloatingShadow] and
// PlatformColors.FloatingShadow.
//
// THE EDGE is the platform's seam flattened over the menu's own plane, drawn
// INSIDE the box the menu reported so the edge costs the plane no height and
// the plane is what lies under it. Without it the plane is separated from what
// it covers by fill alone, and the fill is not always a separation — a menu
// standing on a pane the platform fills the same way is a colour the eye
// cannot find, and text on one side of it running into text on the other
// reads as corruption rather than as two surfaces. The geometry is
// patterns/popover's, because they are the same surface.
//
// The edge is laid ON the plane's outline rather than beside it: a stroke of
// twice its width, centred on that outline and clipped to it, puts every pixel
// of the line inside the box where a stroke of its own width would fall half
// outside. Drawn that way both of its sides follow the corner, which four
// rectangles could not — the same idiom the trigger's edge takes.
func planeShape(gtx layout.Context, size image.Point, p tokens.PlatformColors, rows op.CallOp) {
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	r := gtx.Dp(planeRadius)
	if half := min(size.X, size.Y) / 2; r > half {
		r = half
	}
	plane := clip.RRect{Rect: image.Rectangle{Max: size}, NE: r, NW: r, SE: r, SW: r}

	control.DrawFloatingShadow(gtx, image.Rectangle{Max: size}, r, p)

	area := plane.Push(gtx.Ops)
	rows.Add(gtx.Ops)
	area.Pop()

	w := gtx.Dp(edgeDp)
	if w < 1 {
		w = 1
	}
	border := vgcolor.Flatten(p.Separator, p.WindowBackground)
	path := plane.Path(gtx.Ops)
	edgeArea := plane.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, border, clip.Stroke{Path: path, Width: float32(2 * w)}.Op())
	edgeArea.Pop()
}

// drawField renders the static field — the trigger and, when open, the menu
// it floats — for golden-image testing.
func drawField(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	offsetY, capPx := fitMenu(gtx, shaper, tok, s)
	marked := s

	triggerMacro := op.Record(gtx.Ops)
	triggerDims := drawTrigger(gtx, shaper, tok, marked)
	triggerCall := triggerMacro.Stop()

	if !s.Open || len(s.Options) == 0 {
		triggerCall.Add(gtx.Ops)
		return triggerDims
	}

	menuMacro := op.Record(gtx.Ops)
	menuDims := drawMenu(gtx, shaper, tok, MenuState{
		Options:  s.Options,
		Selected: s.Selected,
	}, capPx)
	menuCall := menuMacro.Stop()

	return floatMenu(gtx, offsetY, tok, triggerCall, triggerDims, menuCall, menuDims)
}

// drawTrigger renders the field trigger bar (the closed face).
//
// The trigger is the platform's pop-up button, and it is measured as that
// control rather than as the text field beside it: MEASURED,
// save-dialog-{light,dark}.png at 1x, the "File Format:" pop-up is 24 px tall
// — the control height, not the field's 28 — its fill runs x 264–451 with no
// edge column of any kind, and both of its insets are spent from that fill's
// own edge: [control.PopupLeadDp] for the label and
// [control.PopupMarkTrailDp] for the mark. The rows of the menu it opens are
// not pop-ups and take their own columns — see picker's rowColumns.
func drawTrigger(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	// No edge column stands between the fill and the surface, so there is no
	// inner edge for an inset to start from and both are spent from the
	// fill's own. That holds in every state: focus is a halo laid on the
	// shape's outline and moves nothing the trigger draws or reports.
	lead := gtx.Dp(control.PopupLeadDp)
	trail := gtx.Dp(control.PopupMarkTrailDp)
	rad := gtx.Dp(unit.Dp(tok.radius.Md))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)
	fieldW := gtx.Constraints.Max.X
	markW := gtx.Dp(control.MarkWDp)

	// A trigger says one of three things, and which foreground it says it in
	// is the difference between a value and a prompt: an unanswered field
	// drawn in the body foreground reads as answered. Two prompts, because
	// "choose one" and "there is nothing to choose" are different sentences
	// and only the caller knows either.
	label := ""
	prompt := true
	switch {
	case len(s.Options) == 0:
		label = s.NoOptions
	case s.Selected >= 0 && s.Selected < len(s.Options):
		label, prompt = s.Options[s.Selected], false
	default:
		label = s.Placeholder
	}

	// The trigger is the platform's ordinary button, so its fill is the push
	// button's own measured fill, and every name the trigger draws over that
	// fill is flattened onto it.
	//
	// Under the pointer it takes the platform's hover overlay and held down
	// its press overlay, a press winning over a hover: the two are one answer
	// and not two laid on each other. MEASURED,
	// control-hover-{light,dark}.png — the Finder toolbar's VIEW POP-UP, the
	// control drawing this very mark, under the pointer: #f2f2f2 on the
	// #ffffff band light and #384146 on the #242d32 band dark, which is the
	// hover overlay over what the control stands on. No capture holds a
	// pressed pop-up, so the press is the push button's, MEASURED off
	// control-pressed-{light,dark}.png: #d5d5d5 and #474d52 over the push
	// button's own fill. The pressed pop-up is on the capture list.
	//
	// WHILE ITS MENU STANDS the trigger takes the same held drawing. The
	// press that opened the menu has not been answered until the menu closes,
	// and a trigger drawn at rest under an open menu reads as a control
	// nothing is happening to. No stored capture holds a pop-up with its menu
	// open either, so what is drawn is the one held state the reference does
	// measure, and that capture is on the reference's list with the pressed
	// pop-up.
	//
	// A disabled trigger fades toward the surface it stands on at the
	// platform's measured disabled coverage, and every name drawn over it is
	// flattened onto the faded fill. Which surface that is, is the caller's
	// to state ([FieldState.Surface]); an unstated one is the window's plane.
	bg := tok.platform.PushButtonFill
	switch {
	case s.Disabled:
		bg = control.Faded(bg, surface.Or(s.Surface, tok.platform.WindowBackground))
	case s.Pressed, s.Open:
		bg = vgcolor.Flatten(tok.platform.PressOverlay, bg)
	case s.Hovered:
		bg = vgcolor.Flatten(tok.platform.HoverOverlay, bg)
	}

	textCol := vgcolor.Flatten(tok.platform.ControlText, bg)
	if prompt {
		textCol = control.Placeholder(tok.platform, bg)
	}
	if s.Disabled {
		textCol = vgcolor.Flatten(tok.platform.DisabledControlText, bg)
	}

	// The mark stands in the trailing inset's slot, and the room held
	// between it and the value is the trigger's own gap rather than one of
	// its two ends: it is what stops a long value running into the mark.
	gap := gtx.Dp(unit.Dp(tok.spacing.S3))
	innerW := fieldW - lead - gap - markW - trail
	if innerW < 1 {
		innerW = 1
	}
	// The pop-up's height is the control height and nothing else, so the line
	// box the label is set in is capped to it: MEASURED, controls.md's small
	// control rows give the platform 19 px for a Compact control, where
	// BodyLarge's line box is 24 dp at every density. The role's SIZE does not
	// move — its cap band measures 12 px and stands inside 19 with room to
	// spare — so what is cut is the leading the line box carries around that
	// band, and the trigger clips what it draws to its own shape so a Compact
	// control cannot paint a row of text outside itself.
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	innerGtx := gtx
	innerGtx.Constraints = layout.Constraints{
		Min: image.Pt(0, 0),
		Max: image.Pt(innerW, minH),
	}

	mTextCol := op.Record(gtx.Ops)
	paint.ColorOp{Color: textCol}.Add(gtx.Ops)
	textMat := mTextCol.Stop()

	mLabel := op.Record(gtx.Ops)
	labelDims := typeset.Layout(innerGtx, shaper, wl, f, textSize, label, textMat)
	labelCall := mLabel.Stop()

	// The pop-up's height is the control height and nothing else: MEASURED,
	// 24 px in both appearances of the save dialog, which is the same number
	// the push button beside it draws and the same the chrome variant draws.
	// A pop-up is not sized by the line box it carries — the text field above
	// it in that capture is the control that is, and it measures 27.
	triggerH := minH
	triggerSize := image.Pt(fieldW, triggerH)

	// The fill meets the surface directly. MEASURED, the same capture: a run
	// down x=350 through the "File Format:" pop-up gives the fill's own
	// #ececec light and #333a3f dark from the control's first row to its
	// last, with no darker column at either end — the platform draws this
	// control's fill as its whole shape and puts no seam around it.
	//
	// Focus adds nothing to that: the trigger keeps the fill and the
	// edgeless shape the platform draws it with, and wears the library's one
	// halo on that outline while it holds the keyboard.
	rrectOuter := clip.RRect{Rect: image.Rectangle{Max: triggerSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, bg, rrectOuter.Op(gtx.Ops))
	if s.Focused {
		standsOn := surface.Or(s.Surface, tok.platform.WindowBackground)
		focus.Halo(gtx, image.Rectangle{Max: triggerSize}, rad,
			focus.Ring(tok.platform, standsOn), focus.Ring(tok.platform, bg))
	}

	// Text label: vertically centred, and clipped to the control's own shape
	// so a line box taller than the control it stands in is cut by the
	// control rather than drawn past it.
	offY := (triggerH - labelDims.Size.Y) / 2
	area := rrectOuter.Push(gtx.Ops)
	st := op.Offset(image.Pt(lead, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()
	area.Pop()

	// The mark: the pair's last column stands [control.PopupMarkTrailDp] clear
	// of the fill's trailing edge, which is the trailing inset this control
	// spends.
	//
	// It draws the platform's own colour for a control's own marks: MEASURED,
	// the pair's fully covered pixels read 36 light and (224,225,226) dark
	// against fills of #ececec and #333a3f, which is ControlText's 216 of 255
	// flattened onto each to the byte. The secondary label's coverage would
	// land at 118 and 163 — this mark is not drawn in it.
	markCol := vgcolor.Flatten(tok.platform.ControlText, bg)
	if s.Disabled {
		markCol = vgcolor.Flatten(tok.platform.DisabledControlText, bg)
	}
	control.DrawMark(gtx, image.Rect(fieldW-trail-markW, 0, fieldW-trail, triggerH), markCol)

	return layout.Dimensions{Size: triggerSize}
}
