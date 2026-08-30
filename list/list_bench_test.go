package list_test

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/layout"

	"github.com/vibrantgio/components/bench"
	"github.com/vibrantgio/components/list"
	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/theme/tokens"
)

// BenchmarkListLayout demonstrates O(visible) layout cost: ns/op stays roughly
// constant as total item count grows from 10 to 10000, because only the
// viewport-visible rows (~5 at viewH=150px, rowPx=30px) are ever laid out.
// Each sub-case plugs into the shared bench.BenchFrame harness.
func BenchmarkListLayout(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 10000} {
		items := makeItems(n)
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			state := list.NewState()
			w := func(gtx layout.Context) layout.Dimensions {
				return list.Layout(gtx, state, items, colorRowFn)
			}
			bench.BenchFrame(b, w, bench.WithSize(image.Pt(viewW, viewH)))
		})
	}
}

// BenchmarkLayoutScrollbar measures one LayoutScrollbar frame over 10k
// fixed-height rows, for comparison against the plain-Layout N=10000
// sub-case of BenchmarkListLayout above (same items, rowFn and viewport).
//
// Measured 2026-07-16 (go test ./list/ -bench ... -benchtime 2s -count 3
// -run '^$', darwin/arm64, Apple M1; ns/op stable across all 3 counts):
//
//	BenchmarkListLayout/N=10000-8        665 ns/op   0 B/op   0 allocs/op
//	BenchmarkLayoutScrollbar/Occupy-8   1379 ns/op   0 B/op   0 allocs/op
//	BenchmarkLayoutScrollbar/Overlay-8  1390 ns/op   0 B/op   0 allocs/op
//
// Finding: the delta over plain Layout is ~+107% (665 → ~1385 ns/op). A CPU
// profile attributes the entire gap to scrollbar.Style.Layout drawing the bar
// itself (~31% of samples, dominated by clip.Path.CubeTo / op.Offset building
// the rounded-corner thumb path in Gio) — a constant ~0.7 µs per frame that is
// independent of item count and adds zero allocations. The per-row list path
// is untouched (colorRowFn + List.layout costs match the baseline), so this is
// the inherent O(1) price of rendering the bar, not a regression in the
// virtual-list machinery.
func BenchmarkLayoutScrollbar(b *testing.B) {
	items := makeItems(10000)
	for _, tc := range []struct {
		name   string
		anchor list.Anchor
	}{
		{"Occupy", list.Occupy},
		{"Overlay", list.Overlay},
	} {
		b.Run(tc.name, func(b *testing.B) {
			state := list.NewState()
			bar := scrollbar.FromTokens(tokens.DefaultLight)
			w := func(gtx layout.Context) layout.Dimensions {
				return list.LayoutScrollbar(gtx, state, bar, tc.anchor, items, colorRowFn)
			}
			bench.BenchFrame(b, w, bench.WithSize(image.Pt(viewW, viewH)))
		})
	}
}
