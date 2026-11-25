// Package dragdrop moves an item to a new place in a window with the
// pointer, and reorders an item in its own sequence.
//
// The user presses an item, moves the pointer and releases it at the place
// the item goes to. A press that does not move past [Threshold] stays a
// click. While the pointer moves, an image of the item follows it, and a
// line marks the place where the item would land, between the items that
// are there. Released over a place, the item lands there and the
// application gets one [Move]. Released anywhere else, or after Escape,
// the application gets nothing. While the pointer is near an edge of a
// scroll area, the area scrolls toward that edge: [State.Edges] and
// [State.Scroll] do it at the top and the bottom of a sequence, and
// [State.SideEdges] and [State.SideScroll] at the leading and the trailing
// edge of an area that scrolls sideways.
//
// The package changes no data. The application keeps its own sequences,
// lays out each item through [State.Item] and the space after the last
// item of a sequence through [State.End], and applies each [Move] that
// [State.Update] returns.
//
// # Frames
//
// Gio gives no way to read the transform of a frame back, and the items of
// a window are in different scrolled and clipped frames. So the
// application gives no rectangles. Each item registers its places in its
// own frame, and Gio's hit test finds the place under the pointer: the
// drag area is the source of a Gio transfer, and each place is a target of
// it, so each place gets the pointer's Enter and Leave while the button is
// down. The line is drawn in the frame of the place that holds the
// pointer, and the image in the frame of the drag area.
package dragdrop

import (
	"image"
	"image/color"
	"io"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/vibrantgio/theme/tokens"
)

// mime is the transfer type that joins the drag area to its places. No
// data moves through it: the drag area offers an empty body, because the
// application keeps the item.
const mime = "application/x-vibrantgio-dragdrop"

// Threshold is the distance the pointer moves from the press before a
// drag starts. DERIVED: the 3 dp slop that Gio's gesture package waits
// for before its own drag starts. The stored reference holds no platform
// drag threshold.
const Threshold = unit.Dp(3)

// LineWidth is the thickness of the line that marks the place. DERIVED
// until a capture of a platform drag is stored.
const LineWidth = unit.Dp(2)

// Opacity is the opacity of the image that follows the pointer. DERIVED:
// the platform draws the dragged image translucent, and the stored
// reference does not measure its opacity.
const Opacity = 0.75

// EdgeDp is the depth of the band along each edge of a scroll area where
// the pointer scrolls it: the top and the bottom of a scrolling sequence,
// and the leading and the trailing edge of an area that scrolls sideways.
// DERIVED: the regular control height.
const EdgeDp = unit.Dp(24)

// ScrollSpeed is how far a scroll area scrolls each second while the
// pointer is in one of its edge bands, in both directions. DERIVED until a
// platform drag is measured.
const ScrollSpeed = unit.Dp(480)

// Place is a position in a sequence of items. Sequence is the application's
// number for the sequence. For an item, Index is the item's index in
// its sequence. For a place where an item lands, Index is the gap
// before the item at that index: 0 is before the first item, and the
// length of the sequence is after the last item.
type Place struct {
	Sequence int
	Index    int
}

// Move is the one message the application gets when an item lands. From is
// the item's place. To is the gap it lands in, counted in the destination
// sequence before the item leaves From.
type Move struct {
	From, To Place
}

// Index returns the index the item has in the destination sequence after
// the move: the application removes the item at From and inserts it at
// this index.
func (m Move) Index() int {
	if m.From.Sequence == m.To.Sequence && m.To.Index > m.From.Index {
		return m.To.Index - 1
	}
	return m.To.Index
}

// Style is what the image and the line are drawn in.
type Style struct {
	// Line is the colour of the line that marks the place.
	Line color.NRGBA
}

// FromTokens returns the style for the platform's colours. The line is in
// the accent fill, the fill of a checked checkbox. DERIVED until a capture
// of a platform drag is stored.
func FromTokens(p tokens.PlatformColors) Style {
	return Style{Line: p.AccentFill}
}

