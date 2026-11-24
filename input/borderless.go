package input

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
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
	"github.com/vibrantgio/components/internal/surface"
	"github.com/vibrantgio/components/pointershape"
	"github.com/vibrantgio/mvu"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// BorderlessTextFieldProps configures a BorderlessTextField instance.
//
// The borderless text field is the text field with no bezel: a title edited
// in its header, a paragraph edited in place. Its text is drawn as the text
// it edits is drawn, in the same role and the same colour, at the same
// origin. It keeps the role of the text it edits, and Role names that role.
// A bordered text field sets its text in the System font role instead.
// The text stands in the same place while it is edited and after, and
// nothing moves when the edit begins or ends.
type BorderlessTextFieldProps struct {
	// Placeholder is shown while the field is empty, also while the field
	// holds the keyboard. It leaves when the user types.
	Placeholder string

	// Description is the screen-reader label. Empty falls back to
	// Placeholder.
	Description string

	// Seed, when non-empty, fills the editor when the field instance is
	// created. Later Seed values have no effect on a live instance.
	Seed string

	// Role answers the text style of the text the field edits. Nil answers
	// the Body role.
	Role func(tokens.Typography) tokens.TextStyle

	// Foreground answers the colour of the text the field edits, for the
	// scheme the field is drawing in. The field flattens it onto Surface.
	// Nil answers the platform's label colour.
	Foreground func(tokens.PlatformColors) color.NRGBA

	// Surface answers the opaque fill the field stands on. The field has no
	// fill of its own: its text, its placeholder and its insertion point are drawn on
	// this fill. Nil answers the window's own plane.
	Surface func(tokens.PlatformColors) color.NRGBA

	// FocusTag, if non-nil, is called once when the field instance is
	// created, with the editor's focus tag.
	FocusTag func(tag event.Tag)

	// Disabled, if non-nil, disables the field when it emits true.
	Disabled rx.Observable[bool]

	// OnChange is called with the new value on every text change.
	OnChange func(gtx layout.Context, text string)

	// Message, if non-nil, causes the field to emit mvu.MessageOp{Message}
	// on every text change.
	Message any

	// Submit, when true, makes Return a submit and not a line break.
	// Shift+Return still breaks the line.
	Submit bool

	// SubmitMessage, if non-nil, is invoked on each submit, and its return
	// value is emitted as an mvu.MessageOp on the current frame.
	SubmitMessage func(text string) any

	// OnSubmit, if non-nil, is invoked on each submit with the editor's
	// text. The field keeps its text after a submit: the caller that ends
	// the edit removes the field or makes a new one.
	OnSubmit func(gtx layout.Context, text string)

	// Shaper overrides the theme's shaper for this field. Leave it nil in
	// normal use.
	Shaper *text.Shaper
}

// BorderlessState is the explicit visual state of a borderless text field
// for the static render path.
type BorderlessState struct {
	// Surface is the opaque fill the field stands on. The zero value is the
	// window's own plane.
	Surface color.NRGBA

	// Foreground is the colour of the text the field edits. The zero value
	// is the platform's label colour. It is flattened onto Surface.
	Foreground color.NRGBA

	// Text, when non-empty, is drawn in place of the placeholder.
	Text string

	// Selection is the run of Text standing selected, a half-open rune
	// range into it.
	Selection [2]int

	Focused  bool
	Disabled bool
}

// borderlessTokens is the per-emission snapshot a borderless field draws
// from.
type borderlessTokens struct {
	platform tokens.PlatformColors
	role     tokens.TextStyle
	shaper   *text.Shaper
}

// borderlessColors returns the fill the field stands on, the colour of its
// text, and the colour of its placeholder. The placeholder is the
// platform's placeholder colour, the colour the bordered field draws its
// placeholder in. A disabled field draws its text and its placeholder in the
// platform's disabled control text.
func borderlessColors(p tokens.PlatformColors, s BorderlessState) (fill, fg, ph color.NRGBA) {
	fill = surface.Or(s.Surface, p.WindowBackground)
	fill.A = 0xff
	fg = s.Foreground
	if fg.A == 0 {
		fg = p.Label
	}
	fg = vgcolor.Flatten(fg, fill)
	ph = control.Placeholder(p, fill)
	if s.Disabled {
		fg = vgcolor.Flatten(p.DisabledControlText, fill)
		ph = fg
	}
	return fill, fg, ph
}

