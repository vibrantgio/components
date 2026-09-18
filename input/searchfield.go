package input

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
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/icons"
	"github.com/vibrantgio/components/internal/control"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// ClearHitDp is the side of the pointer target the clear mark claims, in dp,
// centred on the mark and free to overhang the field.
//
// It is larger than the mark is drawn, because a mark a few dp across is not
// something a pointer can be asked to land on, and small enough not to reach
// past the field's own top and bottom. It is the same number, for the same
// reason, the dismissible chip's mark takes.
const ClearHitDp = 24

// markDp is the square the search field draws each of its marks in, and the
// lead and gap pairs place the leading glyph and the text after it, one pair
// per variant. Each is measured; the provenance is on the method that spends
// it.
const (
	markDp        unit.Dp = 16
	sidebarLeadDp unit.Dp = 9
	sidebarGapDp  unit.Dp = 5
	toolbarLeadDp unit.Dp = 10
	toolbarGapDp  unit.Dp = 8
)

// SearchFieldProps configures a SearchField instance.
//
// The search field is the text field's structure with two slots added — the
// looking glass leading, the clear mark trailing — so everything the text
// field settles about density, focus, the pointer target and the surface it
// stands on is settled here too and is not restated.
type SearchFieldProps struct {
	// Placeholder is shown when the field is empty and unfocused.
	Placeholder string

	// Variant is where this field stands: [Form] on the content's plane,
	// [Chrome] on a sidebar or a toolbar, where the platform draws the field
	// as a flat recess instead of a bordered box. The zero value is Form.
	//
	// State Surface with it: the recess is opaque, but the focus ring and
	// the marks around it still composite onto the chrome material the
	// field stands on.
	Variant Variant

	// Region is which chrome region this field stands in, read only where
	// Variant is [Chrome]. State [Toolbar] for a field standing in a toolbar
	// band; the zero value is [Sidebar], the recess a sidebar carries.
	Region Region

	// Surface answers the opaque fill this field stands on, for the scheme
	// the field is drawing in: what its interior is filled with, and what
	// the platform's coverages composite over. It is a function of the
	// colour set rather than a colour because a live field outlives a change
	// of scheme, and the fill it stands on is a different colour on the
	// other side of one — patterns/pane.Surface is the same shape for the
	// same reason.
	//
	// Leave it nil where the field stands on the window's own plane; state
	// it where the field stands on something else — a chrome rail's
	// material, a card.
	Surface func(tokens.PlatformColors) color.NRGBA

	// Description is the screen-reader label. Falls back to Placeholder when empty.
	Description string

	// Seed, when non-empty, pre-fills the editor when the field instance is
	// created. See [TextFieldProps.Seed]; the field stays uncontrolled.
	Seed string

	// FocusTag, if non-nil, is called once when the field instance is created
	// with the editor's focus tag. See [TextFieldProps.FocusTag].
	FocusTag func(tag event.Tag)

	// Clear, if non-nil, is called once when the field instance is created
	// with a function that empties the field. It is how a query is taken back
	// from outside the field — a key the application binds, a panel closing —
	// where the clear mark takes it back from inside.
	//
	// Emptying the field this way reports the empty query on the next frame,
	// through OnChange and Message like any other edit, which is what
	// dismisses a consumer's highlight along with the query that caused it.
	Clear func(clear func())

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
			rx.CombineLatest5(t.Platform, t.Typography, t.Spacing, t.Radius, t.Density),
			func(n rx.Tuple5[tokens.PlatformColors, tokens.Typography, tokens.SpacingScale, tokens.RadiusScale, tokens.Density]) resolvedTokens {
				typ := n.Second
				return resolvedTokens{
					platform: n.First,
					body:     typ.BodyLarge,
					capBand:  typ.FaceMetrics(typ.BodyLarge).CapHeight,
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
		if props.Clear != nil {
			props.Clear(func() { editor.SetText("") })
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
					Surface:  standsOn(props.Surface, tok.platform),
					Variant:  props.Variant,
					Region:   props.Region,
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
// The parameters are [Render]'s, and mean the same things.
// RenderState.Variant picks the variant here as SearchFieldProps.Variant
// does on the live path, and RenderState.Region the chrome region as
// SearchFieldProps.Region does. The clear mark is drawn whenever RenderState.Text is
// non-empty, which is the state the live field draws it in.
func RenderSearch(
	shaper *text.Shaper,
	placeholder string,
	p tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s RenderState,
) layout.Widget {
	tok := resolvedTokens{platform: p, spacing: sp, radius: rad, body: body, capBand: body.FaceMetrics().CapHeight, density: d}
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

// markPx is the square each of the two slots is cut to, and the square the
// mark in it is drawn at. Both ends take one square, so the field is even end
// to end and neither mark outweighs the other.
//
// It is the field's own number, not the density's control icon size, because
// the platform draws this glyph at a size of its own and that size is
// measured: MEASURED, mail-window.png and voicememos-window.png — the
// magnifier is drawn 12.9 px square in both, its lens 10.25 px across
// outside and its band 1.25 px. The set draws a mark to 20 of its 24
// grid units, so the square that comes out at the platform's size is
// 12.9 × 24/20 = 15.5, and 16 dp draws the glyph 13.3 px across with a
// 10.7 px lens and a 1.33 px band. Density does not move it: no stored
// capture holds a search field at the platform's small size, and the
// checkbox carries its own side length for the same reason.
func (a adorn) markPx(gtx layout.Context) int { return gtx.Dp(markDp) }

// drawingPx is how much of that square the looking glass actually covers.
func (a adorn) drawingPx(gtx layout.Context) float32 {
	return icons.SearchDrawingSize * float32(a.markPx(gtx))
}

// glyphX is the field's leading edge to the looking glass's first pixel.
//
// Every reading is taken from the field's INNER edge, which is where the
// platform sets them. MEASURED,
// system-settings-grouped-box-{light,dark}.png: the sidebar recess carries no
// edge, so its inner edge is its own at x=18, and the glyph's first pixel is
// at x=27 — 9 px in. MEASURED, mail-window.png: a toolbar search field's
// stroke is at x=867 and its fill begins at x=868, with the glyph's first
// pixel at x=878 — 10 px in from that inner edge.
//
// A field with an edge spends that edge's width before the inset, and spends
// it whatever the field's state, so focus — which replaces the edge with a
// wider ring — does not move the glyph. The form field's edge is [hairlineDp]
// and the toolbar recess's rim is drawn at the same hairline, so the two
// spend one expression; the sidebar recess has no edge to spend.
//
// The toolbar recess spends the rim's column in the light appearance too,
// where the platform draws no rim: the box is the same control in both
// schemes and only whether the rim is visible moves, and no stored light
// toolbar outside Voice Memos holds a field to read the inset off.
//
// Voice Memos' own capsule sets its glyph 13 px in from its fill in both
// appearances — x 644 to x 657 dark, x 700 to x 713 light — which is that
// application's number and not the platform's; Finder's dark toolbar field
// agrees with Mail's ten (fill from x=1158, glyph from x=1167).
func (a adorn) glyphX(gtx layout.Context, s RenderState) float32 {
	if s.Variant == Chrome && s.Region == Sidebar {
		return float32(gtx.Dp(sidebarLeadDp))
	}
	return float32(gtx.Dp(hairlineDp) + gtx.Dp(toolbarLeadDp))
}

// promptGapPx is the clear space between the looking glass's last pixel and
// the first of the text beside it.
//
// MEASURED, system-settings-grouped-box-{light,dark}.png: the glyph's last
// pixel is at x=41 and the prompt's first at x=47, five clear columns between
// them. MEASURED, mail-window.png the same way: the glyph's last pixel is at
// x=890 and the prompt's first at x=899, eight clear columns. Voice Memos'
// own field leaves seven (the glyph ends at x=669 and the prompt opens at
// x=677), the same third place's drawing its 13 px inset is.
func (a adorn) promptGapPx(gtx layout.Context, s RenderState) int {
	if s.Variant == Chrome && s.Region == Sidebar {
		return gtx.Dp(sidebarGapDp)
	}
	return gtx.Dp(toolbarGapDp)
}

// trailGapPx is the clear space held between the text and the clear mark.
func (a adorn) trailGapPx(gtx layout.Context, tok resolvedTokens) int {
	return gtx.Dp(unit.Dp(tok.spacing.S2))
}

// insets report where the text starts and how much the field holds at its
// trailing end, both measured from the field's own edges. A field carrying
// no looking glass starts its text at the measured [control.TextLeadDp]
// inside its inner edge, which is what keeps one drawing serving the text
// field as well.
//
// The gap is spent from the last pixel column the looking glass covers, not
// from the fraction of a column its drawing ends on: the glyph is placed at a
// fraction of a pixel so its lens lands where the platform draws it, and a
// drawing ending mid-column still covers that column. Rounding the end down
// would spend one of the gap's columns on the drawing and leave the
// platform's measured clear space a column short.
func (a adorn) insets(gtx layout.Context, tok resolvedTokens, s RenderState, padH int) (lead, trail int) {
	lead = gtx.Dp(hairlineDp) + gtx.Dp(control.TextLeadDp)
	trail = padH
	if a.search {
		end := int(math.Ceil(float64(a.glyphX(gtx, s) + a.drawingPx(gtx))))
		lead = end + a.promptGapPx(gtx, s)
	}
	if a.clear {
		trail = padH + a.markPx(gtx) + a.trailGapPx(gtx, tok)
	}
	return lead, trail
}

// paint draws the marks the field carries into the slots it reserved, and on
// the live path registers the clear mark's own pointer target.
//
// Both are drawn in the platform's secondary label: they say what the control
// is and what it offers, not what it holds, and a mark drawn in the text
// colour reads as content the reader put there. MEASURED,
// system-settings-grouped-box-{light,dark}.png: the looking glass and the
// prompt in the sidebar's recess are one colour, and it is that coverage over
// the recess — #747474 on the light #e8e8e8, to the byte. A disabled field offers no
// clear, so both take the platform's disabled control text with the rest of
// the field's foregrounds.
func (a adorn) paint(gtx layout.Context, tok resolvedTokens, s RenderState, field image.Point, padH int) {
	if !a.search && !a.clear {
		return
	}
	// Both marks stand inside the field, so both are flattened onto the
	// field's own fill: the platform's names carry a coverage.
	slot := a.markPx(gtx)
	fill := fieldFill(tok.platform, s)
	col := vgcolor.Flatten(tok.platform.SecondaryLabel, fill)
	if s.Disabled {
		col = vgcolor.Flatten(tok.platform.DisabledControlText, fill)
	}

	if a.search {
		if g := icons.Mark(icons.Search); g != nil {
			// The square is placed off the drawing inside it rather than off
			// its own edges, and at a fraction of a pixel, because both
			// numbers the platform gives are the glyph's: its first pixel 9
			// in, and its lens — not its bounding box — on the field's centre
			// row. Rounding the square to whole pixels instead would split
			// the lens's band across two columns and draw it grey.
			x := a.glyphX(gtx, s) - icons.SearchDrawingOrigin*float32(slot)
			y := float32(field.Y)/2 - icons.SearchLensCentre*float32(slot)
			st := op.Affine(f32.Affine2D{}.Offset(f32.Pt(x, y))).Push(gtx.Ops)
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