// Image draws the item at from as the image that follows the pointer.
// It gets a context whose constraints are the item's size at the press.
// It draws the item as it is and registers no event handlers of its own;
// a static render is right. Event handlers it registers do not block the
// places under the pointer.
type Image func(gtx layout.Context, from Place) layout.Dimensions

// State is one drag area across frames. The application holds one per drag
// area and uses it only on the frame goroutine.
type State struct {
	// area is non-zero-size so that its address is a unique event tag.
	area struct{ _ byte }

	items map[Place]*itemTags
	edges map[int]*edgeTags
	sides map[int]*edgeTags

	// pressed is true while the button that pressed an item is down.
	pressed bool
	// from is the pressed item, grab the press in the item's frame and
	// size the item's size.
	from Place
	grab f32.Point
	size image.Point
	// pressedItem records an item press seen this frame, before the
	// drag area sees the same press.
	pressedItem bool

	start, at f32.Point // the press and the pointer, in the drag area's frame
	pid       pointer.ID
	dragging  bool
	cancelled bool

	// target is the tag of the place that holds the pointer, and place
	// its gap. They follow the pointer's Enter and Leave also when no
	// drag is in progress, so a drag that starts inside a place knows
	// it.
	target event.Tag
	place  Place

	// style is the drag area's style for the frame being laid out.
	style Style

	// vert is the band at the top or the bottom of a sequence that holds
	// the pointer, and side the band at the leading or the trailing edge
	// of an area that scrolls sideways. The two are apart because a
	// corner of a sequence can be in a band of each.
	vert, side band

	moves []Move
}

type tag struct{ _ byte }

type itemTags struct {
	press         tag
	before, after tag
	size          image.Point
}

// edgeTags are the two bands of one scroll area: the top and the bottom
// band of a sequence, or the leading and the trailing band of an area that
// scrolls sideways.
type edgeTags struct {
	start, end tag
}

// band is the edge band that holds the pointer on one axis. key is the
// application's number for the scroll area, and dir -1 for the band at its
// start and +1 for the band at its end. dir is 0 for none.
type band struct {
	tag  event.Tag
	key  int
	dir  int
	last time.Time
}

// Dragging reports whether a drag is in progress and Escape has not
// cancelled it.
func (s *State) Dragging() bool { return s.dragging && !s.cancelled }

// From returns the place of the item that is dragged. It is valid while
// [State.Dragging] is true.
func (s *State) From() Place { return s.from }

// Target returns the place where the dragged item would land, and false
// when the pointer is over no place.
func (s *State) Target() (Place, bool) {
	if !s.Dragging() || s.target == nil {
		return Place{}, false
	}
	return s.place, true
}

// Update returns the moves of the items that landed since the last
// call, in order. Most frames return none, and a frame returns at most
// one per release.
func (s *State) Update() []Move {
	m := s.moves
	s.moves = nil
	return m
}

func (s *State) itemTags(at Place) *itemTags {
	if s.items == nil {
		s.items = make(map[Place]*itemTags)
	}
	t := s.items[at]
	if t == nil {
		t = new(itemTags)
		s.items[at] = t
	}
	return t
}

// Layout lays out the drag area: content, the items it lays out
// through Item and End, and the image of a dragged item over them. The
// drag area takes the context's maximum constraints; a press outside it
// starts no drag.
func (s *State) Layout(gtx layout.Context, st Style, img Image, content layout.Widget) layout.Dimensions {
	s.style = st
	// Pass mode covers only the drag area's own event handler: a handler
	// that the content registers keeps blocking the handlers under it.
	pass := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, &s.area)
	pass.Pop()
	dims := content(gtx)
	cl.Pop()

	s.update(gtx)

	if s.Dragging() && img != nil {
		s.drawImage(gtx, img)
	}
	return dims
}

