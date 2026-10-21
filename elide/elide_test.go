package elide_test

import (
	"image"
	"strings"
	"testing"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/vibrantgio/components/elide"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

const address = "https://example.test/a/considerably/longer/path/than/the/bar/can/hold/page.html"

func pad() (layout.Context, *op.Ops) {
	ops := new(op.Ops)
	return layout.Context{
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(1<<20, 1<<20)},
		Ops:         ops,
	}, ops
}

// TestTextThatFitsIsLeftAlone is the case every short line is in: nothing is
// taken out and no ellipsis is added.
func TestTextThatFitsIsLeftAlone(t *testing.T) {
	gtx, _ := pad()
	sh := tokens.DefaultTypography.DeterministicShaper()
	style := tokens.DefaultTypography.LabelMedium
	full := typeset.LineWidth(gtx, sh, address, style, font.Normal)
	if got := elide.Middle(gtx, sh, address, style, full); got != address {
		t.Errorf("at its own width the address came back as %q", got)
	}
	if got := elide.Middle(gtx, sh, address, style, full*2); got != address {
		t.Errorf("at twice its own width the address came back as %q", got)
	}
}

// TestTheMiddleGoesAndTheEndsStay narrows the room to half the address and
// reads what is left: one ellipsis between a start and an end that are both
// the address's own, inside the width it was given.
func TestTheMiddleGoesAndTheEndsStay(t *testing.T) {
	gtx, _ := pad()
	sh := tokens.DefaultTypography.DeterministicShaper()
	style := tokens.DefaultTypography.LabelMedium
	width := typeset.LineWidth(gtx, sh, address, style, font.Normal) / 2

	got := elide.Middle(gtx, sh, address, style, width)
	if got == address {
		t.Fatalf("the address fits in half its own width unchanged: %q", got)
	}
	if n := strings.Count(got, elide.Ellipsis); n != 1 {
		t.Fatalf("%q carries %d ellipses, want exactly one", got, n)
	}
	cut := strings.Index(got, elide.Ellipsis)
	head, tail := got[:cut], got[cut+len(elide.Ellipsis):]
	if head == "" || tail == "" {
		t.Fatalf("%q keeps nothing on one of its two sides", got)
	}
	if !strings.HasPrefix(address, head) {
		t.Errorf("%q does not open with the address's own start", got)
	}
	if !strings.HasSuffix(address, tail) {
		t.Errorf("%q does not close with the address's own end", got)
	}
	if w := typeset.LineWidth(gtx, sh, got, style, font.Normal); w > width {
		t.Errorf("%q is %d wide in room for %d", got, w, width)
	}
}

// TestLessRoomKeepsLess walks the room down and reads the lines back: each
// one is at most as long as the one before it, and every one of them fits.
func TestLessRoomKeepsLess(t *testing.T) {
	gtx, _ := pad()
	sh := tokens.DefaultTypography.DeterministicShaper()
	style := tokens.DefaultTypography.LabelMedium
	full := typeset.LineWidth(gtx, sh, address, style, font.Normal)

	prev := len([]rune(address))
	for width := full; width > 0; width -= full / 8 {
		got := elide.Middle(gtx, sh, address, style, width)
		if got == address {
			// Room for nothing at all: the whole line comes back and is
			// drawn clipped, which is the documented floor.
			continue
		}
		if n := len([]rune(got)); n > prev {
			t.Errorf("at %d of room the address kept %d characters, more than the %d it kept in more room", width, n, prev)
		} else {
			prev = n
		}
		if w := typeset.LineWidth(gtx, sh, got, style, font.Normal); w > width {
			t.Errorf("at %d of room %q is %d wide", width, got, w)
		}
	}
}

// TestNoRoomIsTheWholeLine asks for a line in no room at all: the caller gets
// the text it handed over, which is drawn clipped rather than replaced by an
// ellipsis saying nothing.
func TestNoRoomIsTheWholeLine(t *testing.T) {
	gtx, _ := pad()
	sh := tokens.DefaultTypography.DeterministicShaper()
	style := tokens.DefaultTypography.LabelMedium
	if got := elide.Middle(gtx, sh, address, style, 0); got != address {
		t.Errorf("with no room the address came back as %q", got)
	}
	if got := elide.Middle(gtx, sh, "", style, 50); got != "" {
		t.Errorf("the empty line came back as %q", got)
	}
}
