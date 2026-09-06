package toast_test

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/vibrantgio/components/golden"
	"github.com/vibrantgio/components/toast"
	"github.com/vibrantgio/theme/tokens"
)

const (
	frameW, frameH = 272, 68
)

var (
	frameSize = image.Pt(frameW, frameH)
	// Sharp corner radius. Anti-aliased rounded corners vary slightly
	// between GPU contexts, breaking determinism.
	sharpRadius = tokens.RadiusScale{}
)

// defaultShaper returns the shaper every golden here draws with: the default
// typography's faces pinned, system fonts off, so the stored images are the
// same on every machine. A golden test pins its faces with
// DeterministicShaper; application code takes the fallback Shaper.
func defaultShaper(t *testing.T) *text.Shaper {
	t.Helper()
	return tokens.DefaultTypography.DeterministicShaper()
}

// roleText is the message each status role carries. ASCII only: Latin text
// in Roboto rasterises identically on every machine, and no symbol reaches a
// stored image.
func roleText(r toast.Role) string {
	switch r {
	case toast.Success:
		return "Workspace saved"
	case toast.Warning:
		return "Connection is slow"
	case toast.Error:
		return "Upload failed"
	default:
		return "Syncing tokens"
	}
}

// scene stands one toast one edge margin in from a flat background, which is
// how a placement hands it its slot: the toast takes its own width and hugs
// its message, so the frame around it is what the stored image shows of the
// surface it floats over.
func scene(w layout.Widget, bg color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		frame := gtx.Constraints.Max
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: frame}.Op())
		inset := gtx.Dp(unit.Dp(tokens.Spacing.S4))
		defer op.Offset(image.Pt(inset, inset)).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Constraints{
			Min: image.Pt(0, gtx.Dp(unit.Dp(toast.MinHeightDp))),
			Max: frame.Sub(image.Pt(2*inset, 2*inset)),
		}
		w(gtx)
		return layout.Dimensions{Size: frame}
	}
}

// TestToastGolden records or diffs one stored scene per status role in each
// scheme. The role's leading edge is the load-bearing visual signal — the
// fill is the same inverse surface at every role — and the message carries
// the LabelMedium role. The scenes composite over a real pane background
// (SurfaceAt(LevelChrome)), so a fill that stops separating from real app
// backgrounds fails the diff instead of hiding behind an arbitrary grey.
func TestToastGolden(t *testing.T) {
	shaper := defaultShaper(t)
	schemes := []struct {
		name   string
		colors tokens.ColorTokens
	}{
		{"light", tokens.DefaultLight},
		{"dark", tokens.DefaultDark},
	}
	roles := []struct {
		name string
		role toast.Role
	}{
		{"info", toast.Info},
		{"success", toast.Success},
		{"warning", toast.Warning},
		{"error", toast.Error},
	}
	for _, sc := range schemes {
		for _, r := range roles {
			name := r.name + "-" + sc.name
			t.Run(name, func(t *testing.T) {
				w := toast.Render(shaper, toast.Props{Role: r.role, Text: roleText(r.role), Shaper: shaper},
					sc.colors, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelMedium)
				golden.Render(t, name, frameSize, scene(w, sc.colors.SurfaceAt(tokens.LevelChrome)))
			})
		}
	}
}

// TestAlphaFadesTheWholeToast pins what a placement's fade reaches: the
// fill, the leading edge and the message all take Props.Alpha, so a toast on
// its way out never leaves a part of itself behind at full strength. A
// non-positive Alpha is the opaque default, which is what makes the zero
// Props usable.
func TestAlphaFadesTheWholeToast(t *testing.T) {
	shaper := defaultShaper(t)
	bg := color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	render := func(alpha float64) *image.RGBA {
		w := toast.Render(shaper, toast.Props{Role: toast.Warning, Text: roleText(toast.Warning), Alpha: alpha, Shaper: shaper},
			tokens.DefaultLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelMedium)
		return golden.Capture(t, frameSize, scene(w, bg))
	}
	opaque, defaulted, half := render(1), render(0), render(0.5)
	if n := golden.PixelDiff(opaque, defaulted); n != 0 {
		t.Errorf("a zero Alpha differs from a full one in %d pixels; the zero Props must paint solid", n)
	}
	if n := golden.PixelDiff(opaque, half); n == 0 {
		t.Error("half opacity renders identically to full; Alpha reaches nothing")
	}
}

// TestToastHugsItsMessage pins the two metrics a placement measures against:
// a toast takes WidthDp when it is given the room, and a one-line message
// leaves it at the MinHeightDp legibility floor rather than at the height its
// padding alone would give.
func TestToastHugsItsMessage(t *testing.T) {
	shaper := defaultShaper(t)
	var dims layout.Dimensions
	golden.Capture(t, frameSize, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Constraints{Max: frameSize}
		dims = toast.Render(shaper, toast.Props{Role: toast.Info, Text: roleText(toast.Info), Shaper: shaper},
			tokens.DefaultLight, tokens.Spacing, sharpRadius, tokens.DefaultTypography.LabelMedium)(gtx)
		return dims
	})
	if dims.Size.X != toast.WidthDp {
		t.Errorf("a toast in a %d px frame is %d px wide; want the %d dp it asks for", frameW, dims.Size.X, toast.WidthDp)
	}
	if dims.Size.Y != toast.MinHeightDp {
		t.Errorf("a one-line toast is %d px tall; want the %d dp floor", dims.Size.Y, toast.MinHeightDp)
	}
}
