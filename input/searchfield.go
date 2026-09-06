package input

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// ClearHitDp is the side of the pointer target the clear mark claims, in dp,
// centred on the mark and free to overhang the field.
//
// It is WCAG 2.5.8 Target Size (Minimum), the AA criterion, and not the 44 dp
// of [tokens.Density.MinHitTarget]: 44 is this system's floor for a
// standalone control with space around it, and a 44 dp target centred on a
// mark inside the field would reach past the field's own top and bottom. It
// is the same reading, and the same number, the dismissible chip's mark takes.
const ClearHitDp = 24

// SearchFieldProps configures a SearchField instance.
//
// The search field is the text field's structure with two slots added — the
// looking glass leading, the clear mark trailing — so everything the text
// field settles about density, focus, the hit target and the surface it
// stands on is settled here too and is not restated.
type SearchFieldProps struct {
	// Placeholder is shown when the field is empty and unfocused.
	Placeholder string

	// Description is the screen-reader label. Falls back to Placeholder when empty.
	Description string

	// Level is the level of the surface the field stands on — the field has
	// no level of its own. See [TextFieldProps.Level].
	Level tokens.ElevationLevel

	// Seed, when non-empty, pre-fills the editor when the field instance is
	// created. See [TextFieldProps.Seed]; the field stays uncontrolled.
	Seed string

	// FocusTag, if non-nil, is called once when the field instance is created
	// with the editor's focus tag. See [TextFieldProps.FocusTag].
	FocusTag func(tag event.Tag)

	// Disabled, if non-nil, disables the field when it emits true. A disabled
	// field draws its marks faded with the rest of it and offers no clear.
	Disabled rx.Observable[bool]

	// OnChange is called with the new value on every text change, the clear
	// mark's own included: clearing emits an empty string through this
	// callback like any other edit, which is how a consumer's highlight dies
	// with the query that caused it.
	OnChange func(gtx layout.Context, text string)

	// Message, if non-nil, causes the field to emit mvu.MessageOp{Message} on
	// every text change. This is the MVU integration path.
	Message any

	// ClearMessage, if non-nil, is emitted in addition to Message when the
	// clear mark empties the field — for a consumer that has something to do
	// on the dismissal itself beyond seeing the query go empty. A nil
	// ClearMessage means clearing is reported as the change it is and
	// nothing more.
	ClearMessage any

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use. See [TextFieldProps.Shaper].
	Shaper *text.Shaper
}

// SearchField returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes. Interaction state (editor
// content, focus, the clear mark's own press) lives in the rx.Defer scope and
// persists across emissions.
//
// The control is the text field looking as the reader types: the looking
// glass names it, the text is the query, and the clear mark appears while
// there is something to take back. Pressing the mark empties the editor and
// reports the empty query on the same frame.
//
// What is found and how the matches are marked is the consumer's — this
// field holds a query and nothing else.
//
// Both integration paths are supported:
//   - FRP: set SearchFieldProps.OnChange.
//   - MVU: set SearchFieldProps.Message and, if it needs one, ClearMessage.
func SearchField(th rx.Observable[theme.Theme], props SearchFieldProps) rx.Observable[layout.Widget] {
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
		// Allocated once per subscription — survives every theme and disabled
		// emission for the lifetime of this SearchField instance.
		editor := &widget.Editor{SingleLine: true}
		hitTag := new(int)
		clear := &widget.Clickable{}
		if props.Seed != "" {
			editor.SetText(props.Seed)
		}
		if props.FocusTag != nil {
			props.FocusTag(editor)
		}

		return rx.Map(inputs, func(next rx.Tuple2[resolvedTokens, bool]) layout.Widget {
			tok, dis := next.First, next.Second

			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}

			return func(gtx layout.Context) layout.Dimensions {
				if dis {
					gtx = gtx.Disabled()
				}

				emit := func(val string) {
					if props.OnChange != nil {
						props.OnChange(gtx, val)
					}
					if props.Message != nil {
						mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
					}
				}

				// The mark is drained before the editor so a press that
				// empties the field reports the empty query on the frame it
				// was pressed on, rather than a frame late. A double click on
				// a mark is one dismissal, so the presses collapse.
				typing := gtx.Focused(editor)
				cleared := false
				for clear.Clicked(gtx) {
					cleared = true
				}
				if cleared && !dis && editor.Len() > 0 {
					editor.SetText("")
					emit("")
					if props.ClearMessage != nil {
						mvu.MessageOp{Message: props.ClearMessage}.Add(gtx.Ops)
					}
					if typing {
						// Pressing the mark takes the focus, and a reader who
						// was typing a query means to go on typing it. The
						// mark takes back what was searched for, not the
						// keyboard.
						gtx.Execute(key.FocusCmd{Tag: editor})
					}
				}

				for {
					ev, ok := editor.Update(gtx)
					if !ok {
						break
					}
					if _, isChange := ev.(widget.ChangeEvent); isChange {
						emit(editor.Text())
					}
				}

				for {
					ev, ok := gtx.Event(pointer.Filter{Target: hitTag, Kinds: pointer.Press})
					if !ok {
						break
					}
					if _, isPtr := ev.(pointer.Event); isPtr && !dis {
						gtx.Execute(key.FocusCmd{Tag: editor})
					}
				}

				desc := props.Description
				if desc == "" {
					desc = props.Placeholder
				}

				foc := !dis && gtx.Focused(editor)
				showPh := !foc && editor.Len() == 0

				return drawTextFieldLive(gtx, shaper, editor, hitTag, props.Placeholder, desc, tok, RenderState{
					Focused:  foc,
					Disabled: dis,
					Level:    props.Level,
				}, showPh, adorn{
					search:    true,
					clear:     true,
					showClear: !dis && editor.Len() > 0,
					clearBtn:  clear,
					clearDesc: desc,
				})
			}
		})
	})
}

