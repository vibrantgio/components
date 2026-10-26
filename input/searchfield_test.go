package input_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/event"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	golden "github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/components/input"
	"github.com/vibrantgio/components/internal/control"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// clearTrail is where the xmark-circle mark's trailing edge stands: the field's
// outer edge less its trailing inset, which is the hairline the field is
// drawn with plus the measured [control.TextTrailDp] inside it.
const clearTrail = float32(1 + control.TextTrailDp)

// TestSearchFieldGolden records or diffs the search field's three states in
// both schemes: resting with its prompt, holding a query with the xmark-circle mark
// beside it, and focused.
//
// The three are what separate this control from the text field it is built
// on — the looking glass is in every one of them, and the xmark-circle mark is in
// exactly the one that has something to clear.
func TestSearchFieldGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	// Zero corner radius keeps the edges sharp, for the reason
	// TestTextFieldGolden gives.
	sharpRadius := tokens.RadiusScale{}
	cases := []struct {
		name     string
		platform tokens.PlatformColors
		state    input.RenderState
	}{
		{"searchfield-light-normal", tokens.PlatformLight, input.RenderState{}},
		{"searchfield-dark-normal", tokens.PlatformDark, input.RenderState{}},
		{"searchfield-light-typed", tokens.PlatformLight, input.RenderState{Text: "meeting notes"}},
		{"searchfield-dark-typed", tokens.PlatformDark, input.RenderState{Text: "meeting notes"}},
		{"searchfield-light-focused", tokens.PlatformLight, input.RenderState{Focused: true}},
		{"searchfield-dark-focused", tokens.PlatformDark, input.RenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := input.RenderSearch(
				shaper, "Search",
				tc.platform, tokens.Spacing, sharpRadius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				tc.state,
			)
			golden.Render(t, tc.name, size, w)
		})
	}

	// And the compact density, where the field loses height and the marks
	// inside it do not: the looking glass is drawn at the size the platform
	// draws it, which is the field's own number and not the density's icon
	// size.
	t.Run("searchfield-light-compact", func(t *testing.T) {
		w := input.RenderSearch(
			shaper, "Search",
			tokens.PlatformLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.SystemFont, tokens.Compact,
			input.RenderState{Text: "meeting notes"},
		)
		golden.Render(t, "searchfield-light-compact", size, w)
	})
}

// TestSearchFieldReservesBothSlots asserts the structure is paid for in
// layout and not merely painted: a search field spends the same width on its
// marks whether or not the xmark-circle mark is in its slot, so the text a reader is
// typing does not reflow the moment the field stops being empty.
//
// It is read off the drawn width the placeholder is given, which is the field
// width less both slots — the same number the editor is laid out in.
func TestSearchFieldReservesBothSlots(t *testing.T) {
	shaper := defaultShaper(t)
	const fieldW = 300

	measure := func(w layout.Widget) image.Point {
		var ops op.Ops
		gtx := layout.Context{
			Ops:         &ops,
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(fieldW, 60)),
		}
		return w(gtx).Size
	}

	empty := measure(input.RenderSearch(shaper, "Search", tokens.PlatformLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable, input.RenderState{}))
	typed := measure(input.RenderSearch(shaper, "Search", tokens.PlatformLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable, input.RenderState{Text: "q"}))
	plain := measure(input.Render(shaper, "Search", tokens.PlatformLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable, input.RenderState{}))

	if empty != typed {
		t.Errorf("an empty search field measured %v and one holding a query measured %v; the slots are reserved, so the two are one size", empty, typed)
	}
	if empty.X != plain.X {
		t.Errorf("a search field measured %v and a text field %v; the marks are spent out of the field's own width, not added to it", empty, plain)
	}
}

