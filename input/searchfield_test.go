package input_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
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
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// TestSearchFieldGolden records or diffs the search field's three states in
// both schemes: resting with its prompt, holding a query with the clear mark
// beside it, and focused.
//
// The three are what separate this control from the text field it is built
// on — the looking glass is in every one of them, and the clear mark is in
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
				tc.platform, tokens.Spacing, sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
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
			tokens.PlatformLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.BodyLarge, tokens.Compact,
			input.RenderState{Text: "meeting notes"},
		)
		golden.Render(t, "searchfield-light-compact", size, w)
	})
}

// TestSearchFieldReservesBothSlots asserts the structure is paid for in
// layout and not merely painted: a search field spends the same width on its
// marks whether or not the clear mark is in its slot, so the text a reader is
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
		tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{}))
	typed := measure(input.RenderSearch(shaper, "Search", tokens.PlatformLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{Text: "q"}))
	plain := measure(input.Render(shaper, "Search", tokens.PlatformLight, tokens.Spacing,
		tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable, input.RenderState{}))

	if empty != typed {
		t.Errorf("an empty search field measured %v and one holding a query measured %v; the slots are reserved, so the two are one size", empty, typed)
	}
	if empty != plain {
		t.Errorf("a search field measured %v and a text field %v; the marks are spent out of the field's own width, not added to it", empty, plain)
	}
}

// TestSearchFieldClearMarkEmptiesTheField drives a press on the clear mark
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

	// The mark stands in the trailing slot, one horizontal pad in from the
	// field's own edge and centred on its height.
	pos := f32.Pt(float32(dims.Size.X)-tokens.Spacing.S3-float32(icon.Size(tokens.Comfortable))/2,
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
	pos := f32.Pt(float32(dims.Size.X)-tokens.Spacing.S3-float32(icon.Size(tokens.Comfortable))/2,
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
				tc.platform, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
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
				p.col, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				input.RenderState{Variant: input.Chrome, Surface: p.col.SidebarMaterial},
			)
			img := golden.Capture(t, size, onChrome(p.col.SidebarMaterial, w))
			at := func(x, y int) color.NRGBA {
				r, g, b, a := img.At(x, y).RGBA()
				return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
			}
			// The field is drawn at the top of the capture, 28 px tall at
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
// The scan starts past the looking glass — the glyph's last pixel is at x=21
// on chrome and x=23 on a form — and runs to well beyond the prompt's last.
func TestPromptCapBandIsOnTheFieldsCentreRow(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 40)
	// The field draws at the top of the capture, and Comfortable draws it
	// 28 px tall.
	const fieldH = 28
	for _, tc := range []struct {
		name    string
		colors  tokens.PlatformColors
		variant input.Variant
	}{
		{"form-light", tokens.PlatformLight, input.Form},
		{"form-dark", tokens.PlatformDark, input.Form},
		{"chrome-light", tokens.PlatformLight, input.Chrome},
		{"chrome-dark", tokens.PlatformDark, input.Chrome},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := input.RenderState{Variant: tc.variant}
			stands := tc.colors.WindowBackground
			if tc.variant == input.Chrome {
				st.Surface = tc.colors.SidebarMaterial
				stands = tc.colors.SidebarMaterial
			}
			w := input.RenderSearch(
				shaper, "Search",
				tc.colors, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
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
			if got := (top + bottom + 1) - fieldH; got != 1 {
				t.Errorf("the prompt's band is rows %d–%d in a %d px field, which is %v half pixels off the centre row; the platform's is one low",
					top, bottom, fieldH, got)
			}
		})
	}
}

// TestLeadingInsetIsMeasuredPerVariant reads off the drawn pixels the number
// that places the looking glass, per variant: the field's inner edge to the
// glyph's first pixel.
//
// MEASURED, system-settings-grouped-box-{light,dark}.png: the recess carries
// no edge, so its inner edge is its own at x=18, and the glyph's first pixel
// is at x=27 — 9 px in. MEASURED, mail-window.png: the toolbar field's stroke
// is at x=867 and its fill begins at x=868, with the glyph's first pixel at
// x=878 — 10 px in.
//
// The gap after the glyph is not read here. Both captures measure it from the
// glyph's last pixel, and this library draws that glyph a third of a column
// wider than either capture holds, so the count of clear columns is not the
// same measurement on both sides. The goldens hold the drawn result.
func TestLeadingInsetIsMeasuredPerVariant(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 40)
	for _, tc := range []struct {
		name      string
		variant   input.Variant
		innerEdge int // the field's outer edge to its inner one
		inset     int
	}{
		{"form", input.Form, 1, 10},
		{"chrome", input.Chrome, 0, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tokens.PlatformLight
			st := input.RenderState{Variant: tc.variant}
			stands := p.WindowBackground
			if tc.variant == input.Chrome {
				st.Surface = p.SidebarMaterial
				stands = p.SidebarMaterial
			}
			w := input.RenderSearch(
				shaper, "Search",
				p, tokens.Spacing, tokens.Radius, tokens.DefaultTypography.BodyLarge, tokens.Comfortable,
				st,
			)
			img := golden.Capture(t, size, onChrome(stands, w))
			// x=200 is clear of the prompt and both marks; rows 6 to 22 hold
			// the glyph and stay clear of the field's corners.
			fill := img.RGBAAt(200, 14)
			first := -1
			for x := 5; x < 200 && first < 0; x++ {
				for y := 6; y <= 22; y++ {
					if img.RGBAAt(x, y) != fill {
						first = x
						break
					}
				}
			}
			if first < 0 {
				t.Fatal("no looking glass found inside the field; this measures nothing")
			}
			if got := first - tc.innerEdge; got != tc.inset {
				t.Errorf("the glyph's first pixel is %d px in from the field's inner edge, want the measured %d", got, tc.inset)
			}
		})
	}
}