// update reads the drag area's events. It runs after the content, so the
// item press and the place changes of this frame are known before the
// drag area's press and release of the same frame.
func (s *State) update(gtx layout.Context) {
	for {
		e, ok := gtx.Event(
			pointer.Filter{Target: &s.area, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel},
			transfer.SourceFilter{Target: &s.area, Type: mime},
		)
		if !ok {
			break
		}
		switch e := e.(type) {
		case pointer.Event:
			s.pointer(gtx, e)
		case transfer.RequestEvent:
			gtx.Execute(transfer.OfferCmd{Tag: &s.area, Type: e.Type, Data: io.NopCloser(strings.NewReader(""))})
		}
	}
	if s.dragging {
		for {
			e, ok := gtx.Event(key.Filter{Name: key.NameEscape})
			if !ok {
				break
			}
			if e, ok := e.(key.Event); ok && e.State == key.Press {
				s.cancelled = true
			}
		}
	}
	s.pressedItem = false
}

func (s *State) pointer(gtx layout.Context, e pointer.Event) {
	switch e.Kind {
	case pointer.Press:
		s.reset()
		if !s.pressedItem || e.Buttons != pointer.ButtonPrimary {
			return
		}
		s.pressed = true
		s.start, s.at, s.pid = e.Position, e.Position, e.PointerID
	case pointer.Drag:
		if !s.pressed || e.PointerID != s.pid {
			return
		}
		s.at = e.Position
		if !s.dragging {
			d := s.at.Sub(s.start)
			slop := float32(gtx.Dp(Threshold))
			if d.X*d.X+d.Y*d.Y <= slop*slop {
				return
			}
			s.dragging = true
			// The grab cancels the press for the item's own controls,
			// so a drag that ends over the item is no click.
			gtx.Execute(pointer.GrabCmd{Tag: &s.area, ID: e.PointerID})
		}
		gtx.Execute(op.InvalidateCmd{})
	case pointer.Release:
		if s.Dragging() && s.target != nil && e.PointerID == s.pid {
			m := Move{From: s.from, To: s.place}
			if !noop(m) {
				s.moves = append(s.moves, m)
			}
		}
		s.reset()
	case pointer.Cancel:
		s.reset()
	}
}

// noop reports whether m lands the item in one of the two gaps next to
// it in its own sequence, where it already is.
func noop(m Move) bool {
	return m.From.Sequence == m.To.Sequence &&
		(m.To.Index == m.From.Index || m.To.Index == m.From.Index+1)
}

func (s *State) reset() {
	s.pressed, s.dragging, s.cancelled = false, false, false
}

