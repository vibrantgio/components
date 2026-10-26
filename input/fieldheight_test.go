package input_test

import (
	"image"
	"testing"

	gioinput "gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// TestAFieldsTextIsSetInTheSystemFontRole pins the role a live field takes
// its text from. The theme here sets the System font role's line box at 40 dp
// and leaves Body at its own: a field set in the System font role draws 40
// tall, over every measured floor, and one set in Body draws its floor.
func TestAFieldsTextIsSetInTheSystemFontRole(t *testing.T) {
	const tall = 40
	typ := tokens.DefaultTypography
	typ.SystemFont.LineHeight = tall
	th := theme.Default()
	th.Typography = rx.Of(typ)

	other := tokens.DefaultTypography
	other.Body.LineHeight = tall
	thBody := theme.Default()
	thBody.Typography = rx.Of(other)

	fields := []struct {
		name string
		make func(theme.Theme) rx.Observable[layout.Widget]
	}{
		{"text field", func(th theme.Theme) rx.Observable[layout.Widget] {
			return input.TextField(rx.Of(th), input.TextFieldProps{Placeholder: "Name", Shaper: defaultShaper(t)})
		}},
		{"search field on the content", func(th theme.Theme) rx.Observable[layout.Widget] {
			return input.SearchField(rx.Of(th), input.SearchFieldProps{Placeholder: "Search", Shaper: defaultShaper(t)})
		}},
		{"search field in a sidebar", func(th theme.Theme) rx.Observable[layout.Widget] {
			return input.SearchField(rx.Of(th), input.SearchFieldProps{Placeholder: "Search", Variant: input.Chrome, Shaper: defaultShaper(t)})
		}},
	}
	for _, f := range fields {
		got := driveTextFieldFrame(materialize(t, f.make(th)), new(op.Ops), new(gioinput.Router), image.Pt(300, 120)).Size.Y
		if got != tall {
			t.Errorf("%s: drawn %d px with the System font role's line box at %d; its text is not set in that role", f.name, got, tall)
		}
		if got := driveTextFieldFrame(materialize(t, f.make(thBody)), new(op.Ops), new(gioinput.Router), image.Pt(300, 120)).Size.Y; got == tall {
			t.Errorf("%s: drawn %d px with Body's line box at %d; its text is set in Body", f.name, got, tall)
		}
	}
}

// TestEachFieldIsItsMeasuredHeight pins each field's Comfortable height to
// its row in organisation/reference/macos/controls.md. The System font
// role's 16 dp line is under every floor, so each field draws its floor.
func TestEachFieldIsItsMeasuredHeight(t *testing.T) {
	gtx := func() layout.Context {
		return layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(300, 120)),
			Ops:         new(op.Ops),
		}
	}
	shaper := defaultShaper(t)
	body := tokens.DefaultTypography.SystemFont
	render := func(search bool, s input.RenderState) int {
		if search {
			return input.RenderSearch(shaper, "Search", tokens.PlatformLight, tokens.Spacing, tokens.Radius, body, tokens.Comfortable, s)(gtx()).Size.Y
		}
		return input.Render(shaper, "Name", tokens.PlatformLight, tokens.Spacing, tokens.Radius, body, tokens.Comfortable, s)(gtx()).Size.Y
	}
	for _, tc := range []struct {
		name   string
		search bool
		state  input.RenderState
		row    int
		from   string
	}{
		{"text field in a form", false, input.RenderState{}, 27, `save-dialog-{light,dark}.png, the "Tags:" field, y 243–269`},
		{"search field on the content", true, input.RenderState{Variant: input.Form}, 26, "systemsettings-add-input-source-search-{light,dark}.png, y 347–372"},
		{"search field in a sidebar", true, input.RenderState{Variant: input.Chrome, Region: input.Sidebar}, 28, "system-settings-grouped-box-{light,dark}.png, y 61–88"},
		{"search field in a toolbar", true, input.RenderState{Variant: input.Chrome, Region: input.Toolbar}, 36, "voicememos-window.png, y 8–43"},
	} {
		if got := render(tc.search, tc.state); got != tc.row {
			t.Errorf("%s = %d px, want its measured row %d (%s)", tc.name, got, tc.row, tc.from)
		}
	}
}
