package picker

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
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
	"github.com/vibrantgio/components/list"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// The form trigger's mark is the platform's pop-up mark: two chevrons stacked
// point to point, the upper pointing up and the lower down. Its proportions
// are MEASURED off save-dialog-{light,dark}.png at 1x, both appearances
// agreeing to the pixel — the "File Format:" pop-up runs y 336–359, 24 px
// tall, and the pair it draws spans x 435–442 and y 343–353: eight columns
// wide, the upper chevron y 343–347 and the lower y 349–353, five rows each,
// with one clear row between them.
//
// The Finder toolbar draws the same glyph at the same size in a control 36 px
// tall (finder-window-light.png, x 726–733, upper y 21–25, lower y 27–31), so
// the platform sizes this mark by its point size and not by the control. The
// ratios below are the dialog reading, which is the control this trigger is
// drawn as; expressing them as ratios of the control's height is what keeps
// the proportion at a density the platform has not been captured at.
//
//	markWidthRatio  the pair's column, 8 of the control's 24
//	markAspect      one chevron's height, 5 of the pair's own 8
//	markGapRatio    the clear row between the two, 1 of the control's 24
const (
	markWidthRatio = 8.0 / 24.0
	markAspect     = 5.0 / 8.0
	markGapRatio   = 1.0 / 24.0
)

// markStroke is the pair's line weight. MEASURED off the same capture: an arm
// crossing a row covers about 2.1 columns — the light capture's row y=346
// reads 108, 37 and 156 against a 236 fill and a 36 foreground, which is
// 0.64 + 0.99 + 0.40 of a column — and the arm runs at 45°, so perpendicular
// it is 2.1 × sin 45° ≈ 1.5 px. The same weight the chrome variant's single
// chevron is drawn at.
const markStroke = unit.Dp(1.5)

// edgeDp is the hairline the dropped menu's plane is drawn with, and so the
// distance from the plane's outer edge to its inner one. The trigger draws no
// edge at all — see [drawTrigger].
const edgeDp = unit.Dp(1)

// dismissReach is how far the outside-press absorber reaches beyond the box
// the open field was offered, on every side. A component cannot see the
// window from inside its own layout, so the reach is simply larger than any
// display and whatever the field stands in clips it back.
const dismissReach = unit.Dp(8192)

// Drop is the side an open [Field] PREFERS to float its menu on.
//
// It answers a question only the caller can see: whether the room beneath the
// trigger is the room the menu should take. A field at the foot of a dialog
// has none — a menu dropped there would stand off the bottom edge — so that
// caller says [DropUp] and the menu floats above the trigger instead, over
// whatever the window laid out before it.
//
// It is a preference and not an instruction. A caller that reports the
// available room ([FieldProps.AvailableRoom]) is telling the field how much
// there is on each side, and a menu that cannot be seen whole on the preferred
// side while the other side holds more of it flips to that other side. With
// no room reported the preference is simply obeyed.
//
// Either way the open field reports its TRIGGER and nothing else. The menu is
// a floating surface deferred to the end of the frame, and a floating surface
// asks the container its anchor stands in for no room: an open field is placed
// exactly where a closed one is, and the direction changes what the menu
// covers, never the box the field reports.
//
// The trigger's mark says nothing about it. The platform's pop-up mark is a
// pair of chevrons pointing opposite ways — it says the choice can move
// either way and cannot say a direction — so a field that drops upwards draws
// the same trigger as one that drops down, and the direction is read off the
// menu itself. See [drawMark].
type Drop uint8