// TestSearchFieldClearMarkEmptiesTheField drives a press on the xmark-circle mark
// through a live search field holding a query, and checks the two things the
// mark promises: the field is empty afterwards, and the empty query is
// reported the way a keystroke reports any other — which is what dismisses a
// consumer's highlight along with the search that caused it.
func TestSearchFieldClearMarkEmptiesTheField(t *testing.T) {
	var changes []string
	type clearMsg struct{}
	w := materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
		Placeholder:  "Search",
		Seed:         "meeting notes",
		Shaper:       defaultShaper(t),
		ClearMessage: clearMsg{},
		OnChange:     func(_ layout.Context, s string) { changes = append(changes, s) },
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 120)

	dims := driveTextFieldFrame(w, ops, r, size)

	// The mark stands in the trailing slot, the field's trailing inset in
	// from its own edge — the hairline plus the measured
	// [control.TextTrailDp] — and centred on its height.
	pos := f32.Pt(float32(dims.Size.X)-clearTrail-float32(icon.Size(tokens.Comfortable))/2,
		float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)

	if len(changes) == 0 {
		t.Fatal("pressing the clear mark reported no change at all")
	}
	if got := changes[len(changes)-1]; got != "" {
		t.Errorf("the clear mark reported %q; it empties the field, so it reports the empty query", got)
	}

	// And the field is empty afterwards, read off the mark itself: a second
	// press has nothing to take back and reports nothing.
	n := len(changes)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)
	if len(changes) != n {
		t.Errorf("a press where the mark stood on an emptied field reported %v; the field was already empty and the mark is not drawn there any more", changes[n:])
	}
}

// TestSearchFieldClearMarkPostsTheEmptyChangeOnce presses the xmark-circle mark on a
// field holding a query and counts what is posted: the empty change is posted
// once on the frame of the press, the way Escape posts it.
func TestSearchFieldClearMarkPostsTheEmptyChangeOnce(t *testing.T) {
	type clearMsg struct{}
	var changes []string
	w := materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
		Placeholder:  "Search",
		Seed:         "meeting notes",
		Shaper:       defaultShaper(t),
		ClearMessage: clearMsg{},
		OnChange:     func(_ layout.Context, s string) { changes = append(changes, s) },
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 120)
	dims := driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)
	before := len(changes)

	pos := f32.Pt(float32(dims.Size.X)-clearTrail-float32(icon.Size(tokens.Comfortable))/2,
		float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)

	if got := changes[before:]; len(got) != 1 || got[0] != "" {
		t.Fatalf("pressing the clear mark posted %q; it empties the field and posts the empty change once", got)
	}
}

// TestSearchFieldClearsFromOutside drives SearchFieldProps.Clear: the query is
// taken back by the application rather than by the mark in the field, the
// empty query is reported the way every other edit is, and the field is empty
// afterwards.
func TestSearchFieldClearsFromOutside(t *testing.T) {
	var (
		changes []string
		clear   func()
	)
	w := materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
		Placeholder: "Search",
		Seed:        "meeting notes",
		Shaper:      defaultShaper(t),
		Clear:       func(c func()) { clear = c },
		OnChange:    func(_ layout.Context, s string) { changes = append(changes, s) },
	}))
	if clear == nil {
		t.Fatal("the field never handed the caller a way to empty it")
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(300, 120)
	dims := driveTextFieldFrame(w, ops, r, size)
	// The seeded field reports what it was seeded with on its first frame;
	// what this test is about is everything after that.
	seeded := len(changes)

	clear()
	driveTextFieldFrame(w, ops, r, size)
	if len(changes) == seeded {
		t.Fatal("emptying the field from outside reported no change at all")
	}
	if got := changes[len(changes)-1]; got != "" {
		t.Errorf("emptying the field from outside reported %q; the field is empty, so it reports the empty query", got)
	}
	emptied := len(changes)

	// And the field is empty, read off the mark: an emptied field draws no
	// clear mark, so a press where it stood takes nothing back.
	pos := f32.Pt(float32(dims.Size.X)-clearTrail-float32(icon.Size(tokens.Comfortable))/2,
		float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: pos, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, size)
	driveTextFieldFrame(w, ops, r, size)
	if len(changes) != emptied {
		t.Errorf("a press where the mark stood reported %v; the field was already empty and the mark is not drawn there", changes[emptied:])
	}
}

// onChrome paints the chrome material over the whole capture and lays w on
// it, which is the surface the chrome variant is measured against: a recess
// with no edge only reads as one against the chrome material it is cut into.
func onChrome(fill color.NRGBA, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, fill, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return w(gtx)
	}
}

