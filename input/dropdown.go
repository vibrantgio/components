package input

import (
	"gioui.org/layout"
	"gioui.org/text"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/picker"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// DropdownRenderState holds explicit visual state for static rendering.
//
// Deprecated: use picker.FieldState. This is an alias of it, so a state
// written against either name is the same value.
type DropdownRenderState = picker.FieldState

// DropdownProps configures a Dropdown instance.
//
// Deprecated: use picker.FieldProps. This is an alias of it, so props written
// against either name are the same value.
type DropdownProps = picker.FieldProps

// Dropdown returns an rx.Observable[layout.Widget] that emits a new widget
// whenever the theme or disabled state changes.
//
// Deprecated: the pick-one-from-many affordance is components/picker's, where
// one menu serves both the form register's trigger and the chrome register's.
// Use picker.Field, which this forwards to; it draws the same control from the
// same code, and picker.Menu is the surface it drops.
func Dropdown(th rx.Observable[theme.Theme], props DropdownProps) rx.Observable[layout.Widget] {
	return picker.Field(th, props)
}

// RenderDropdown produces a layout.Widget for a dropdown in an explicit visual
// state, without any event processing or rx machinery.
//
// Deprecated: use picker.RenderField, which this forwards to.
func RenderDropdown(
	shaper *text.Shaper,
	colors tokens.ColorTokens,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	body tokens.TextStyle,
	d tokens.Density,
	s DropdownRenderState,
) layout.Widget {
	return picker.RenderField(shaper, colors, sp, rad, body, d, s)
}
