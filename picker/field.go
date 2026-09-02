package picker

import (
	"image"
	"image/color"

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
	"github.com/vibrantgio/components/internal/hit"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// fieldChevron is the box the field trigger's mark is drawn in: a solid
// triangle, the form variant's own mark rather than the chrome variant's
// stroked chevron. It is a fixed size because the trigger's other numbers come
// from the density and the mark is what tells the two variants apart at a
// glance.
const fieldChevron = unit.Dp(16)

// dismissReach is how far the outside-press absorber reaches beyond the box
// the open field was offered, on every side. A component cannot see the
// window from inside its own layout, so the reach is simply larger than any
// display and whatever the field stands in clips it back.
const dismissReach = unit.Dp(8192)

// Drop is the direction an open [Field] stacks its menu in.
//
// It answers a question only the caller can see: whether the room beneath the
// trigger is room the menu may have. A field at the foot of a dialog whose
// action row is drawn after the body has none — a menu dropped there is
// painted over by the row — so that caller says [DropUp] and the menu stands
// above instead.
//
// Either way the open field is the trigger plus the menu, stacked, and the
// widget reports both; what changes is the order. [DropUp] therefore puts the
// TRIGGER at the bottom of the reported box, so a caller placing an upward
// field aligns that box's BOTTOM edge with the row the trigger stands in —
// record the widget, read the height it reports, and offset by it.
//
// It is also what the trigger's own mark says, open or closed: the triangle
// points the way the menu will go. A mark that pointed down over a menu that
// stands above it would be announcing a motion the control does not have.
type Drop uint8

const (
	// DropDown is the zero value: the menu stands directly beneath the
	// trigger, which is what a form's select does.
	DropDown Drop = iota

	// DropUp stands the menu directly above the trigger.
	DropUp
)

// FieldState holds the explicit visual state a static field render draws in.
// All fields default to their zero values (normal, closed, idle, on the window
// ground).
//
// Intended for golden-image testing; production code obtains state from the
// Gio event system via [Field].
type FieldState struct {
	Open     bool
	Focused  bool
	Disabled bool
	Selected int
	Options  []string

	// Drop is which way the open menu stacks. The zero value is [DropDown],
	// beneath the trigger. See [Drop].
	Drop Drop

	// Ground is the elevation storey of the surface hosting the trigger —
	// the local ground its resting border is derived against, in the same
	// vocabulary the host names its own fill (tokens.SurfaceAt). A dialog at
	// tokens.Level2 passes Level2 and the border takes whichever neutral
	// rung clears the floor over that storey. The zero value is
	// tokens.Level0, the window ground. It governs the trigger only: the
	// open menu is its own plane, a level-3 overlay carrying the edge that
	// storey draws (see planeEdge).
	Ground tokens.ElevationLevel

	// MaxHeight caps the open menu's plane; above it the rows scroll inside
	// the cap. The zero value is no cap. See [MenuProps.MaxHeight].
	MaxHeight unit.Dp

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

	// Drop is which way the open menu stacks, copied straight into
	// [FieldState.Drop] on every frame. The zero value is [DropDown]. A caller
	// with no room beneath the trigger says [DropUp] and then places the
	// widget by its bottom edge. See [Drop].
	Drop Drop

	// Ground is the elevation storey of the surface hosting the field, copied
	// straight into [FieldState.Ground] on every frame: the local ground the
	// trigger's resting border is derived against. A container that raises its
	// surface passes its own storey here; the zero value is the window ground.
	// See [FieldState.Ground].
	Ground tokens.ElevationLevel

	// MaxHeight caps the open menu's plane; above it the rows scroll inside
	// the cap and the selected row is kept in view. The zero value is no cap
	// and the menu draws every option. A field whose options are a
	// catalogue rather than a handful wants one: uncapped, the far end of
	// the list is drawn past the bottom of the window, where nothing reaches
	// it. See [MenuProps.MaxHeight] for what the cap trades.
	MaxHeight unit.Dp

	// Placeholder is what the trigger says while the field holds no value:
	// the prompt that stands where the chosen option will, drawn in the
	// prompt's own ink so an unanswered field cannot read as an answered
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
	// prompt ink, and opens no menu.
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
	// A shaper is not safe to use from two goroutines; Gio lays the widget
	// forest out on the one goroutine that runs the event loop, which is what
	// makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper
}

// Field returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes: the form variant's picker —
// the flat trigger bar and, while it is open, the [Menu] it stacks against
// itself, beneath by default and above under [DropUp]. Widget state
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
// The open menu is a transient overlay and owns both ways out of one: a press
// landing anywhere but on the field, and Escape. Both are the field's while
// the menu stands and neither is the field's while it is closed, so a dialog
// hosting the field keeps its own Escape until the moment there is a menu to
// spend it on. Opening the menu also gives the trigger the keyboard, which is
// what a key is bound to and what the ring the trigger then wears is saying.
// See dismissed.
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
			rx.CombineLatest5(t.Color, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.ColorTokens, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					color:   n.First,
					body:    typ.BodyLarge,
					spacing: n.Third,
					radius:  n.Fourth,
					density: n.Fifth,
					shaper:  typ.Shaper(),
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
						// reachable: a key filter is bound to a focus, and
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
					Open:        open,
					Focused:     foc,
					Disabled:    dis,
					Selected:    selected,
					Options:     props.Options,
					Drop:        props.Drop,
					Ground:      props.Ground,
					MaxHeight:   props.MaxHeight,
					Placeholder: props.Placeholder,
					NoOptions:   props.NoOptions,
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

// dismissed reports whether this frame carried one of the two events that
// close an open menu without choosing from it, and drains both either way.
//
// A PRESS LANDING ELSEWHERE. The absorber under the open field catches it,
// and catching is the whole of what it does: while a menu stands, the next
// press anywhere is spent on putting it away, not on whatever it landed on.
// That is what a transient overlay is — the press that dismisses it is not
// also a press on the dialog behind it — and it is why the absorber is
// registered UNDER the trigger and the rows, which answer presses inside
// their own bounds first.
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
		e, ok := gtx.Event(pointer.Filter{Target: outside, Kinds: pointer.Press})
		if !ok {
			break
		}
		if pe, ok := e.(pointer.Event); ok && pe.Kind == pointer.Press {
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
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s FieldState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, body: body, density: d}
	return func(gtx layout.Context) layout.Dimensions {
		return drawField(gtx, shaper, tok, s)
	}
}

// layoutFieldLive lays out the interactive field with Clickable hit areas.
func layoutFieldLive(gtx layout.Context, shaper *text.Shaper, trigger *widget.Clickable, optClicks []widget.Clickable, rows *list.State, outside *int, tok resolvedTokens, desc string, s FieldState) layout.Dimensions {
	// The trigger's pointer area is at least MinHitTarget (44 dp) on each
	// axis, centred on the visual bar: density shrinks the drawn trigger,
	// never the hit target. The menu's rows are not extended — see
	// layoutMenuLive.
	triggerMacro := op.Record(gtx.Ops)
	triggerDims := hit.Extend(gtx, gtx.Dp(unit.Dp(tok.density.MinHitTarget())), trigger.Layout, func(gtx layout.Context) layout.Dimensions {
		semantic.Button.Add(gtx.Ops)
		if desc != "" {
			semantic.DescriptionOp(desc).Add(gtx.Ops)
		}
		return drawTrigger(gtx, shaper, tok, s)
	})
	triggerCall := triggerMacro.Stop()

	if !s.Open || len(s.Options) == 0 {
		triggerCall.Add(gtx.Ops)
		return triggerDims
	}

	menuMacro := op.Record(gtx.Ops)
	menuDims := layoutMenuLive(gtx, shaper, optClicks, rows, tok, MenuState{
		Options:   s.Options,
		Selected:  s.Selected,
		Hovered:   hoveredRow(optClicks),
		MaxHeight: s.MaxHeight,
	})
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

	return stackOpen(gtx, s.Drop, tok, triggerCall, triggerDims, menuCall, menuDims)
}

// stackOpen places the recorded trigger and menu in the [Drop]'s order and
// reports the whole stack, because what was drawn is what a container has to
// make room for. Under [DropUp] the trigger is the LOWER half, which is what
// lets an upward field be placed by the bottom edge of the box it reports.
//
// The menu's plane takes its edge here, in both directions, because the plane
// is the field's to draw: [Menu] handed to a pattern is circled by that
// pattern's own surface and would wear two lines.
func stackOpen(gtx layout.Context, d Drop, tok resolvedTokens, trigger op.CallOp, triggerDims layout.Dimensions, menu op.CallOp, menuDims layout.Dimensions) layout.Dimensions {
	triggerY, menuY := 0, triggerDims.Size.Y
	if d == DropUp {
		triggerY, menuY = menuDims.Size.Y, 0
	}
	off := op.Offset(image.Pt(0, triggerY)).Push(gtx.Ops)
	trigger.Add(gtx.Ops)
	off.Pop()
	off = op.Offset(image.Pt(0, menuY)).Push(gtx.Ops)
	menu.Add(gtx.Ops)
	planeEdge(gtx, menuDims.Size, tok.color)
	off.Pop()
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, triggerDims.Size.Y+menuDims.Size.Y)}
}