// TestSearchFieldChromeVariantGolden records the variant a search field takes
// on chrome — a sidebar, a toolbar: the platform's flat recess, no edge, its
// ends fully rounded, standing on the chrome material rather than on a fill
// of the surface beneath it.
func TestSearchFieldChromeVariantGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	cases := []struct {
		name     string
		platform tokens.PlatformColors
		state    input.RenderState
	}{
		{"searchfield-chrome-light-normal", tokens.PlatformLight, input.RenderState{}},
		{"searchfield-chrome-dark-normal", tokens.PlatformDark, input.RenderState{}},
		{"searchfield-chrome-light-typed", tokens.PlatformLight, input.RenderState{Text: "meeting notes"}},
		{"searchfield-chrome-dark-typed", tokens.PlatformDark, input.RenderState{Text: "meeting notes"}},
		{"searchfield-chrome-light-focused", tokens.PlatformLight, input.RenderState{Focused: true}},
		{"searchfield-chrome-dark-focused", tokens.PlatformDark, input.RenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := tc.state
			st.Variant = input.Chrome
			st.Surface = tc.platform.SidebarMaterial
			w := input.RenderSearch(
				shaper, "Search",
				tc.platform, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				st,
			)
			golden.Render(t, tc.name, size, onChrome(tc.platform.SidebarMaterial, w))
		})
	}
}

// TestSearchFieldChromeVariantIsTheMeasuredRecess reads the three things the
// measurement settles off the drawn pixels: the recess carries the platform's
// measured fill, it carries no edge — the row above its middle is the chrome
// it stands on and not a hairline — and its ends are fully rounded, so the
// corner a half-height radius cuts away leaves the chrome showing.
func TestSearchFieldChromeVariantIsTheMeasuredRecess(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 40)
	for _, p := range []struct {
		name string
		col  tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(p.name, func(t *testing.T) {
			w := input.RenderSearch(
				shaper, "Search",
				p.col, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				input.RenderState{Variant: input.Chrome, Surface: p.col.SidebarMaterial},
			)
			img := golden.Capture(t, size, onChrome(p.col.SidebarMaterial, w))
			at := func(x, y int) color.NRGBA {
				r, g, b, a := img.At(x, y).RGBA()
				return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
			}
			// The field is drawn at the top of the capture, 27 px tall at
			// this density; x=200 is clear of the prompt and both marks.
			if got, want := at(200, 14), p.col.SidebarSearchFill; got != want {
				t.Errorf("the recess's fill = %v, want the measured %v", got, want)
			}
			if got, want := at(200, 0), p.col.SidebarSearchFill; got != want {
				t.Errorf("the recess's first row = %v, want the fill %v: the chrome variant draws no edge", got, want)
			}
			if got, want := at(0, 0), p.col.SidebarMaterial; got != want {
				t.Errorf("the recess's top-left corner = %v, want the chrome %v: the ends are fully rounded", got, want)
			}
		})
	}
}

// TestPromptCapBandIsOnTheFieldsCentreRow reads off the drawn pixels what the
// goldens lock but do not explain: the band the prompt's letters occupy is
// centred on the field's centre row, in both variants and in both schemes.
//
// The platform's own relation, measured the same way: in `mail-window.png`
// "Search" occupies y 21–31 in a field of y 8–43, a band centre of 26.5
// against the field's 26.0; in `system-settings-grouped-box-{light,dark}.png`
// it occupies y 70–80 in a field of y 61–88, 75.5 against 75.0. Both sit half
// a pixel low, which is where the rounding falls, and that is what this
// asserts of ours.
//
// The scan starts past the looking glass — the symbol's last pixel is at x=21
// on chrome and x=23 on a form — and runs to well beyond the prompt's last.
func TestPromptCapBandIsOnTheFieldsCentreRow(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 40)
	// The field draws at the top of the capture at its Comfortable measured
	// height: 26 on the content, 28 in a sidebar and 36 in a toolbar band.
	for _, tc := range []struct {
		name    string
		colors  tokens.PlatformColors
		variant input.Variant
		region  input.Region
		fieldH  int
	}{
		{"form-light", tokens.PlatformLight, input.Form, input.Sidebar, 26},
		{"form-dark", tokens.PlatformDark, input.Form, input.Sidebar, 26},
		{"chrome-light", tokens.PlatformLight, input.Chrome, input.Sidebar, 28},
		{"chrome-dark", tokens.PlatformDark, input.Chrome, input.Sidebar, 28},
		{"toolbar-light", tokens.PlatformLight, input.Chrome, input.Toolbar, 36},
		{"toolbar-dark", tokens.PlatformDark, input.Chrome, input.Toolbar, 36},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fieldH := tc.fieldH
			st := input.RenderState{Variant: tc.variant, Region: tc.region}
			stands := tc.colors.WindowBackground
			if tc.variant == input.Chrome {
				st.Surface = tc.colors.SidebarMaterial
				stands = tc.colors.SidebarMaterial
			}
			w := input.RenderSearch(
				shaper, "Search",
				tc.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				st,
			)
			img := golden.Capture(t, size, onChrome(stands, w))
			// x=200 is clear of the prompt and both marks, so it reads the
			// field's own fill; the prompt is what leaves it.
			fill := img.RGBAAt(200, fieldH/2)
			top, bottom := -1, -1
			for x := 25; x < 200; x++ {
				for y := 1; y < fieldH-1; y++ {
					if img.RGBAAt(x, y) == fill {
						continue
					}
					if top < 0 || y < top {
						top = y
					}
					if y > bottom {
						bottom = y
					}
				}
			}
			if top < 0 {
				t.Fatal("no prompt found past the looking glass; this measures nothing")
			}
			// Both centres in half pixels: the band's is top+bottom+1 and the
			// field's is fieldH, so the platform's half-pixel-low relation is
			// a difference of exactly one.
			// A band and a field of the same parity centre exactly; otherwise
			// the rounding falls a half pixel low.
			want := (bottom - top + 1 + fieldH) % 2
			if got := (top + bottom + 1) - fieldH; got != want {
				t.Errorf("the prompt's band is rows %d–%d in a %d px field, which is %v half pixels off the centre row, want %d: centred, the rounding falling half a pixel low",
					top, bottom, fieldH, got, want)
			}
		})
	}
}

