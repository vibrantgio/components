package richtext_test

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/font"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	golden "github.com/vibrantgio/prism/golden"
	"github.com/vibrantgio/prism/richtext"
	"github.com/vibrantgio/spectrum/tokens"
)

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper. See
// AGENTS.md.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// mixedSpans is the canonical test paragraph: regular, bold, italic, and
// monospace spans plus one hyperlink (link index 0).
func mixedSpans() []richtext.SpanStyle {
	return []richtext.SpanStyle{
		{Content: "The quick "},
		{Content: "brown", Weight: font.Bold},
		{Content: " fox "},
		{Content: "jumps", Style: font.Italic},
		{Content: " over "},
		{Content: "the lazy dog", Typeface: "Go Mono, monospace"},
		{Content: " via "},
		{Content: "a link", URL: "https://gioui.org"},
		{Content: " home."},
	}
}

// ---- Golden-image tests ----

// TestParagraphGolden records or diffs the mixed-style paragraph (regular,
// bold, italic, mono, link) wrapped over multiple lines, in light and dark
// token themes.
func TestParagraphGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	cases := []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"paragraph-light", tokens.DefaultLight},
		{"paragraph-dark", tokens.DefaultDark},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			style := richtext.FromTokens(tc.colors, tokens.DefaultTypography.BodyLarge)
			w := richtext.Render(shaper, style, mixedSpans(), richtext.Idle())
			golden.Render(t, tc.name, size, w)
		})
	}
}

// TestLinkStateGolden records or diffs the link interaction treatments: the
// paragraph's only link hovered (blended colour) and focused (focus ring).
func TestLinkStateGolden(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	cases := []struct {
		name  string
		state richtext.RenderState
	}{
		{"link-hovered", richtext.RenderState{HoveredLink: 0, FocusedLink: richtext.NoLink}},
		{"link-focused", richtext.RenderState{HoveredLink: richtext.NoLink, FocusedLink: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := richtext.Render(shaper, style, mixedSpans(), tc.state)
			golden.Render(t, tc.name, size, w)
		})
	}
}

// TestHoveredLinkIsVisuallyDistinct confirms the hovered link treatment
// produces different pixels from the idle paragraph.
func TestHoveredLinkIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	idle := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(), richtext.Idle()))
	hovered := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(),
		richtext.RenderState{HoveredLink: 0, FocusedLink: richtext.NoLink}))
	if idle == nil || hovered == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(idle, hovered); n == 0 {
		t.Error("hovered and idle paragraphs render identically; expected hover treatment pixels to differ")
	}
}

// TestFocusedLinkIsVisuallyDistinct confirms the focused link renders a
// visible focus ring: different pixels from the idle paragraph.
func TestFocusedLinkIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	idle := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(), richtext.Idle()))
	focused := golden.Capture(t, size, richtext.Render(shaper, style, mixedSpans(),
		richtext.RenderState{HoveredLink: richtext.NoLink, FocusedLink: 0}))
	if idle == nil || focused == nil {
		return
	}
	if n := golden.PixelDiff(idle, focused); n == 0 {
		t.Error("focused and idle paragraphs render identically; expected focus ring pixels to differ")
	}
}

// TestStrikethroughIsVisuallyDistinct confirms a strikethrough span renders a
// visible line through the text: different pixels from the same span without
// the decoration.
func TestStrikethroughIsVisuallyDistinct(t *testing.T) {
	shaper := defaultShaper(t)
	size := image.Pt(300, 100)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	plain := golden.Capture(t, size, richtext.Render(shaper, style,
		[]richtext.SpanStyle{{Content: "deleted text"}}, richtext.Idle()))
	struck := golden.Capture(t, size, richtext.Render(shaper, style,
		[]richtext.SpanStyle{{Content: "deleted text", Strikethrough: true}}, richtext.Idle()))
	if plain == nil || struck == nil {
		return // headless unavailable; Capture called t.Skip
	}
	if n := golden.PixelDiff(plain, struck); n == 0 {
		t.Error("strikethrough and plain spans render identically; expected line-through pixels to differ")
	}
}

// ---- Layout tests ----

func measure(shaper *text.Shaper, style richtext.Style, spans []richtext.SpanStyle, maxWidth int) layout.Dimensions {
	var ops op.Ops
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(maxWidth, 10_000)},
		Ops:         &ops,
	}
	return richtext.Render(shaper, style, spans, richtext.Idle())(gtx)
}

// TestParagraphWraps verifies that narrowing the constraint wraps the spans
// into more lines: the narrow layout must be taller than the wide one, and
// both must respect their max width.
func TestParagraphWraps(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	wide := measure(shaper, style, mixedSpans(), 600)
	narrow := measure(shaper, style, mixedSpans(), 150)

	if wide.Size.X > 600 || narrow.Size.X > 150 {
		t.Errorf("layout exceeds max width: wide %v (max 600), narrow %v (max 150)", wide.Size, narrow.Size)
	}
	if narrow.Size.Y <= wide.Size.Y {
		t.Errorf("narrow layout height %d not greater than wide %d; spans did not wrap", narrow.Size.Y, wide.Size.Y)
	}
	if wide.Size.Y == 0 || wide.Size.X == 0 {
		t.Errorf("wide layout has empty size %v", wide.Size)
	}
}

// TestHardNewlineBreaksLine verifies that a \n inside a span forces a line
// break: the two-line content must be taller than the same content on one
// line.
func TestHardNewlineBreaksLine(t *testing.T) {
	shaper := defaultShaper(t)
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)

	oneLine := measure(shaper, style, []richtext.SpanStyle{{Content: "alpha beta"}}, 600)
	twoLines := measure(shaper, style, []richtext.SpanStyle{{Content: "alpha\nbeta"}}, 600)

	if twoLines.Size.Y <= oneLine.Size.Y {
		t.Errorf("newline content height %d not greater than single line %d", twoLines.Size.Y, oneLine.Size.Y)
	}
}