// BorderlessTextField returns an rx.Observable[layout.Widget] that emits a
// new layout.Widget whenever the theme or the disabled state changes. The editor's
// text and focus live in the rx.Defer scope and persist across emissions.
//
// The field draws no bezel, no fill, no height floor, no inset and no focus
// ring. DERIVED: the platform draws no focus ring around a title it edits in
// place, as Finder's rename and Reminders' title show, and the stored
// reference holds no measurement of one. The field is as wide as the width
// it is given and as tall as its text: its lines wrap where a label of the
// same role wraps them, and the field grows with them.
func BorderlessTextField(th rx.Observable[theme.Theme], props BorderlessTextFieldProps) rx.Observable[layout.Widget] {
	disabled := props.Disabled
	if disabled == nil {
		disabled = rx.Of(false)
	}
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[borderlessTokens] {
		return rx.Map(rx.CombineLatest2(t.Platform, t.Typography),
			func(n rx.Tuple2[tokens.PlatformColors, tokens.Typography]) borderlessTokens {
				role := n.Second.Body
				if props.Role != nil {
					role = props.Role(n.Second)
				}
				return borderlessTokens{platform: n.First, role: role, shaper: n.Second.Shaper()}
			})
	})
	inputs := rx.CombineLatest2(resolved, disabled)

	return rx.Defer(func() rx.Observable[layout.Widget] {
		editor := &widget.Editor{Submit: props.Submit}
		in := newInsertion()
		if props.Seed != "" {
			editor.SetText(props.Seed)
		}
		if props.FocusTag != nil {
			props.FocusTag(editor)
		}
		return rx.Map(inputs, func(next rx.Tuple2[borderlessTokens, bool]) layout.Widget {
			tok, dis := next.First, next.Second
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			return func(gtx layout.Context) layout.Dimensions {
				if dis {
					gtx = gtx.Disabled()
				}
				in.begin(gtx, editor, !dis && gtx.Focused(editor))
				for {
					ev, ok := editor.Update(gtx)
					if !ok {
						break
					}
					switch ev := ev.(type) {
					case widget.ChangeEvent:
						if props.OnChange != nil {
							props.OnChange(gtx, editor.Text())
						}
						if props.Message != nil {
							mvu.MessageOp{Message: props.Message}.Add(gtx.Ops)
						}
					case widget.SubmitEvent:
						if props.SubmitMessage != nil {
							mvu.MessageOp{Message: props.SubmitMessage(ev.Text)}.Add(gtx.Ops)
						}
						if props.OnSubmit != nil {
							props.OnSubmit(gtx, ev.Text)
						}
					}
				}
				desc := props.Description
				if desc == "" {
					desc = props.Placeholder
				}
				foc := !dis && gtx.Focused(editor)
				in.presses(gtx)
				s := BorderlessState{
					Surface:  standsOn(props.Surface, tok.platform),
					Focused:  foc,
					Disabled: dis,
				}
				if props.Foreground != nil {
					s.Foreground = props.Foreground(tok.platform)
				}
				return drawBorderlessLive(gtx, shaper, editor, in, props.Placeholder, desc, tok, s)
			}
		})
	})
}

// RenderBorderless produces a layout.Widget for a borderless text field in an
// explicit visual state, without event processing. It is the static path for
// golden images. role is the text style of the text the field edits.
func RenderBorderless(shaper *text.Shaper, placeholder string, p tokens.PlatformColors, role tokens.TextStyle, s BorderlessState) layout.Widget {
	tok := borderlessTokens{platform: p, role: role, shaper: shaper}
	return func(gtx layout.Context) layout.Dimensions {
		return drawBorderlessStatic(gtx, shaper, placeholder, tok, s)
	}
}

// borderlessLabel is the font, the label and the size the field draws its
// text with on gtx: the role's font at the stem weight of fg on the fill gtx
// carries, and a label of the role with no line limit. They are the font and
// the label typeset.Text draws the same text with.
func borderlessLabel(gtx layout.Context, role tokens.TextStyle, fg color.NRGBA) (font.Font, widget.Label, unit.Sp) {
	return typeset.FontOn(gtx, role, font.Normal, fg), typeset.Label(role, 0), unit.Sp(role.Size)
}

// paintOf records a paint source of col.
func paintOf(gtx layout.Context, col color.NRGBA) op.CallOp {
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	return rec.Stop()
}

