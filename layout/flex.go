package layout

import gio "gioui.org/layout"

// Row lays its children out as Rigid in a horizontal Flex row, left-to-right.
// For mixed Rigid/Flexed children use gioui.org/layout.Flex directly.
func Row(gtx gio.Context, children ...gio.Widget) gio.Dimensions {
	return gio.Flex{}.Layout(gtx, rigid(children)...)
}

// Col lays its children out as Rigid in a vertical Flex column, top-to-bottom.
// For mixed Rigid/Flexed children use gioui.org/layout.Flex directly.
func Col(gtx gio.Context, children ...gio.Widget) gio.Dimensions {
	return gio.Flex{Axis: gio.Vertical}.Layout(gtx, rigid(children)...)
}

func rigid(ws []gio.Widget) []gio.FlexChild {
	cs := make([]gio.FlexChild, len(ws))
	for i, w := range ws {
		cs[i] = gio.Rigid(w)
	}
	return cs
}