// TestLeadingInsetIsMeasuredPerVariant reads off the drawn pixels the two
// numbers that place the looking glass and the prompt after it, per variant:
// the field's inner edge to the symbol's first pixel, and the clear columns
// between the symbol's last pixel and the prompt's first.
//
// MEASURED, system-settings-grouped-box-{light,dark}.png: the recess carries
// no edge, so its inner edge is its own at x=18, and the symbol's first pixel
// is at x=27 — 9 px in; the symbol's last pixel is at x=41 and the prompt's
// first at x=47, five clear columns. MEASURED, mail-window.png: the toolbar
// field's stroke is at x=867 and its fill begins at x=868, with the symbol's
// first pixel at x=878 — 10 px in; the symbol's last pixel is at x=890 and the
// prompt's first at x=899, eight clear columns. MEASURED,
// systemsettings-add-input-source-search-{light,dark}.png: the field on
// content's edge column is at x=19, its inner edge at x=20, and the symbol's
// first pixel is at x=27 — 7 px in; the symbol's last pixel is at x=40 and
// the prompt's first at x=45, four clear columns.
//
// The gap is a count of clear columns, not a distance, which is what lets it
// be read on both sides: this library draws the symbol a third of a column
// wider than either capture holds, and a column the drawing ends inside is a
// column it covers.
func TestLeadingInsetIsMeasuredPerVariant(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 40)
	for _, tc := range []struct {
		name      string
		variant   input.Variant
		region    input.Region
		fieldH    int // the height the variant draws at, Comfortable
		innerEdge int // the field's outer edge to its inner one
		inset     int
		gap       int
	}{
		{"form", input.Form, input.Sidebar, 27, 1, 7, 4},
		{"chrome-sidebar", input.Chrome, input.Sidebar, 27, 0, 9, 5},
		// The toolbar recess spends the rim's column before the inset, the
		// way the form field spends its hairline, so its inner edge is one
		// in even in the light appearance where no rim is drawn. It is drawn
		// at the density's toolbar control height, so the band the symbol and
		// the prompt occupy sits lower.
		{"chrome-toolbar", input.Chrome, input.Toolbar, 36, 1, 10, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tokens.PlatformLight
			st := input.RenderState{Variant: tc.variant, Region: tc.region}
			stands := p.WindowBackground
			if tc.variant == input.Chrome {
				st.Surface = p.SidebarMaterial
				stands = p.SidebarMaterial
			}
			w := input.RenderSearch(
				shaper, "Search",
				p, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				st,
			)
			img := golden.Capture(t, size, onChrome(stands, w))
			// x=200 is clear of the prompt and both marks; the sixteen rows
			// either side of the field's centre hold the whole symbol — the
			// magnifier's handle reaches the lowest of them — and stay clear
			// of the field's own corners.
			mid := tc.fieldH / 2
			fill := img.RGBAAt(200, mid)
			// A column is covered when one of its pixels differs from the fill
			// by more than two levels on a channel. The ring's antialiasing
			// leaves a one-level fringe in the column before its first pixel at
			// some vertical positions, and no reader sees a one-level fringe.
			off := func(a, b uint8) bool { return int(a) > int(b)+2 || int(b) > int(a)+2 }
			covered := func(x int) bool {
				for y := mid - 8; y <= mid+8; y++ {
					c := img.RGBAAt(x, y)
					if off(c.R, fill.R) || off(c.G, fill.G) || off(c.B, fill.B) {
						return true
					}
				}
				return false
			}
			first, clear, prompt := -1, -1, -1
			for x := 5; x < 200; x++ {
				switch {
				case first < 0:
					if covered(x) {
						first = x
					}
				case clear < 0:
					if !covered(x) {
						clear = x
					}
				case prompt < 0:
					if covered(x) {
						prompt = x
					}
				}
			}
			if first < 0 {
				t.Fatal("no looking glass found inside the field; this measures nothing")
			}
			if prompt < 0 {
				t.Fatal("no prompt found past the looking glass; the gap measures nothing")
			}
			if got := first - tc.innerEdge; got != tc.inset {
				t.Errorf("the symbol's first pixel is %d px in from the field's inner edge, want the measured %d", got, tc.inset)
			}
			if got := prompt - clear; got != tc.gap {
				t.Errorf("the symbol's last pixel is at x=%d and the prompt's first at x=%d, %d clear columns; want the measured %d",
					clear-1, prompt, got, tc.gap)
			}
		})
	}
}