// drawBorderlessLive draws a live borderless field around editor.
//
// The editor shapes its text with the font, the size and the line height
// typeset.Text shapes it with, and at the same width, so its glyphs and its
// line breaks are those of the label. typeset.Layout puts half the line
// box's deficit above the glyphs, and a widget.Editor puts all of it below
// them. The field therefore draws the editor that half lower, and the
// editor's glyphs land on the label's pixels.
func drawBorderlessLive(gtx layout.Context, shaper *text.Shaper, editor *widget.Editor, in *insertion, placeholder, desc string, tok borderlessTokens, s BorderlessState) layout.Dimensions {
	fill, fg, phColor := borderlessColors(tok.platform, s)
	gtx = typeset.WithFill(gtx, fill)
	width := gtx.Constraints.Max.X

	semantic.ClassOp(semantic.Editor).Add(gtx.Ops)
	semantic.DescriptionOp(desc).Add(gtx.Ops)
	semantic.EnabledOp(!s.Disabled).Add(gtx.Ops)

	inner := gtx
	inner.Constraints = layout.Constraints{Max: image.Pt(width, gtx.Constraints.Max.Y)}

	// The height is the line box of what the field shows: the value, or the
	// placeholder while the field is empty.
	shown, shownColor := editor.Text(), fg
	if shown == "" {
		shown, shownColor = placeholder, phColor
	}
	f, lbl, size := borderlessLabel(gtx, tok.role, shownColor)
	shownMat := paintOf(gtx, shownColor)
	rec := op.Record(gtx.Ops)
	dims := typeset.Layout(inner, shaper, lbl, f, size, shown, shownMat)
	shownCall := rec.Stop()
	if editor.Len() == 0 {
		shownCall.Add(gtx.Ops)
	}

	textF, _, _ := borderlessLabel(gtx, tok.role, fg)
	if tok.role.LineHeight > 0 {
		editor.LineHeight = unit.Sp(tok.role.LineHeight)
		editor.LineHeightScale = 1
	}
	selFill, selRun := selectionPair(tok.platform, s.Focused)
	egtx := inner
	egtx.Constraints = layout.Constraints{
		Min: image.Pt(width, 0),
		Max: image.Pt(width, max(dims.Size.Y, 1)),
	}
	shift := editorTextShift(inner, shaper, lbl, f, size, shown, shownMat, dims.Size.Y)
	st := op.Offset(image.Pt(0, shift)).Push(gtx.Ops)
	layoutEditor(egtx, editor, func(gtx layout.Context) layout.Dimensions {
		return editor.Layout(gtx, shaper, textF, size, paintOf(gtx, fg), paintOf(gtx, selFill))
	})
	under := color.NRGBA{}
	if !s.Focused {
		under = selFill
	}
	runF, _, _ := borderlessLabel(gtx, tok.role, selRun)
	paintSelectedRun(egtx, editor, shaper, runF, size, under, selRun)
	// The insertion point spans the line box, which starts shift above the
	// editor's own top.
	ascent, lineH := lineBox(inner, shaper, lbl, f, size, shown, shownMat)
	in.paint(egtx, editor, s.Focused, tok.platform, ascent+shift, lineH, width)
	st.Pop()

	out := image.Pt(width, dims.Size.Y)
	if !s.Disabled {
		pointershape.OverSize(gtx.Ops, out, pointer.CursorText)
	}
	in.target(gtx, out)
	return layout.Dimensions{Size: out, Baseline: dims.Baseline}
}

// drawBorderlessStatic draws a borderless field in an explicit state: the
// value or the placeholder through typeset.Layout, a selected run over the
// selection's fill, and the insertion point while it is drawn as focused.
func drawBorderlessStatic(gtx layout.Context, shaper *text.Shaper, placeholder string, tok borderlessTokens, s BorderlessState) layout.Dimensions {
	fill, fg, phColor := borderlessColors(tok.platform, s)
	gtx = typeset.WithFill(gtx, fill)
	width := gtx.Constraints.Max.X
	inner := gtx
	inner.Constraints = layout.Constraints{Max: image.Pt(width, gtx.Constraints.Max.Y)}

	shown, shownColor := s.Text, fg
	if shown == "" {
		shown, shownColor = placeholder, phColor
	}
	sel, selected := borderlessSelection(gtx, shaper, tok, s, width)
	selFill, selRun := selectionPair(tok.platform, s.Focused)
	if selected {
		paint.FillShape(gtx.Ops, selFill, clip.Rect(sel).Op())
	}
	f, lbl, size := borderlessLabel(gtx, tok.role, shownColor)
	dims := typeset.Layout(inner, shaper, lbl, f, size, shown, paintOf(gtx, shownColor))
	if selected {
		runF, _, _ := borderlessLabel(gtx, tok.role, selRun)
		rec := op.Record(gtx.Ops)
		typeset.Layout(inner, shaper, lbl, runF, size, shown, paintOf(gtx, selRun))
		run := rec.Stop()
		area := clip.Rect(sel).Push(gtx.Ops)
		run.Add(gtx.Ops)
		area.Pop()
	}
	// A field drawn as focused draws its insertion point after its value, or
	// where the text begins beside the prompt. A selected run draws none.
	if s.Focused && !s.Disabled && !selected {
		_, lineH := lineBox(inner, shaper, lbl, f, size, shown, paintOf(gtx, shownColor))
		textF, textLbl, _ := borderlessLabel(gtx, tok.role, fg)
		x := staticInsertionX(gtx, shaper, textF, textLbl, size, s.Text, width)
		drawInsertion(gtx, tok.platform, x, 0, lineH, width)
	}
	return layout.Dimensions{Size: image.Pt(width, dims.Size.Y), Baseline: dims.Baseline}
}

// borderlessSelection is the box the static path's selected run stands in,
// for a value on one line. It is [staticSelection] for the role the field
// edits, across that role's line box.
func borderlessSelection(gtx layout.Context, shaper *text.Shaper, tok borderlessTokens, s BorderlessState, width int) (image.Rectangle, bool) {
	box := gtx.Sp(unit.Sp(tok.role.LineHeight))
	rt := resolvedTokens{platform: tok.platform, body: tok.role}
	return staticSelection(gtx, shaper, rt, RenderState{Text: s.Text, Selection: s.Selection}, s.Text, width, box)
}