const (
	// DropDown is the zero value: the menu floats directly beneath the
	// trigger, which is what a form's select does.
	DropDown Drop = iota

	// DropUp floats the menu directly above the trigger.
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

	// Drop is the side the open menu floats on. The zero value is
	// [DropDown], beneath the trigger. See [Drop].
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
	// before the menu is laid out.
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

	// Drop is the side the open menu PREFERS to float on, copied straight into
	// [FieldState.Drop] on every frame. The zero value is [DropDown]. A caller
	// with no room beneath the trigger says [DropUp]; either way the field
	// reports its trigger alone and an open one is placed where a closed one
	// is. A field that has been told the available room flips to the other
	// side when this one cannot hold the menu and the other holds more of it.
	// See [Drop].
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
	// — and it settles both questions a floating surface has: which side of
	// the trigger the menu goes on, and how tall it may be. The menu is capped
	// to what the chosen side leaves and scrolls inside that cap, and [Drop]
	// is a preference the room can overrule.
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
	drop, capPx := fitMenu(gtx, shaper, tok, s)
	marked := s
	marked.Drop = drop

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

	return floatMenu(gtx, drop, tok, triggerCall, triggerDims, menuCall, menuDims)
}

// fitMenu settles the two questions the available room answers before an open
// menu is laid out: which side of the trigger it floats on, and how tall its
// plane may be. A closed field, or one with nothing to pick, is asked nothing
// and keeps the caller's own side and preference.
//
// [FieldState.Drop] is a preference and stands unless the room says otherwise.
// The menu FLIPS when the preferred side cannot hold what the menu would draw
// and the other side holds more of it: a surface the reader can see more of is
// worth the side the caller did not ask for, and a surface that fits where it
// was asked to go is not moved for a roomier neighbour. The cap is then the
// chosen side's room, tightened further by [FieldState.MaxHeight] where the
// caller stated one — the room may tighten a preference and never loosen it —
// and a cap no smaller than the menu's own height is no cap at all, so a menu
// with room to spare stacks its rows plainly and draws whole.
//
// With no room reported ([FieldState.AvailableRoom] nil) the caller's side and
// cap are the whole answer and the menu is bounded by the window.
func fitMenu(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) (Drop, int) {
	prefer := capPixels(gtx, s.MaxHeight)
	if !s.Open || len(s.Options) == 0 || s.AvailableRoom == nil {
		return s.Drop, prefer
	}

	// What the menu would draw uncapped, row by row, so the cap can end on a
	// row's edge as well as inside the room.
	heights := rowHeights(gtx, shaper, tok, MenuState{Options: s.Options, Selected: s.Selected})
	uncapped := 0
	for _, h := range heights {
		uncapped += h
	}

	above, below := s.AvailableRoom(gtx)
	drop := s.Drop
	room, other := below, above
	if drop == DropUp {
		room, other = above, below
	}
	want := uncapped
	if prefer > 0 && prefer < want {
		want = prefer
	}
	if room < want && other > room {
		if drop == DropUp {
			drop = DropDown
		} else {
			drop = DropUp
		}
		room = other
	}
	capPx := room
	if prefer > 0 && prefer < capPx {
		capPx = prefer
	}
	// The plane ends on a row's edge: a cap taken raw stops through the
	// letters of whatever row the room ran out inside, which reads as a
	// drawing fault rather than as more rows below. A room too small for even
	// one row keeps the room, since the room is the bound.
	whole := 0
	for _, h := range heights {
		if whole+h > capPx {
			break
		}
		whole += h
	}
	if whole > 0 {
		capPx = whole
	}
	// A side that leaves nothing is still a bound and not the absence of one,
	// so the plane is capped to a sliver rather than uncapped.
	if capPx < 1 {
		capPx = 1
	}
	if capPx >= uncapped {
		return drop, 0
	}
	return drop, capPx
}

