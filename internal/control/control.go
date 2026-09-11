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

	"github.com/vibrantgio/theme/tokens"
)

// Border is the hairline a control draws around itself at rest — the
// unchecked box, the unselected radio, the text field, the picker's field
// trigger: the platform's field edge, measured off the Save dialog's
// unfocused field in both appearances.
func Border(p tokens.PlatformColors) color.NRGBA { return p.FieldEdge }

// Fill is the interior of a control that paints a box of its own — the
// unchecked box, the unselected radio's gap ring, the text field, the
// picker's field trigger: the platform's text background, which is what it
// fills a field with.
func Fill(p tokens.PlatformColors) color.NRGBA { return p.TextBackground }

// Placeholder is the foreground a control's prompt is drawn in: the wording
// a text field or a picker's field trigger shows in the space its value will
// occupy, while there is no value there yet. It is the platform's
// placeholder text, which composites over the field's fill.
func Placeholder(p tokens.PlatformColors) color.NRGBA { return p.PlaceholderText }