// TestSearchFieldToolbarRecessGolden records the recess a search field takes
// in a toolbar band: the density's toolbar control height, the platform's
// toolbar search fill, its ends fully rounded, and in the dark appearance the
// highlight rim every bordered control in a dark toolbar has.
func TestSearchFieldToolbarRecessGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)

	cases := []struct {
		name     string
		platform tokens.PlatformColors
		state    input.RenderState
	}{
		{"searchfield-toolbar-light-normal", tokens.PlatformLight, input.RenderState{}},
		{"searchfield-toolbar-dark-normal", tokens.PlatformDark, input.RenderState{}},
		{"searchfield-toolbar-light-typed", tokens.PlatformLight, input.RenderState{Text: "meeting notes"}},
		{"searchfield-toolbar-dark-typed", tokens.PlatformDark, input.RenderState{Text: "meeting notes"}},
		{"searchfield-toolbar-light-focused", tokens.PlatformLight, input.RenderState{Focused: true}},
		{"searchfield-toolbar-dark-focused", tokens.PlatformDark, input.RenderState{Focused: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := tc.state
			st.Variant = input.Chrome
			st.Region = input.Toolbar
			st.Surface = tc.platform.SidebarMaterial
			w := input.RenderSearch(
				shaper, "Search",
				tc.platform, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				st,
			)
			golden.Render(t, tc.name, size, onChrome(tc.platform.SidebarMaterial, w))
		})
	}
}

