package icons

import (
	"image/color"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// TestEveryMarkParses is the check that keeps the parse in Register and New
// from ever failing on a released tree: the source ships inside the binary, so
// a malformed or misnamed file is a broken build and this is where it breaks.
func TestEveryMarkParses(t *testing.T) {
	marks, err := loadMarks()
	if err != nil {
		t.Fatal(err)
	}
	if len(marks) == 0 {
		t.Fatal("the set is empty")
	}
	for key, parsed := range marks {
		if parsed == nil {
			t.Errorf("%q parsed to nothing", key)
			continue
		}
		if len(parsed.Paths) == 0 {
			t.Errorf("%q carries no paths", key)
		}
		if parsed.ViewBox.W != 24 || parsed.ViewBox.H != 24 {
			t.Errorf("%q is drawn on a %g×%g grid, and the set's grid is 24×24",
				key, parsed.ViewBox.W, parsed.ViewBox.H)
		}
	}
}

// TestEveryPlatformDrawingHasAFallback is the promise resolution rests on: a
// name that answers on one operating system answers on all of them.
func TestEveryPlatformDrawingHasAFallback(t *testing.T) {
	marks, err := loadMarks()
	if err != nil {
		t.Fatal(err)
	}
	for key := range marks {
		name, _, isVariant := strings.Cut(key, platformSep)
		if !isVariant {
			continue
		}
		if _, ok := marks[name]; !ok {
			t.Errorf("%q is drawn for one platform and nothing answers on the others", key)
		}
	}
}

// TestRegistryKeyFromFileName pins the one registration mechanism the set has:
// what a mark is called on disk is what it is called in the registry.
func TestRegistryKeyFromFileName(t *testing.T) {
	for _, tc := range []struct {
		file    string
		want    string
		wantErr bool
	}{
		{file: "sidebar.svg", want: "sidebar"},
		{file: "sidebar.darwin.svg", want: "sidebar@darwin"},
		{file: "history-back.svg", want: "history-back"},
		{file: "history-back.windows.svg", want: "history-back@windows"},
		{file: "Sidebar.svg", wantErr: true},
		{file: "side_bar.svg", wantErr: true},
		{file: "sidebar-.svg", wantErr: true},
		{file: "sidebar.darwin.extra.svg", wantErr: true},
		{file: "sidebar..svg", wantErr: true},
		{file: ".svg", wantErr: true},
	} {
		got, err := registryKey(tc.file)
		switch {
		case tc.wantErr && err == nil:
			t.Errorf("%s: accepted, and it should not have been (got %q)", tc.file, got)
		case !tc.wantErr && err != nil:
			t.Errorf("%s: %v", tc.file, err)
		case !tc.wantErr && got != tc.want:
			t.Errorf("%s: key %q, want %q", tc.file, got, tc.want)
		}
	}
}

// TestBuiltOpsAreKeyedBySizeAlone is the colour decision, asserted: painting
// one mark in twenty colours builds it once, and only a new size builds it
// again. A control animating its foreground therefore rebuilds nothing.
func TestBuiltOpsAreKeyedBySizeAlone(t *testing.T) {
	s := New("darwin")
	mark := s.Mark(Sidebar)
	if mark == nil {
		t.Fatal("no painter for the sidebar")
	}

	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	for i := range 20 {
		mark(gtx, 20, color.NRGBA{R: uint8(i * 12), A: 0xff})
	}
	if got := s.built(); got != 1 {
		t.Errorf("twenty colours built %d drawings, want 1", got)
	}

	for _, px := range []int{16, 24} {
		mark(gtx, px, black)
	}
	if got := s.built(); got != 3 {
		t.Errorf("three sizes built %d drawings, want 3", got)
	}

	// And a size already built is not built again.
	mark(gtx, 16, black)
	if got := s.built(); got != 3 {
		t.Errorf("a size already drawn built %d drawings, want 3", got)
	}
}

// TestNothingIsBuiltForAnEmptySquare: a control with no room for a glyph asks
// for none, and gets no ops and no cache entry rather than a zero-sized one.
func TestNothingIsBuiltForAnEmptySquare(t *testing.T) {
	s := New("darwin")
	mark := s.Mark(Sidebar)
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	mark(gtx, 0, black)
	mark(gtx, -8, black)
	if got := s.built(); got != 0 {
		t.Errorf("%d drawings built for a square with no room in it, want 0", got)
	}
}

var black = color.NRGBA{A: 0xff}

// built reports how many drawings the set has built so far.
func (s *Set) built() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.cache)
}