// planeEdge draws the open menu's own edge: the one line that says where the
// transient plane ends.
//
// Without it the plane is separated from what it covers by fill alone, and the
// fill is not always a separation — a level-3 menu over a level-2 dialog
// measures 1.03:1 in the light scheme, which is not a seam but a colour the
// eye cannot find, and text on one side of it running into text on the other
// reads as corruption rather than as two surfaces.
//
// The ink and the geometry are patterns/popover's, because they are the same
// surface: the neutral rung that reaches the graphic floor against the plane
// the line circles — here the menu's own level-3 fill, which is the harder of
// the line's two sides — laid one dp wide. The rung is what a named step
// cannot be, since the paired ramps put a fixed step at the same perceptual
// depth in both schemes while the ground under it moves the whole way.
//
// It is drawn INSIDE the box the menu reported, on all four sides, so the edge
// costs the stack no height and the two drop directions are one drawing.
func planeEdge(gtx layout.Context, size image.Point, c tokens.ColorTokens) {
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	w := gtx.Dp(1)
	if w < 1 {
		w = 1
	}
	ink := control.Border(c, tokens.Level3)
	for _, r := range [...]image.Rectangle{
		{Max: image.Pt(size.X, w)},
		{Min: image.Pt(0, size.Y-w), Max: size},
		{Max: image.Pt(w, size.Y)},
		{Min: image.Pt(size.X-w, 0), Max: size},
	} {
		paint.FillShape(gtx.Ops, ink, clip.Rect(r).Op())
	}
}

