package palette_test

import (
	"strings"
	"testing"

	"github.com/vibrantgio/components/gallery/palette"
)

// TestBoardHeadsItsNamesName pins the head over the name column and the hint
// beside the section's title: every role of the set has a name, whether the
// platform answers for it or the library does, so neither text says the
// platform answers for the names.
func TestBoardHeadsItsNamesName(t *testing.T) {
	if palette.NameHead != "Name" {
		t.Errorf("NameHead = %q, want %q", palette.NameHead, "Name")
	}
	if strings.Contains(palette.SetHint, "answers for") {
		t.Errorf("SetHint = %q, it says the platform answers for the names", palette.SetHint)
	}
	if !strings.Contains(palette.SetHint, "in the order the set carries them") {
		t.Errorf("SetHint = %q, want the clause on the order", palette.SetHint)
	}
}