func (s *State) drawImage(gtx layout.Context, img Image) {
	m := op.Record(gtx.Ops)
	off := s.at.Sub(s.grab)
	op.Offset(image.Pt(int(off.X+0.5), int(off.Y+0.5))).Add(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	o := paint.PushOpacity(gtx.Ops, Opacity)
	g := gtx
	g.Constraints = layout.Exact(s.size)
	img(g, s.from)
	o.Pop()
	pass.Pop()
	op.Defer(gtx.Ops, m.Stop())
}

// Item lays out w as the item at the place at and registers the item's two
// places: the gap before it over its leading half and the gap after it
// over its trailing half. The sequence is vertical. gap is the space in
// pixels the application leaves between items: each half reaches half of
// it beyond the item, and the line is drawn in its middle. A press on the
// item starts a drag once it passes [Threshold]; the item's own controls
// still get the press, and a press that does not move is their click.
func (s *State) Item(gtx layout.Context, at Place, gap int, w layout.Widget) layout.Dimensions {
	t := s.itemTags(at)
	for {
		e, ok := gtx.Event(pointer.Filter{Target: &t.press, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := e.(pointer.Event); ok && e.Kind == pointer.Press && e.Buttons == pointer.ButtonPrimary {
			s.pressedItem = true
			s.from, s.grab, s.size = at, e.Position, t.size
		}
	}
	s.readPlace(gtx, &t.before, Place{at.Sequence, at.Index})
	s.readPlace(gtx, &t.after, Place{at.Sequence, at.Index + 1})

	dims := w(gtx)
	t.size = dims.Size
	if s.pressedItem && s.from == at {
		s.size = dims.Size
	}

	half := dims.Size.Y / 2
	reach := gap / 2
	pass := pointer.PassOp{}.Push(gtx.Ops)
	a := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &t.press)
	a.Pop()
	s.addHandler(gtx, &t.before, image.Rect(0, -reach, dims.Size.X, half))
	s.addHandler(gtx, &t.after, image.Rect(0, half, dims.Size.X, dims.Size.Y+gap-reach))
	pass.Pop()

	if s.target == &t.before {
		s.drawLine(gtx, -reach, dims.Size.X)
	} else if s.target == &t.after {
		s.drawLine(gtx, dims.Size.Y+gap-reach, dims.Size.X)
	}
	return dims
}

// End registers the place after the last item of the sequence
// seq, which holds n items, over the context's maximum
// constraints. The application lays it out after the last item, or over the
// whole of an empty sequence. gap is the gap Item takes: the line is
// drawn half of it below the top, where the last item's line is.
func (s *State) End(gtx layout.Context, seq, n, gap int) layout.Dimensions {
	t := s.itemTags(Place{seq, -1})
	at := Place{seq, n}
	s.readPlace(gtx, &t.after, at)
	size := gtx.Constraints.Max
	pass := pointer.PassOp{}.Push(gtx.Ops)
	s.addHandler(gtx, &t.after, image.Rectangle{Max: size})
	pass.Pop()
	if s.target == &t.after {
		s.drawLine(gtx, gap-gap/2, size.X)
	}
	return layout.Dimensions{Size: size}
}

// readPlace reads the events of one place's tag. The place holds the
// pointer from its Enter to its Leave.
func (s *State) readPlace(gtx layout.Context, tg event.Tag, at Place) {
	for {
		e, ok := gtx.Event(
			pointer.Filter{Target: tg, Kinds: pointer.Enter | pointer.Leave},
			transfer.TargetFilter{Target: tg, Type: mime},
		)
		if !ok {
			break
		}
		switch e := e.(type) {
		case pointer.Event:
			switch e.Kind {
			case pointer.Enter:
				s.target, s.place = tg, at
			case pointer.Leave:
				if s.target == tg {
					s.target = nil
				}
			}
		case transfer.DataEvent:
			e.Open().Close()
		}
	}
	if s.target == tg {
		// A place can change its gap between frames when the application moves
		// items; the place that holds the pointer keeps its tag's gap.
		s.place = at
	}
}

func (s *State) addHandler(gtx layout.Context, tg event.Tag, r image.Rectangle) {
	a := clip.Rect(r).Push(gtx.Ops)
	event.Op(gtx.Ops, tg)
	a.Pop()
}

// drawLine draws the line across width at y, centred on y. It is
// deferred so that it is drawn over the items after it.
func (s *State) drawLine(gtx layout.Context, y, width int) {
	if !s.Dragging() {
		return
	}
	lw := max(gtx.Dp(LineWidth), 1)
	m := op.Record(gtx.Ops)
	top := y - lw/2
	paint.FillShape(gtx.Ops, s.style.Line, clip.Rect{Min: image.Pt(0, top), Max: image.Pt(width, top+lw)}.Op())
	op.Defer(gtx.Ops, m.Stop())
}

// Edges registers the edge bands of the scrolling sequence seq over a
// viewport of size, in the viewport's frame: a band at its top and a band
// at its bottom. The application lays it out after the sequence's items,
// so that the bands are over them.
func (s *State) Edges(gtx layout.Context, seq int, size image.Point) {
	t := edgeTagsFor(&s.edges, seq)
	s.edgeEvents(gtx, &s.vert, &t.start, seq, -1)
	s.edgeEvents(gtx, &s.vert, &t.end, seq, 1)
	d := min(gtx.Dp(EdgeDp), size.Y/2)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	s.addHandler(gtx, &t.start, image.Rect(0, 0, size.X, d))
	s.addHandler(gtx, &t.end, image.Rect(0, size.Y-d, size.X, size.Y))
	pass.Pop()
}

// SideEdges registers the edge bands of a scroll area that scrolls
// sideways over a viewport of size, in the viewport's frame: a band at its
// leading edge and a band at its trailing edge. area is the application's
// number for the scroll area, apart from the numbers of the sequences. The
// application lays it out after the area's content, so that the bands are
// over it. The places in the area stay vertical sequences.
func (s *State) SideEdges(gtx layout.Context, area int, size image.Point) {
	t := edgeTagsFor(&s.sides, area)
	s.edgeEvents(gtx, &s.side, &t.start, area, -1)
	s.edgeEvents(gtx, &s.side, &t.end, area, 1)
	d := min(gtx.Dp(EdgeDp), size.X/2)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	s.addHandler(gtx, &t.start, image.Rect(0, 0, d, size.Y))
	s.addHandler(gtx, &t.end, image.Rect(size.X-d, 0, size.X, size.Y))
	pass.Pop()
}

func edgeTagsFor(m *map[int]*edgeTags, key int) *edgeTags {
	if *m == nil {
		*m = make(map[int]*edgeTags)
	}
	t := (*m)[key]
	if t == nil {
		t = new(edgeTags)
		(*m)[key] = t
	}
	return t
}

func (s *State) edgeEvents(gtx layout.Context, b *band, tg event.Tag, key, dir int) {
	for {
		e, ok := gtx.Event(
			pointer.Filter{Target: tg, Kinds: pointer.Enter | pointer.Leave},
			transfer.TargetFilter{Target: tg, Type: mime},
		)
		if !ok {
			break
		}
		switch e := e.(type) {
		case pointer.Event:
			switch e.Kind {
			case pointer.Enter:
				b.tag, b.key, b.dir = tg, key, dir
			case pointer.Leave:
				if b.tag == tg {
					b.tag, b.dir = nil, 0
				}
			}
		case transfer.DataEvent:
			e.Open().Close()
		}
	}
}

// Scroll returns the pixels the sequence seq scrolls by in this frame:
// negative towards its top, positive towards its bottom, and 0 while the
// pointer is in neither of its edge bands. The application adds it to the
// sequence's scroll position before it lays the sequence out.
func (s *State) Scroll(gtx layout.Context, seq int) int {
	return s.scroll(gtx, &s.vert, seq)
}

// SideScroll returns the pixels the area that scrolls sideways scrolls by
// in this frame: negative towards its leading edge, positive towards its
// trailing edge, and 0 while the pointer is in neither of its edge bands.
// area is the number that [State.SideEdges] takes. The application adds
// it to the area's scroll offset before it lays the area out.
func (s *State) SideScroll(gtx layout.Context, area int) int {
	return s.scroll(gtx, &s.side, area)
}

func (s *State) scroll(gtx layout.Context, b *band, key int) int {
	if !s.Dragging() || b.dir == 0 {
		b.last = time.Time{}
		return 0
	}
	if b.key != key {
		return 0
	}
	dt := time.Second / 60
	if !b.last.IsZero() && gtx.Now.After(b.last) {
		dt = min(gtx.Now.Sub(b.last), time.Second/10)
	}
	b.last = gtx.Now
	gtx.Execute(op.InvalidateCmd{})
	px := int(float32(gtx.Dp(ScrollSpeed))*float32(dt)/float32(time.Second) + 0.5)
	return b.dir * max(px, 1)
}