// TestSearchFieldToolbarRecessIsTheMeasuredControl reads off the drawn pixels
// what the toolbar recess was measured to be, and what tells it from the
// sidebar's.
//
// MEASURED, voicememos-window.png and voicememos-sidebar-light.png, the
// search field at the trailing end of a frontmost Voice Memos toolbar: it
// spans y 8–43 dark and y 46–81 light — 36 px, the toolbar control height —
// where the sidebar recess in system-settings-grouped-box-{light,dark}.png
// spans 28. Its interior is #363636 dark and #e8e8e8 light. Dark it has a
// 1 px #4d4d4d rim the whole way round, lighter than both its fill and the
// #1e1e1e band; light the band steps straight to the fill with no rim on any
// side.
func TestSearchFieldToolbarRecessIsTheMeasuredControl(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	const fieldH = 36
	for _, p := range []struct {
		name string
		col  tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(p.name, func(t *testing.T) {
			w := input.RenderSearch(
				shaper, "Search",
				p.col, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				input.RenderState{Variant: input.Chrome, Region: input.Toolbar, Surface: p.col.SidebarMaterial},
			)
			img := golden.Capture(t, size, onChrome(p.col.SidebarMaterial, w))
			at := func(x, y int) color.NRGBA {
				r, g, b, a := img.At(x, y).RGBA()
				return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
			}
			fill := control.ToolbarRecess(p.col)
			// x=200 is clear of the prompt and both marks.
			if got := at(200, fieldH/2); got != fill {
				t.Errorf("the recess's fill = %v, want the measured %v", got, fill)
			}
			if got := at(200, fieldH/2); got == p.col.SidebarSearchFill && p.name == "dark" {
				t.Errorf("the recess's fill = %v, the sidebar's; the two were measured apart", got)
			}
			// The last row the field covers and the first it does not: the
			// height is the toolbar control's and not the field's 28.
			if got := at(200, fieldH-1); got == p.col.SidebarMaterial {
				t.Errorf("row %d reads the chrome %v; the recess is drawn shorter than the measured %d px",
					fieldH-1, got, fieldH)
			}
			// The row under it is the band carrying the shadow this
			// control casts on it, which is the band and not the fill:
			// MEASURED, voicememos-sidebar-light.png, where the row under
			// the recess reads 244 on a #ffffff band. A row still reading
			// the fill would be a recess drawn taller than the measured 36.
			band := vgcolor.Flatten(p.col.ToolbarControlShadow.Peak, p.col.SidebarMaterial)
			if got := at(200, fieldH); got != band {
				t.Errorf("row %d reads %v, want the chrome under this control's shadow %v; the recess is drawn taller than the measured %d px",
					fieldH, got, band, fieldH)
			}
			// The rim: the platform draws one in the dark appearance only,
			// and where it draws none the field's first row is its own fill.
			// It is the recess's own measured value and not the separator
			// over the fill, which falls three of 255 short of the pixel.
			rim := control.ToolbarSearchRim(p.col)
			switch p.name {
			case "dark":
				if rim.A == 0 {
					t.Fatal("the dark toolbar rim answers no colour; the capture holds one")
				}
				if want := (color.NRGBA{R: 0x4d, G: 0x4d, B: 0x4d, A: 0xff}); rim != want {
					t.Errorf("the dark toolbar search rim = %v, want the measured %v", rim, want)
				}
				if rim == vgcolor.Flatten(p.col.Separator, fill) {
					t.Error("the rim is the separator flattened over the fill; the capture reads three of 255 lighter")
				}
				if got := at(200, 0); got != rim {
					t.Errorf("the recess's first row = %v, want the rim %v", got, rim)
				}
				if got := at(200, 1); got != fill {
					t.Errorf("the row under the rim = %v, want the fill %v: the rim is one pixel", got, fill)
				}
			case "light":
				if rim.A != 0 {
					t.Errorf("the light toolbar rim answers %v; the platform draws none", rim)
				}
				if got := at(200, 0); got != fill {
					t.Errorf("the recess's first row = %v, want the fill %v: the platform draws no light rim", got, fill)
				}
			}
			// The ends are fully rounded, so the corner leaves the chrome. The
			// shadow's own rounded corner reaches that pixel, so it reads the
			// chrome at most the shadow's peak darker, and never the fill.
			chrome := p.col.SidebarMaterial
			if got := at(0, 0); got == fill || got.R > chrome.R ||
				int(chrome.R)-int(got.R) > int(p.col.ToolbarControlShadow.Peak.A) {
				t.Errorf("the recess's top-left corner = %v, want the chrome %v under at most the shadow's peak: the ends are fully rounded",
					got, chrome)
			}
		})
	}
}