// floatMenu draws the recorded trigger where the caller put it, floats the
// recorded menu against it on the side fitMenu chose — directly beneath under
// [DropDown], directly above under [DropUp] — and reports the TRIGGER, which
// is the whole of what the field measures whether its menu stands or not.
//
// The menu and its edge go through op.Defer, so the open plane paints and
// hit-tests above every sibling the window lays out after the field's slot,
// bounded by the window rather than by whatever clipped the trigger.
// patterns/popover's package doc states the idiom and what deferral keeps and
// drops. A floating surface takes no room from the container its anchor
// stands in, so the box the field reports is the trigger's either way and an
// open field is placed exactly where a closed one is.
//
// The menu's plane takes its edge here, in both directions, because the plane
// is the field's to draw: [Menu] handed to a pattern is circled by that
// pattern's own surface and would wear two lines.
func floatMenu(gtx layout.Context, d Drop, tok resolvedTokens, trigger op.CallOp, triggerDims layout.Dimensions, menu op.CallOp, menuDims layout.Dimensions) layout.Dimensions {
	trigger.Add(gtx.Ops)
	menuY := triggerDims.Size.Y
	if d == DropUp {
		menuY = -menuDims.Size.Y
	}
	floating := op.Record(gtx.Ops)
	menuOff := op.Offset(image.Pt(0, menuY)).Push(gtx.Ops)
	menu.Add(gtx.Ops)
	planeEdge(gtx, menuDims.Size, tok.platform)
	menuOff.Pop()
	op.Defer(gtx.Ops, floating.Stop())
	return triggerDims
}

// planeEdge draws the open menu's own edge: the one line that says where the
// transient plane ends.
//
// Without it the plane is separated from what it covers by fill alone, and the
// fill is not always a separation — a menu standing on a pane the platform
// fills the same way is a colour the eye cannot find, and text on one side of
// it running into text on the other reads as corruption rather than as two
// surfaces.
//
// The line is the platform's seam flattened over the menu's own plane — it is
// drawn inside the box, so that plane is what lies under it — one dp wide.
// The geometry is patterns/popover's, because they are the same surface.
//
// It is drawn INSIDE the box the menu reported, on all four sides, so the edge
// costs the plane no height and the two drop directions are one drawing.
func planeEdge(gtx layout.Context, size image.Point, p tokens.PlatformColors) {
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	w := gtx.Dp(edgeDp)
	if w < 1 {
		w = 1
	}
	border := vgcolor.Flatten(p.Separator, p.ControlBackground)
	for _, r := range [...]image.Rectangle{
		{Max: image.Pt(size.X, w)},
		{Min: image.Pt(0, size.Y-w), Max: size},
		{Max: image.Pt(w, size.Y)},
		{Min: image.Pt(size.X-w, 0), Max: size},
	} {
		paint.FillShape(gtx.Ops, border, clip.Rect(r).Op())
	}
}

// drawField renders the static field — the trigger and, when open, the menu
// it floats — for golden-image testing.
func drawField(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	drop, capPx := fitMenu(gtx, shaper, tok, s)
	marked := s
	marked.Drop = drop

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

	return floatMenu(gtx, drop, tok, triggerCall, triggerDims, menuCall, menuDims)
}

