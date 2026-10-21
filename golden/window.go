package golden

import (
	"fmt"
	"image"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/reactivego/rx"

	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
)

// ThemeOption overrides one field of the emission [StaticTheme] freezes.
type ThemeOption func(*theme.Theme)

// WithDensity sets the density the frozen theme emits.
func WithDensity(d tokens.Density) ThemeOption {
	return func(th *theme.Theme) { th.Density = rx.Of(d) }
}

// WithTypography sets the typography the frozen theme emits.
func WithTypography(typo tokens.Typography) ThemeOption {
	return func(th *theme.Theme) { th.Typography = rx.Of(typo) }
}

// WithRadius sets the radius scale the frozen theme emits.
//
// A stored render pins it to the zero scale: anti-aliased rounded corners vary
// between GPU contexts by enough to fail a pixel-exact diff.
func WithRadius(r tokens.RadiusScale) ThemeOption {
	return func(th *theme.Theme) { th.Radius = rx.Of(r) }
}

// StaticTheme freezes one colour scheme into a theme whose every field emits
// once — the shape theme/window feeds an application's layers, minus the live
// platform poll. Every field but the colours is the package default; the
// options override one each.
func StaticTheme(c tokens.PlatformColors, opts ...ThemeOption) theme.Theme {
	th := theme.Default()
	th.Platform = rx.Of(c)
	for _, opt := range opts {
		opt(&th)
	}
	return th
}

// resolveAttempts and resolveWindow bound [Resolve]: how long one subscription
// is given to emit, and how many are tried before the layer is called dead.
const (
	resolveAttempts = 3
	resolveWindow   = 2 * time.Second
)

// Resolve returns the first layout.Widget a layer emits.
//
// A layer that folds live streams onto its output never completes, so waiting
// for completion would block forever; the first non-nil emission is taken off
// a goroutine subscription that is then unsubscribed, with a timeout for a
// layer that emits nothing at all.
//
// The completion case drains the value first. A cold chain delivers its value
// and its completion back to back on the subscription goroutine, so both
// channels are ready by the time this one selects, and Go picks a ready case
// at random: measured, a bare rx.Of mapped to a layout.Widget took the
// completion with the value already buffered in 95 of 200 runs. The retries
// cover only the other case, a subscription that delivers nothing inside the
// window.
func Resolve(obs rx.Observable[layout.Widget]) (layout.Widget, error) {
	for i := 0; ; i++ {
		gotChan := make(chan layout.Widget, 1)
		errChan := make(chan error, 1)
		sub := obs.Subscribe(rx.GoroutineContext(), func(v layout.Widget, err error, done bool) {
			if done {
				select {
				case errChan <- err:
				default:
				}
				return
			}
			if v != nil {
				select {
				case gotChan <- v:
				default:
				}
			}
		})
		select {
		case w := <-gotChan:
			sub.Unsubscribe()
			return w, nil
		case err := <-errChan:
			select {
			case w := <-gotChan:
				sub.Unsubscribe()
				return w, nil
			default:
			}
			sub.Unsubscribe()
			if err != nil {
				return nil, err
			}
			// Completed with no value at all: nothing a retry can recover,
			// since the chain is cold and has already run to completion.
			return nil, fmt.Errorf("golden: layer completed without emitting a layout.Widget")
		case <-time.After(resolveWindow):
			sub.Unsubscribe()
			if i+1 >= resolveAttempts {
				return nil, fmt.Errorf("golden: layer emitted nothing within %v", resolveWindow)
			}
		}
	}
}

// Stack draws layers back to front into one layout.Widget and reports the
// frontmost one's dimensions. It is the composition an mvu window performs on
// the layer stack it is rendering, with the frame's event loop left out.
func Stack(layers ...layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		var dims layout.Dimensions
		for _, w := range layers {
			dims = w(gtx)
		}
		return dims
	}
}

// Layers resolves a layer stack — the one an application builds for its
// window to render — into the single layout.Widget [Stack] composes.
func Layers(t *testing.T, layers ...rx.Observable[layout.Widget]) layout.Widget {
	t.Helper()
	resolved := make([]layout.Widget, len(layers))
	for i, layer := range layers {
		w, err := Resolve(layer)
		if err != nil {
			t.Fatalf("golden: layer %d: %v", i, err)
		}
		resolved[i] = w
	}
	return Stack(resolved...)
}

// Window lays an application's whole layer stack out at size and captures it
// headlessly: the window's frame as the application composes it, with no
// display and no event loop. Pass one scheme's theme to the stack builder and
// call it again with the other to get the pair.
func Window(t *testing.T, size image.Point, layers ...rx.Observable[layout.Widget]) *image.RGBA {
	t.Helper()
	return Capture(t, size, Layers(t, layers...))
}

// WindowGolden is [Window] stored: the captured frame diffed against
// testdata/golden/<name>.png, with the update, missing-file and mismatch
// behaviour of [Render].
func WindowGolden(t *testing.T, name string, size image.Point, layers ...rx.Observable[layout.Widget]) {
	t.Helper()
	Compare(t, name, Window(t, size, layers...))
}
