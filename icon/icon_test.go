package icon_test

import (
	"testing"

	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/svg"
)

func TestRegistryResolveSVG(t *testing.T) {
	r := icon.New()
	svgIcon := &svg.Icon{ViewBox: svg.ViewBox{W: 24, H: 24}}
	r.Register("circle", icon.FromSVG(svgIcon))

	got, ok := r.Icon("circle")
	if !ok {
		t.Fatal("circle not found in registry")
	}
	if got.Kind() != icon.KindSVG {
		t.Errorf("kind = %v, want KindSVG", got.Kind())
	}
	if got.SVG() != svgIcon {
		t.Errorf("SVG() returned wrong pointer")
	}
}

func TestRegistryResolveIVG(t *testing.T) {
	r := icon.New()
	// minimal valid IVG header: magic bytes + end-of-metadata byte
	ivgData := []byte{0x89, 0x49, 0x56, 0x47, 0x00}
	r.Register("info", icon.FromIVG(ivgData))

	got, ok := r.Icon("info")
	if !ok {
		t.Fatal("info not found in registry")
	}
	if got.Kind() != icon.KindIVG {
		t.Errorf("kind = %v, want KindIVG", got.Kind())
	}
	if len(got.IVG()) != len(ivgData) {
		t.Errorf("IVG() length = %d, want %d", len(got.IVG()), len(ivgData))
	}
}

func TestRegistryMissing(t *testing.T) {
	r := icon.New()
	_, ok := r.Icon("nonexistent")
	if ok {
		t.Error("expected false for missing icon, got true")
	}
}