// ---- Interaction tests ----

func driveFrame(w layout.Widget, ops *op.Ops, r *gioinput.Router, size image.Point) layout.Dimensions {
	ops.Reset()
	gtx := layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: size},
		Ops:         ops,
		Source:      r.Source(),
	}
	dims := w(gtx)
	r.Frame(ops)
	return dims
}

// linkFirstSpans puts the link at the paragraph origin so its interactive
// area is at a known position for synthetic pointer events.
func linkFirstSpans(url string) []richtext.SpanStyle {
	return []richtext.SpanStyle{
		{Content: "click here", URL: url},
		{Content: " for docs."},
	}
}

// TestLinkClickFiresOnLinkClick drives a pointer press+release over the link
// segment and expects OnLinkClick to fire with the link's URL and a live
// layout.Context (GX.8).
func TestLinkClickFiresOnLinkClick(t *testing.T) {
	shaper := defaultShaper(t)
	const url = "https://example.com/docs"

	var gotURL string
	var gotOps bool
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	style.OnLinkClick = func(gtx layout.Context, u string) {
		gotURL = u
		gotOps = gtx.Ops != nil
	}

	state := richtext.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return richtext.Layout(gtx, state, shaper, style, linkFirstSpans(url))
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(400, 100)

	// Frame 1 registers the link area; the click then lands inside the
	// link's first segment (origin at 0,0; ~16 px tall at 16 sp).
	driveFrame(w, ops, r, size)
	hit := f32.Pt(6, 8)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: hit, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: hit, Source: pointer.Mouse},
	)
	driveFrame(w, ops, r, size)

	if gotURL != url {
		t.Fatalf("OnLinkClick url = %q, want %q", gotURL, url)
	}
	if !gotOps {
		t.Error("OnLinkClick received a layout.Context without Ops; callbacks must carry the live gtx (GX.8)")
	}
}

// TestLinkFocusTraversalAndKeyboardActivation moves focus forward across two
// links (the router-level traversal Gio's window drives from Tab), asserting
// the focus order matches document order, and activates each focused link
// with Enter and Space.
func TestLinkFocusTraversalAndKeyboardActivation(t *testing.T) {
	shaper := defaultShaper(t)

	var clicks []string
	style := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	style.OnLinkClick = func(_ layout.Context, u string) { clicks = append(clicks, u) }

	spans := []richtext.SpanStyle{
		{Content: "first", URL: "https://a.example"},
		{Content: " and "},
		{Content: "second", URL: "https://b.example"},
	}
	state := richtext.NewState()
	w := func(gtx layout.Context) layout.Dimensions {
		return richtext.Layout(gtx, state, shaper, style, spans)
	}

	r := new(gioinput.Router)
	ops := new(op.Ops)
	size := image.Pt(400, 100)

	probe := func() layout.Context {
		return layout.Context{
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: size},
			Ops:         new(op.Ops),
			Source:      r.Source(),
		}
	}

	// Frame 1 registers both links as focusable.
	driveFrame(w, ops, r, size)
	if got := state.FocusedLink(probe()); got != richtext.NoLink {
		t.Fatalf("initial FocusedLink = %d, want NoLink", got)
	}

	// Forward traversal reaches link 0, then link 1, in document order.
	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if got := state.FocusedLink(probe()); got != 0 {
		t.Fatalf("after first MoveFocus, FocusedLink = %d, want 0", got)
	}

	// Enter activates the focused link.
	r.Queue(
		key.Event{Name: key.NameReturn, State: key.Press},
		key.Event{Name: key.NameReturn, State: key.Release},
	)
	driveFrame(w, ops, r, size)
	if len(clicks) != 1 || clicks[0] != "https://a.example" {
		t.Fatalf("after Enter on link 0, clicks = %v, want [https://a.example]", clicks)
	}

	r.MoveFocus(key.FocusForward)
	driveFrame(w, ops, r, size)
	if got := state.FocusedLink(probe()); got != 1 {
		t.Fatalf("after second MoveFocus, FocusedLink = %d, want 1", got)
	}

	// Space also activates.
	r.Queue(
		key.Event{Name: key.NameSpace, State: key.Press},
		key.Event{Name: key.NameSpace, State: key.Release},
	)
	driveFrame(w, ops, r, size)
	if len(clicks) != 2 || clicks[1] != "https://b.example" {
		t.Fatalf("after Space on link 1, clicks = %v, want [... https://b.example]", clicks)
	}
}

// ---- Token defaults ----

// TestFromTokensDefaults pins the FromTokens contract: body text in Text at
// BodyLarge, links in Primary, focus ring in FocusRing.
func TestFromTokensDefaults(t *testing.T) {
	st := richtext.FromTokens(tokens.DefaultLight, tokens.DefaultTypography.BodyLarge)
	if st.Color != tokens.DefaultLight.Text {
		t.Errorf("Color = %v, want Text %v", st.Color, tokens.DefaultLight.Text)
	}
	if st.LinkColor != tokens.DefaultLight.Primary {
		t.Errorf("LinkColor = %v, want Primary %v", st.LinkColor, tokens.DefaultLight.Primary)
	}
	if st.FocusColor != tokens.DefaultLight.FocusRing() {
		t.Errorf("FocusColor = %v, want FocusRing %v", st.FocusColor, tokens.DefaultLight.FocusRing())
	}
	if st.Size != unit.Sp(tokens.DefaultTypography.BodyLarge.Size) {
		t.Errorf("Size = %v, want BodyLarge %v", st.Size, tokens.DefaultTypography.BodyLarge.Size)
	}
}
