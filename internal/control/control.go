// Package control holds what every Vibrant Gio form control draws its box
// by: the colours — the interior it fills, the edge it draws around it, and
// the foreground of the prompt it shows where a value is not there yet — the
// two insets it spends on the text standing inside that box, and the pop-up
// mark both of the picker's triggers wear. They live here rather than in
// components/input because components/picker's field trigger is the same box
// under a different package — one control family, one set of names and
// numbers, and no second answer to keep in step.
//
// Each colour is the platform's own name for what that part of a field is;
// each inset and the mark's whole geometry are read off a stored capture at
// 1x. Nothing here walks or derives.
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

// Recess is the interior of a search field standing on chrome — a sidebar, a
// toolbar: the platform's flat recess, a fill of its own rather than the
// chrome material showing through the way [FieldFill] has a form field
// show it.
//
// MEASURED, system-settings-grouped-box-light.png and -dark.png, the field at
// the top of System Settings' sidebar: #e8e8e8 over a #fafaf9 sidebar light
// and #2f3234 over a #1c2124 one dark. The light value stands unchanged on
// Voice Memos' #ffffff toolbar band, so it is a colour and not a coverage
// over what the field stands on.
func Recess(p tokens.PlatformColors) color.NRGBA { return p.SidebarSearchFill }

// ToolbarRecess is the interior of a search field standing in a TOOLBAR: the
// platform's second recess, a different fill from the sidebar's [Recess] and
// from the bordered control beside it in the same band.
//
// MEASURED, voicememos-sidebar-light.png and voicememos-window.png, the
// search field at the trailing end of a frontmost Voice Memos toolbar: its
// interior is flat #e8e8e8 light — the value the sidebar recess carries — and
// #363636 dark over a #1e1e1e band, where the sidebar recess reads #2f3234
// and a bordered toolbar control reads #262626. Light the two recesses are
// one value; dark they are not, which is why the toolbar's carries a name of
// its own.
func ToolbarRecess(p tokens.PlatformColors) color.NRGBA { return p.ToolbarSearchFill }

// ToolbarSearchRim is the hairline that recess wears round its own edge: the
// platform's measured value for it and not an alpha name flattened, because
// no name lands the pixel. It answers no colour in the light appearance,
// where the platform draws no rim at all, and a caller draws nothing there.
//
// It is apart from [ToolbarRim], which is what a BORDERED toolbar control
// wears: that rim is the separator over the control's own fill and misses by
// five of 255 there, where over the recess's fill it misses by three. The two
// are different pixels over different fills, so the recess carries its own
// name rather than borrowing that one.
//
// MEASURED, voicememos-window.png, the search field at x 643-967, y 8-43 of a
// frontmost dark toolbar: #4d4d4d flat along y=8 and y=43 over x 669-941, and
// #4b4b4b and #4a4a4a down the columns at either end, so the rim runs the
// whole way round. MEASURED, voicememos-sidebar-light.png: the light band
// steps straight to the fill with no stroke row on any side.
func ToolbarSearchRim(p tokens.PlatformColors) color.NRGBA { return p.ToolbarSearchRim }

// ToolbarRim is the hairline a control standing in a toolbar band wears: the
// platform's separator flattened over the fill it is drawn on — but only
// where that hairline LIFTS the fill. Where it would darken it instead, the
// platform draws no hairline at all and this answers the zero value, which is
// no colour.
//
// MEASURED. Dark, finder-window-untinted-dark.png: a bordered toolbar control
// wears a 1 px rim reading #404040 over its #262626 fill, lighter than both
// the fill and the #1e1e1e band — a highlight that lifts the control's edge.
// The seam over that fill gives #3b3b3b, five of 255 short of the pixel,
// which is the miss this name carries. voicememos-window.png agrees on the
// toolbar search field: a 1 px #4d4d4d rim running the whole way round a
// #363636 fill — the left and right columns of the capsule carry it as well
// as the rows above and below, so it is the control's own edge and not the
// band's seam — where the seam over that fill gives #4a4a4a, three of 255
// short. Light, finder-window-light.png and voicememos-sidebar-light.png: the
// band steps straight to the fill with no darker row on any side, and the
// seam's own black would be an edge the platform does not draw.
func ToolbarRim(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	rim := vgcolor.Flatten(p.Separator, beneath)
	if rim.R <= beneath.R && rim.G <= beneath.G && rim.B <= beneath.B {
		return color.NRGBA{}
	}
	return rim
}

// Placeholder is the foreground a control's prompt is drawn in: the wording
// a text field or a picker's field trigger shows in the space its value will
// occupy, while there is no value there yet. It is the platform's
// placeholder text flattened over beneath — the control's own fill, which is
// [Fill] for a text field and the push button's for a picker's trigger.
func Placeholder(p tokens.PlatformColors, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(p.PlaceholderText, beneath)
}

// Faded is what a switched-off control paints one part of itself in: that
// part's own colour at the platform's measured disabled coverage
// ([tokens.DisabledCoverage]), landed on the surface the control stands on.
//
// One call covers a fill and an edge alike. An opaque fill — the push
// button's — comes back as that fill mixed toward the surface; a part that
// already carries a coverage — the seam a button's hairline is drawn in —
// comes back at that coverage scaled down and then landed, which is the same
// arithmetic. beneath is what the part is drawn on: the surface for the
// control's own fill, and the faded fill itself for anything drawn over it.
//
// Text does not go through here. A switched-off control's wording is
// [tokens.PlatformColors.DisabledControlText], the platform's own reduced
// coverage, which the Save dialog's two switched-off checkboxes read at
// without any further fading.
func Faded(part, beneath color.NRGBA) color.NRGBA {
	return vgcolor.Flatten(vgcolor.Fade(part, tokens.DisabledCoverage), beneath)
}
