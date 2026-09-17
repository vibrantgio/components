// Package control holds the colours every Vibrant Gio form control paints
// the box it draws: the interior it fills, the edge it draws around it, and
// the foreground of the prompt it shows where a value is not there yet. They
// live here rather than in components/input because components/picker's
// field trigger is the same box under a different package — one control
// family, one set of names, and no second answer to keep in step.
//
// Each is the platform's own name for what that part of a field is; nothing
// here measures, walks or derives.
package control

import (
	"image/color"

	"github.com/vibrantgio/components/internal/surface"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/tokens"
)

// Border is the hairline a control draws around itself at rest — the
// unchecked box, the unselected radio, the text field, the picker's field
// trigger: the platform's field edge, measured off the Save dialog's
// unfocused field in both appearances.
func Border(p tokens.PlatformColors) color.NRGBA { return p.FieldEdge }

// Fill is the interior of a control that paints a box of its own — the
// unchecked box, the unselected radio's gap ring: the platform's text
// background, which is what it fills a box with.
func Fill(p tokens.PlatformColors) color.NRGBA { return p.TextBackground }

// FieldFill is the interior of a text field: the surface the field stands
// on, with [Border] the only thing the field draws of its own.
//
// MEASURED, save-dialog-light.png and save-dialog-dark.png, the unfocused
// "Tags:" field: its interior reads its sheet's own fill in both
// appearances — #ffffff light and #232a2f dark — where the platform's text
// background is #1e1e1e dark. The dark reading is what settles it: a field
// filling itself with the text background could not land on #232a2f. So a
// field on a sidebar material is that material inside its hairline, not a
// white box standing on it.
//
// standsOn is the caller's stated surface; alpha zero is no answer, and the
// platform's text background stands in — which is the window's own plane and
// the content's fill alike on this platform.
func FieldFill(p tokens.PlatformColors, standsOn color.NRGBA) color.NRGBA {
	return surface.Or(standsOn, p.TextBackground)
}

// Placeholder is the foreground a control's prompt is drawn in: the wording
// a text field or a picker's field trigger shows in the space its value will
// occupy, while there is no value there yet. It is the platform's
// placeholder text flattened over beneath — the control's own fill, which is
// [Fill] for a text field and the push button's for a picker's trigger.
func Placeholder(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.PlaceholderText, beneath)
}

// DisabledFill is the interior of a control that is switched off and still
// carries a value — the checked box, the selected radio. The accent drains:
// the platform's disabled coverage is laid over what the control stands on,
// and beneath is that surface.
//
// The platform draws no accent on a disabled control. The Save dialog's two
// switched-off checkboxes carry none (save-dialog-light.png and -dark.png,
// the "Options:" rows), and their wording reads at that same coverage —
// #bdbdbd on the light sheet's white and #595f62 on the dark sheet's
// #232a2f, which DisabledControlText's black and white reproduce to the byte
// in dark and within three 255ths in light.
func DisabledFill(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.DisabledControlText, beneath)
}

// DisabledMark is the foreground of the mark a switched-off control still
// shows — the check, the radio's dot — over [DisabledFill]. It is the
// platform's own control text flattened onto that fill, so the mark stays
// readable in both appearances where white would be lost on the light
// fill's grey.
func DisabledMark(p tokens.PlatformColors, fill color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.ControlText, fill)
}