// drawTrigger renders the field trigger bar (the closed face).
//
// The trigger is the platform's pop-up button, and it is measured as that
// control rather than as the text field beside it: MEASURED,
// save-dialog-{light,dark}.png at 1x, the "File Format:" pop-up is 24 px tall
// — the control height, not the field's 28 — its fill runs x 264–451 with no
// edge column of any kind, and both of its insets are spent from that fill's
// own edge: [control.PopupLeadDp] for the label and
// [control.PopupMarkTrailDp] for the mark. The rows of the menu it drops are
// not pop-ups and keep the field's insets.
func drawTrigger(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	// No edge column stands between the fill and the surface, so there is no
	// inner edge for an inset to start from and both are spent from the
	// fill's own. That holds in every state: focus draws its ring in the
	// fill's outermost pixels rather than outside them, so nothing moves.
	lead := gtx.Dp(control.PopupLeadDp)
	trail := gtx.Dp(control.PopupMarkTrailDp)
	rad := gtx.Dp(unit.Dp(tok.radius.Md))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X
	markW := gtx.Dp(unit.Dp(tok.density.ControlHeight * markWidthRatio))

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
	// fill is flattened onto it. A disabled trigger keeps the fill: the
	// platform fades the wording and leaves the control.
	bg := tok.platform.PushButtonFill

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
	innerGtx := gtx
	innerGtx.Constraints = layout.Constraints{
		Min: image.Pt(0, 0),
		Max: image.Pt(innerW, gtx.Constraints.Max.Y),
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
	// Focus is the exception, because a focus ring is not an edge the control
	// wears but the one idiom every control in the library wears while it
	// holds the keyboard. It is a band laid ON the shape's outline rather
	// than a shape beneath it: a stroke of twice the ring's width, centred on
	// that outline and clipped to it, puts every pixel inside the box the
	// trigger reports where a stroke of the ring's own width would fall half
	// outside. Drawn that way both of the band's sides follow the corner,
	// which four rectangles could not.
	rrectOuter := clip.RRect{Rect: image.Rectangle{Max: triggerSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, bg, rrectOuter.Op(gtx.Ops))
	if s.Focused {
		ringPx := gtx.Dp(focus.Width)
		edgePath := rrectOuter.Path(gtx.Ops)
		edgeArea := rrectOuter.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, focus.Ring(tok.platform, bg),
			clip.Stroke{Path: edgePath, Width: float32(2 * ringPx)}.Op())
		edgeArea.Pop()
	}

	// Text label: vertically centered.
	offY := (triggerH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(lead, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

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
	drawMark(gtx, image.Rect(fieldW-trail-markW, 0, fieldW-trail, triggerH), tok.density, markCol)

	return layout.Dimensions{Size: triggerSize}
}

// drawMark paints the pop-up mark inside box: two chevrons stacked point to
// point, the upper pointing up and the lower down, hairline strokes spanning
// box horizontally and centred in it vertically.
//
// It is STATIC. The platform's pop-up mark says "this control holds one of
// several values" and never "the menu is open" or "it opens upwards"; a pair
// pointing both ways cannot say a direction, which is the whole reason the
// platform draws a pair here and a single chevron on a pull-down. See [Drop]
// for where the direction is actually settled.
func drawMark(gtx layout.Context, box image.Rectangle, d tokens.Density, col color.NRGBA) {
	stroke := float32(gtx.Dp(markStroke))
	if stroke < 1 {
		stroke = 1
	}
	// The measurements are of COVERAGE — the rows and columns the capture
	// shows covered — and a stroke spreads half its width either side of the
	// line it is drawn on, so box is that covered extent and the path is box
	// inset by half a stroke on every side. The rasterizer puts the half back.
	w := float32(box.Dx())
	coveredH := w * markAspect
	clear := float32(gtx.Dp(unit.Dp(d.ControlHeight * markGapRatio)))
	armH := coveredH - stroke
	sep := clear + stroke

	// Centred in the control, and where the centring lands between two rows it
	// takes the lower one: MEASURED, the pair covers y 343–353 in a control of
	// y 336–359 — eleven rows in twenty-four, seven above them and six below,
	// which is the exact centre of 6.5 rounded up.
	coveredTop := float32(math.Ceil(float64(float32(box.Dy())-(2*coveredH+clear)) / 2))
	top := float32(box.Min.Y) + coveredTop + stroke/2
	x0, x1 := float32(box.Min.X)+stroke/2, float32(box.Max.X)-stroke/2
	mid := (x0 + x1) / 2

	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(x0, top+armH))
	p.LineTo(f32.Pt(mid, top))
	p.LineTo(f32.Pt(x1, top+armH))
	p.MoveTo(f32.Pt(x0, top+armH+sep))
	p.LineTo(f32.Pt(mid, top+2*armH+sep))
	p.LineTo(f32.Pt(x1, top+armH+sep))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: stroke}.Op())
}