// drawField renders the static field — the trigger and, when open, the menu
// under it — for golden-image testing.
func drawField(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	triggerMacro := op.Record(gtx.Ops)
	triggerDims := drawTrigger(gtx, shaper, tok, s)
	triggerCall := triggerMacro.Stop()

	if !s.Open || len(s.Options) == 0 {
		triggerCall.Add(gtx.Ops)
		return triggerDims
	}

	menuMacro := op.Record(gtx.Ops)
	menuDims := drawMenu(gtx, shaper, tok, MenuState{
		Options:   s.Options,
		Selected:  s.Selected,
		MaxHeight: s.MaxHeight,
	})
	menuCall := menuMacro.Stop()

	return stackOpen(gtx, s.Drop, tok, triggerCall, triggerDims, menuCall, menuDims)
}

// drawTrigger renders the field trigger bar (the closed face).
func drawTrigger(gtx layout.Context, shaper *text.Shaper, tok resolvedTokens, s FieldState) layout.Dimensions {
	// The trigger follows the text field's sizing rule — height =
	// Density.ControlHeight, vertical padding = Density.PaddingY, horizontal
	// padding a static spacing.S3 (12 dp).
	padH := gtx.Dp(unit.Dp(tok.spacing.S3))
	padV := gtx.Dp(unit.Dp(tok.density.PaddingY))
	rad := gtx.Dp(unit.Dp(tok.radius.Md))
	// Shape with the BodyLarge role's typeface, weight, size and line height.
	f, wl, textSize := bodyLabel(tok)
	minH := gtx.Dp(unit.Dp(tok.density.ControlHeight))
	fieldW := gtx.Constraints.Max.X
	chevronSz := gtx.Dp(fieldChevron)

	// A trigger says one of three things, and which ink it says it in is the
	// difference between a value and a prompt: an unanswered field drawn in
	// the body ink reads as answered. Two prompts, because "choose one" and
	// "there is nothing to choose" are different sentences and only the
	// caller knows either.
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

	textCol := tok.color.Text
	if prompt {
		textCol = control.Placeholder(tok.color)
	}
	if s.Disabled {
		textCol = tokens.Disabled(textCol)
	}

	// Reserve space for chevron: padH on the right side plus chevron width.
	innerW := fieldW - 2*padH - chevronSz - padH
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

	triggerH := labelDims.Size.Y + 2*padV
	if triggerH < minH {
		triggerH = minH
	}
	triggerSize := image.Pt(fieldW, triggerH)

	// The trigger's own fill is the storey it is raised to over the ground it
	// stands on (control.Fill), the same walk the field, the box and the radio
	// take. Disabled fades that fill rather than naming a second one, so the
	// state follows the storey wherever the trigger was put.
	bg := control.Fill(tok.color, s.Ground)
	if s.Disabled {
		bg = tokens.Disabled(bg)
	}
	// Focus promotes the trigger's border to the focus ring, the one idiom
	// every control in the library wears: focus.Ring, the scheme's single
	// focus colour, so promoting the edge changes its hue and not what it has
	// to answer to, and a focused trigger in a dialog draws the same pixel as
	// a focused control on the paper behind it.
	// At rest the trigger's border is the neutral rung the ramp measures as
	// clearing the graphic floor against both sides of the edge, the same edge
	// the field and the radio wear (control.Border).
	borderCol := control.Border(tok.color, s.Ground)
	if s.Focused {
		borderCol = focus.Ring(tok.color)
	}
	if s.Disabled {
		borderCol = tokens.Disabled(control.Border(tok.color, s.Ground))
	}
	borderPx := gtx.Dp(1)
	if s.Focused {
		borderPx = gtx.Dp(focus.Width)
	}
	innerRad := rad - borderPx
	if innerRad < 0 {
		innerRad = 0
	}

	rrectOuter := clip.RRect{Rect: image.Rectangle{Max: triggerSize}, SE: rad, SW: rad, NE: rad, NW: rad}
	paint.FillShape(gtx.Ops, borderCol, rrectOuter.Op(gtx.Ops))
	rrectInner := clip.RRect{
		Rect: image.Rectangle{
			Min: image.Pt(borderPx, borderPx),
			Max: image.Pt(triggerSize.X-borderPx, triggerSize.Y-borderPx),
		},
		SE: innerRad, SW: innerRad, NE: innerRad, NW: innerRad,
	}
	paint.FillShape(gtx.Ops, bg, rrectInner.Op(gtx.Ops))

	// Text label: vertically centered.
	offY := (triggerH - labelDims.Size.Y) / 2
	st := op.Offset(image.Pt(padH, offY)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	st.Pop()

	// The mark, aligned to the right: the triangle points the way this field's
	// menu opens, which is the whole of what it has to say. See [Drop].
	cx := fieldW - padH - chevronSz/2
	cy := triggerH / 2
	chevronCol := tok.color.Ramps.Neutral.Step(700) // low-contrast glyph
	if s.Disabled {
		chevronCol = tokens.Disabled(chevronCol)
	}
	drawChevron(gtx, cx, cy, chevronSz, chevronCol, s.Drop == DropUp)

	return layout.Dimensions{Size: triggerSize}
}

// drawChevron draws a solid triangle centered at (cx, cy) with overall width
// sz, pointing up when up is set and down otherwise.
func drawChevron(gtx layout.Context, cx, cy, sz int, col color.NRGBA, up bool) {
	half := float32(sz) / 2
	quarter := float32(sz) / 4
	fcx := float32(cx)
	fcy := float32(cy)

	base, apex := fcy-quarter, fcy+quarter
	if up {
		base, apex = fcy+quarter, fcy-quarter
	}

	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(fcx-half, base))
	p.LineTo(f32.Pt(fcx+half, base))
	p.LineTo(f32.Pt(fcx, apex))
	p.Close()
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}
