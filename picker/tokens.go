package picker

import (
	"gioui.org/font"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// resolvedTokens is the concrete per-emission snapshot the widget closures
// draw from: the whole theme flattened to the values one frame needs.
//
// Two text roles, because the two triggers are drawn for two registers. The
// field and the menu rows are BodyLarge, the role the form controls beside
// them are set in; the anchor is LabelLarge, the role the chip family it
// belongs to is set in.
type resolvedTokens struct {
	color   tokens.ColorTokens
	body    tokens.TextStyle // the BodyLarge role: the field and the menu rows
	label   tokens.TextStyle // the LabelLarge role: the anchor
	spacing tokens.SpacingScale
	radius  tokens.RadiusScale
	density tokens.Density // control height and inner padding
	shaper  *text.Shaper   // the theme's shaper; nil in the Render* paths
}

// bodyLabel derives the Gio font, a single-line label and the text size from
// the BodyLarge role carried in tok. Zero fields fall back to the shaper's
// defaults. Lay the returned label out with typeset.Layout rather than
// widget.Label.Layout: the role's line height is the height of the line box,
// which Gio does not give a single line on its own.
func bodyLabel(tok resolvedTokens) (font.Font, widget.Label, unit.Sp) {
	style := tok.body
	return typeset.Font(style, font.Normal), typeset.Label(style, 1), unit.Sp(style.Size)
}