// RenderSearch produces a layout.Widget for a search field in an explicit
// visual state, without any event processing or rx machinery. Intended for
// golden-image testing and static demonstrations; production code should use
// [SearchField].
//
// The parameters are [Render]'s, and mean the same things. The clear mark is
// drawn whenever RenderState.Text is non-empty, which is the state the live
// field draws it in.
func RenderSearch(
	shaper *text.Shaper,
	placeholder string,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{color: colors, spacing: sp, radius: rad, body: body, density: d}
	ad := adorn{search: true, clear: true, showClear: !s.Disabled && s.Text != ""}
	return func(gtx layout.Context) layout.Dimensions {
		return drawTextFieldStatic(gtx, shaper, placeholder, tok, s, ad)
	}
}

// adorn is the pair of slots the search field adds to the text field's
// structure: the looking glass at the leading end and the clear mark at the
// trailing one. The zero value is a plain text field — it reserves no width
// and draws nothing, which is what keeps one drawing serving both controls.
type adorn struct {
	// search reserves and draws the leading looking glass.
	search bool
	// clear reserves the trailing slot. The slot is held whether or not the
	// mark is in it, so the text the reader is typing never reflows under
	// them when the field stops being empty.
	clear bool
	// showClear draws the mark in the reserved slot.
	showClear bool
	// clearBtn is the mark's own pointer target on the live path; nil on the
	// static one, which registers no target.
	clearBtn *widget.Clickable
	// clearDesc names, for a reader who cannot see the mark, the field the
	// mark empties.
	clearDesc string
}

// slotPx is the square each of the two slots is cut to, and the square the
// mark in it is drawn at: the control icon size the density gives, which is
// the same square components/button gives an icon beside a label and the
// same one every other mark in this library is drawn at. Both ends take one
// square, so the field is even end to end and neither mark outweighs the
// other.
func (a adorn) slotPx(gtx layout.Context, tok resolvedTokens) int {
	return gtx.Dp(icon.Size(tok.density))
}

// gapPx is the air between a mark and the text beside it.
func (a adorn) gapPx(gtx layout.Context, tok resolvedTokens) int {
	return gtx.Dp(unit.Dp(tok.spacing.S2))
}

// slots reports the width, in pixels, the field spends at each end on the
// marks — the mark's square plus the gap after it, and zero for a slot this
// field does not carry.
func (a adorn) slots(gtx layout.Context, tok resolvedTokens) (lead, trail int) {
	if !a.search && !a.clear {
		return 0, 0
	}
	w := a.slotPx(gtx, tok) + a.gapPx(gtx, tok)
	if a.search {
		lead = w
	}
	if a.clear {
		trail = w
	}
	return lead, trail
}

// paint draws the marks the field carries into the slots it reserved, and on
// the live path registers the clear mark's own pointer target.
//
// Both are drawn in the control family's prompt foreground: they say what the
// control is and what it offers, not what it holds, and a mark drawn in the
// text colour reads as content the reader put there. Disabled fades them with
// the rest of the field.
func (a adorn) paint(gtx layout.Context, tok resolvedTokens, s RenderState, field image.Point, padH int) {
	if !a.search && !a.clear {
		return
	}
	slot := a.slotPx(gtx, tok)
	col := control.Placeholder(tok.color)
	if s.Disabled {
		col = tokens.Disabled(col)
	}

	if a.search {
		if g := icons.Mark(icons.Search); g != nil {
			st := op.Offset(image.Pt(padH, (field.Y-slot)/2)).Push(gtx.Ops)
			g(gtx, slot, col)
			st.Pop()
		}
	}
	if a.clear && a.showClear {
		origin := image.Pt(field.X-padH-slot, (field.Y-slot)/2)
		if g := icons.Mark(icons.Clear); g != nil {
			st := op.Offset(origin).Push(gtx.Ops)
			g(gtx, slot, col)
			st.Pop()
		}
		a.registerClear(gtx, origin, slot)
	}
}

// registerClear puts the clear mark's clickable over the mark, grown to
// [ClearHitDp] on each axis and centred on it.
//
// The field's own reported size is unaffected, and so is the editor's box:
// the slot the mark stands in was taken out of the text's width before
// either was laid out, so the target overlaps nothing the reader types into.
// It is registered after the editor's area, which is what makes it take the
// pointer where the two nevertheless meet.
func (a adorn) registerClear(gtx layout.Context, origin image.Point, mark int) {
	if a.clearBtn == nil {
		return
	}
	target := max(gtx.Dp(unit.Dp(ClearHitDp)), mark)
	off := op.Offset(image.Pt(origin.X-(target-mark)/2, origin.Y-(target-mark)/2)).Push(gtx.Ops)
	a.clearBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		semantic.ClassOp(semantic.Button).Add(gtx.Ops)
		// What the mark empties is this field, and a reader reaching it
		// should be told which one rather than a word this package invented.
		semantic.LabelOp(a.clearDesc).Add(gtx.Ops)
		semantic.EnabledOp(true).Add(gtx.Ops)
		pointer.CursorPointer.Add(gtx.Ops)
		return layout.Dimensions{Size: image.Pt(target, target)}
	})
	off.Pop()
}