// TestTheCountStandsInsideTheFieldAtItsTrailingEnd reads what a search field
// reports about its query off the drawn pixels: it stands INSIDE the field,
// between the text and the xmark-circle mark, and it moves neither.
//
// MEASURED, reference/macos/voicememos-multi-folder-search-2026-09-18.png —
// a frontmost Voice Memos window whose toolbar field holds a query: the
// field's fill runs x 700–1023 and the xmark-circle mark's disc x 996–1009, so the
// mark stands fourteen clear of the field's trailing edge against the
// magnifier's thirteen at its leading one. No stored capture holds a field
// that reports a count, so where the count stands is composed from those two
// readings: it takes the trailing end, the xmark-circle mark keeps its own inset,
// and the clear space between them is the eight the platform leaves between
// the magnifier and the prompt at the other end (mail-window.png).
func TestTheCountStandsInsideTheFieldAtItsTrailingEnd(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 60)
	p := tokens.PlatformLight
	field := func(count string) *image.RGBA {
		w := input.RenderSearch(shaper, "Search", p, tokens.Spacing, tokens.Radius,
			tokens.DefaultTypography.SystemFont, tokens.Comfortable, input.RenderState{
				Text: "margin", Count: count,
				Variant: input.Chrome, Region: input.Toolbar, Surface: p.SidebarMaterial,
			})
		return golden.Capture(t, size, onChrome(p.SidebarMaterial, w))
	}
	// The columns carrying paint, read on the field's own centre row against
	// the recess's fill.
	painted := func(img *image.RGBA) []int {
		var xs []int
		fill := p.ToolbarSearchFill
		for x := 0; x < size.X; x++ {
			if c := img.RGBAAt(x, 18); c.R != fill.R || c.G != fill.G || c.B != fill.B {
				xs = append(xs, x)
			}
		}
		return xs
	}
	bare, counted := painted(field("")), painted(field("2 of 3"))
	if len(bare) == 0 || len(counted) == 0 {
		t.Fatal("the field drew nothing on its own centre row")
	}
	// The clear mark is the last paint in the field either way, and the count
	// is spent out of the text's width rather than added to the field's, so
	// the mark does not move.
	if bare[len(bare)-1] != counted[len(counted)-1] {
		t.Errorf("the clear mark ends at x=%d without a count and x=%d with one; the count is spent out of the text's width",
			bare[len(bare)-1], counted[len(counted)-1])
	}
	if bare[0] != counted[0] {
		t.Errorf("the magnifier begins at x=%d without a count and x=%d with one; a count at the trailing end moves nothing at the leading one",
			bare[0], counted[0])
	}
	// What the count adds is paint the bare field does not have, and all of it
	// stands inside the field, leading of the xmark-circle mark.
	added := 0
	seen := map[int]bool{}
	for _, x := range bare {
		seen[x] = true
	}
	for _, x := range counted {
		if !seen[x] {
			added++
			if x >= counted[len(counted)-1] {
				t.Errorf("the count paints at x=%d, at or past the clear mark; it stands leading of it", x)
			}
		}
	}
	if added == 0 {
		t.Error("a field given a count drew nothing the same field without one did not; the count is not reaching the drawing")
	}
}

// escapeTestSize is the window the Escape tests lay the field out in.
var escapeTestSize = image.Pt(300, 120)

// focusSearchField presses in the middle of the field so it takes the
// keyboard, the way a reader starts typing a query.
func focusSearchField(w layout.Widget, ops *op.Ops, r *gioinput.Router) {
	dims := driveTextFieldFrame(w, ops, r, escapeTestSize)
	mid := f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)/2)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: mid, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: mid, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
	)
	driveTextFieldFrame(w, ops, r, escapeTestSize)
}

// TestSearchFieldEscapeEmptiesTheField drives Escape in a field with text
// typed into it, the Escape on the frame right after the typing: the field
// empties, posts the empty change once, and the keyboard stays in the field.
// A second Escape, on the field now empty, posts nothing.
func TestSearchFieldEscapeEmptiesTheField(t *testing.T) {
	var (
		changes []string
		tag     event.Tag
	)
	w := materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
		Placeholder: "Search",
		Shaper:      defaultShaper(t),
		FocusTag:    func(t event.Tag) { tag = t },
		OnChange:    func(_ layout.Context, s string) { changes = append(changes, s) },
	}))

	r := new(gioinput.Router)
	ops := new(op.Ops)
	focusSearchField(w, ops, r)
	if !r.Source().Focused(tag) {
		t.Fatal("pressing in the field did not give it the keyboard")
	}
	r.Queue(key.EditEvent{Text: "meeting notes"})
	driveTextFieldFrame(w, ops, r, escapeTestSize)
	if n := len(changes); n == 0 || changes[n-1] != "meeting notes" {
		t.Fatalf("typing into the field posted %q, want the typed text last", changes)
	}
	before := len(changes)

	r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	driveTextFieldFrame(w, ops, r, escapeTestSize)
	driveTextFieldFrame(w, ops, r, escapeTestSize)

	if got := changes[before:]; len(got) != 1 || got[0] != "" {
		t.Fatalf("Escape in a field with text posted %q; it empties the field and posts the empty change once", got)
	}
	if !r.Source().Focused(tag) {
		t.Error("Escape took the keyboard out of the field; it empties the field and the keyboard stays in it")
	}

	emptied := len(changes)
	r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	driveTextFieldFrame(w, ops, r, escapeTestSize)
	driveTextFieldFrame(w, ops, r, escapeTestSize)
	if len(changes) != emptied {
		t.Errorf("Escape in an empty field posted %q; there is nothing to empty", changes[emptied:])
	}
}

// TestSearchFieldEscapeOnAnEmptyFieldIsNotTaken drives Escape in a focused
// field with nothing typed: the field posts nothing and does not take the
// key, so a handler around the field receives it.
func TestSearchFieldEscapeOnAnEmptyFieldIsNotTaken(t *testing.T) {
	var (
		changes []string
		tag     event.Tag
	)
	field := materialize(t, input.SearchField(rx.Of(theme.Default()), input.SearchFieldProps{
		Placeholder: "Search",
		Shaper:      defaultShaper(t),
		FocusTag:    func(t event.Tag) { tag = t },
		OnChange:    func(_ layout.Context, s string) { changes = append(changes, s) },
	}))

	// The handler around the field asks for Escape after the field has laid
	// out, so the field is offered the key first, and the handler receives
	// it only when the field does not take it.
	received := 0
	w := func(gtx layout.Context) layout.Dimensions {
		dims := field(gtx)
		for {
			ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
			if !ok {
				break
			}
			if ke, isKey := ev.(key.Event); isKey && ke.State == key.Press {
				received++
			}
		}
		return dims
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	focusSearchField(w, ops, r)
	if !r.Source().Focused(tag) {
		t.Fatal("pressing in the field did not give it the keyboard")
	}
	before := len(changes)

	r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	driveTextFieldFrame(w, ops, r, escapeTestSize)
	driveTextFieldFrame(w, ops, r, escapeTestSize)

	if len(changes) != before {
		t.Errorf("Escape in an empty field posted %q; there is nothing to empty", changes[before:])
	}
	if received != 1 {
		t.Errorf("the handler around the field received Escape %d times; an empty field does not take the key, so it is received once", received)
	}
}

// TestSearchFieldOnContentIsTheMeasuredCapsule reads off the drawn pixels the
// field on content's box: the form's hairline around the surface the field
// stands on, its ends fully rounded, where a plain text field on the same
// form keeps the radius scale's corner.
//
// MEASURED, systemsettings-add-input-source-search-{light,dark}.png: the
// field's interior is the sheet's own fill, its edge is one row of
// #f3f3f3 light, and a circular fit to the edge at its leading end gives a
// centreline radius of 12.34 against a 26 px height — half the height. The
// edge's top row therefore starts about half the height in from the field's
// leading end, where a corner of the radius scale's 6 starts it six in.
func TestSearchFieldOnContentIsTheMeasuredCapsule(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 40)
	const fieldH = 27
	for _, p := range []struct {
		name string
		col  tokens.PlatformColors
	}{{"light", tokens.PlatformLight}, {"dark", tokens.PlatformDark}} {
		t.Run(p.name, func(t *testing.T) {
			stands := p.col.WindowBackground
			search := golden.Capture(t, size, onChrome(stands, input.RenderSearch(
				shaper, "Search",
				p.col, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				input.RenderState{Variant: input.Form},
			)))
			plain := golden.Capture(t, size, onChrome(stands, input.Render(
				shaper, "Name",
				p.col, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.SystemFont, tokens.Comfortable,
				input.RenderState{Variant: input.Form},
			)))
			at := func(img *image.RGBA, x, y int) color.NRGBA {
				c := img.RGBAAt(x, y)
				return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
			}
			if got, want := at(search, 200, 0), p.col.FieldEdge; got != want {
				t.Errorf("the field's top row at x=200 = %v, want the form's hairline %v", got, want)
			}
			if got, want := at(search, 200, fieldH/2), stands; got != want {
				t.Errorf("the field's interior at x=200 = %v, want the surface it stands on %v", got, want)
			}
			// x=7 is past a 6 px corner and well inside a 13.5 px one.
			if got := at(search, 7, 0); got != stands {
				t.Errorf("the search field's top row at x=7 = %v, want the surface %v: its ends are fully rounded", got, stands)
			}
			if got := at(plain, 7, 0); got != p.col.FieldEdge {
				t.Errorf("the text field's top row at x=7 = %v, want its hairline %v: a plain field keeps the radius scale's corner", got, p.col.FieldEdge)
			}
			// The leading end's middle row is the edge itself.
			if got := at(search, 0, fieldH/2); got != p.col.FieldEdge {
				t.Errorf("the search field's leading column at its middle row = %v, want the hairline %v", got, p.col.FieldEdge)
			}
		})
	}
}
