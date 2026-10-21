// Package tooltip provides the tooltip signal: it names a control or
// explains another signal on demand, a small annotation floating beside
// its trigger, appearing by itself after a short delay on hover or focus
// and leaving when they do. It holds text only, never a control;
// anything the user must operate is the job for a popover.
//
// Arbitration is frame state: a plain Arbiter written and read during
// layout on the frame goroutine, in which a tooltip is visible exactly
// while it holds top. See arbitration.go.
//
// The package follows the Composition contract: Tooltip is a callable Go
// function consuming a components theme observable, returning a stream of
// layout.Widget. The source is intentionally short and free of opaque
// configuration — copy it into your own app and modify as needed.
//
// Colour: the platform's own names. A tooltip on this platform is the
// window's own plane — windowBackgroundColor — inside a separatorColor
// hairline, with its text in labelColor, and it is the same in both
// appearances. It carries no colour role of its own.
//
// No stored capture in the organization's macOS reference holds a tooltip,
// so the fill, the edge and the text here are the Language's mapping of
// what a floating annotation is on this platform rather than three numbers
// read off a capture; the hairline is what a still tooltip is told from its
// surroundings by, since the cast shadow belongs to whatever places it.
//
// THE FLOATING SURFACE PAINTS LAST. The trigger is drawn where the caller
// put it; the annotation goes through op.Defer. patterns/popover's package
// doc states the idiom and what deferral keeps and drops — this package
// applies it unchanged. The annotation leaves with its trigger: it stands
// only while the trigger holds hover or focus, and a trigger clipped out of
// view is no longer hit-tested, so the hover ends there and takes the
// annotation with it.
//
// The trigger renders at the frame centre, or where Props.Anchor says it
// already stands for a caller that drew its own; the tooltip surface is
// placed adjacent per Placement, taking the opposite side when the side
// asked for has no room left in the frame and the opposite side holds it
// whole, and shifted inward when it would otherwise cross the frame's
// leading or trailing edge. A centred trigger has the same room on both
// sides, so it is an anchored one that the flip is for. Show/hide is
// instantaneous in this package; entrance/exit transitions are deferred to a
// later Effects-integration goal. Touch long-press is out of scope.
//
// AN ANCHORED TRIGGER IS THE CALLER'S. The component registers no hit area
// and no focus tag over an anchor: the thing standing there is already a
// control of the caller's own, hit-tested and focusable by whatever drew
// it,
// and a second area over it would be an invisible tab stop and a hover that
// competes with the real one. What an anchored tooltip stands on instead is
// the anchor itself — a non-empty rectangle is the trigger standing, the
// empty rectangle is it gone — so the caller that knows its trigger holds
// the pointer or the keyboard says so by saying where it is.
package tooltip

import (
	"image"
	"image/color"
	"sync/atomic"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/elide"
	"github.com/vibrantgio/components/internal/surface"
	vgcolor "github.com/vibrantgio/theme/color"
	"github.com/vibrantgio/theme/theme"
	"github.com/vibrantgio/theme/tokens"
	"github.com/vibrantgio/theme/typeset"
)

// DefaultDelay is the show-after-entry delay applied when Props.Delay is
// zero or negative. It resolves from the motion scale's slowest stop
// (DurXSlow, 500 ms); the live path reads the same stop from
// its Theme.Motion snapshot, so a themed motion scale retimes the delay.
var DefaultDelay = tokens.Motion.DurXSlow

// Placement is the side of the trigger the tooltip surface is asked for. A
// surface with no room left on that side of the frame takes the opposite
// side when that one holds it whole; see drawSurface.
type Placement int

const (
	Top Placement = iota
	Bottom
	Left
	Right
)

// Props configures a Tooltip. Trigger must be non-nil unless Anchor says
// where the trigger already stands; Text may be empty (the surface still
// renders, but at minimum padded size).
type Props struct {
	Text      string
	Trigger   layout.Widget
	Delay     time.Duration
	Placement Placement

	// Anchor is where the trigger already stands inside the frame, for a
	// caller that has drawn its own trigger and wants nothing from this
	// component but the annotation placed against it — a word in a
	// paragraph, a cell of a table, anything whose place is a result of a
	// layout this component did not run. It is read once a frame, because
	// such a place moves with the wrapping and the scroll.
	//
	// With an anchor the frame is no longer the trigger's own box: the
	// trigger is where the anchor says and the frame is whatever room the
	// caller gave, which is what lets the annotation take the side of the
	// trigger that has room (see drawSurface). Props.Trigger is then
	// optional, and a trigger that is given is drawn at the anchor rather
	// than centred.
	//
	// The anchor is also what says the trigger is there at all: the
	// component registers no area over it and hit-tests nothing, so an
	// anchor that comes back empty is the trigger gone and the annotation
	// leaves with it. A caller lays an anchored tooltip out while its own
	// trigger holds the pointer or the keyboard, and gives the empty
	// rectangle when it holds neither.
	//
	// A nil Anchor centres Props.Trigger in the frame, which is what a
	// tooltip wrapping its own control wants.
	Anchor func(gtx layout.Context) image.Rectangle

	// Shaper is an explicit per-instance override of the text shaper. Leave
	// it nil in normal use: the tooltip then shapes its text with the
	// theme's shaper (Typography.Shaper()), which is built once for the
	// process and shared by every component reading that typography — the
	// cache lives behind the Typography value, so it survives the copy this
	// component's map function makes of it. Set it only when this instance
	// must shape with a different shaper than the theme provides.
	//
	// A shaper is not safe to use from two goroutines; Gio lays every
	// layout.Widget out on the one goroutine that runs the event loop,
	// which is what makes sharing it correct. See theme/tokens.Typography.Shaper.
	Shaper *text.Shaper

	// Surface is the opaque fill the bubble's own hairline lands on: the
	// hairline is painted as a shape the bubble's fill is inset inside, and
	// the platform's separator carries a coverage rather than a colour. The
	// zero value — no colour — is the window's own plane, which is what a
	// floating surface stands over on this platform.
	Surface color.NRGBA

	// Arbiter is the set of tooltips this one arbitrates within: showing it
	// hides whichever tooltip of the same set was up. Give each window its
	// own (see Arbiter). A nil Arbiter gets this tooltip one of its own, so
	// it arbitrates with nobody: sharing is the explicit act.
	Arbiter *Arbiter
}

type resolvedTokens struct {
	platform tokens.PlatformColors
	spacing  tokens.SpacingScale
	radius   tokens.RadiusScale
	style    tokens.TextStyle // the LabelSmall role: typeface, weight, size, line height
	shaper   *text.Shaper     // the theme's shaper; nil in the Render path
	// delay is the show-after-entry delay, the motion scale's DurXSlow
	// stop. Props.Delay overrides it per instance.
	delay time.Duration
}

// Tooltip returns an rx.Observable[layout.Widget] that emits a new one
// whenever the theme changes. State (the arbitration set and hold, hover
// gesture, focus tag, dwell stamp) persists across emissions in the
// rx.Defer scope.
func Tooltip(th rx.Observable[theme.Theme], props Props) rx.Observable[layout.Widget] {
	// Flatten the nested theme observables into a concrete snapshot. The
	// typography emission supplies both the LabelSmall text style and the
	// theme's cached shaper; the motion emission supplies the show delay.
	resolved := rx.SwitchMap(th, func(t theme.Theme) rx.Observable[resolvedTokens] {
		return rx.Map(
			rx.CombineLatest5(t.Platform, t.Spacing, t.Radius, t.Typography, t.Motion),
			func(n rx.Tuple5[tokens.PlatformColors, tokens.SpacingScale, tokens.RadiusScale, tokens.Typography, tokens.MotionScale]) resolvedTokens {
				typ := n.Fourth
				return resolvedTokens{
					platform: n.First,
					spacing:  n.Second,
					radius:   n.Third,
					style:    typ.LabelSmall,
					shaper:   typ.Shaper(),
					delay:    n.Fifth.DurXSlow,
				}
			},
		)
	})
	return func(observe rx.Observer[layout.Widget], sched rx.Scheduler, sub rx.Subscriber) {
		// rx.Defer's scope with a teardown: the state is per subscription,
		// and a subscription that goes while its tooltip is up has to say
		// so, or the arbiter it is top of would hold a tooltip nothing can
		// hide. The flag is read on the next frame a tooltip of that set
		// lays out; see [Arbiter].
		st := newState(props)
		// A stream that ran out is not a tooltip that went: the last
		// layout.Widget it emitted is still the caller's and is still laid
		// out, so only an unsubscription arriving before the completion is
		// a drop.
		var complete atomic.Bool
		sub.OnUnsubscribe(func() {
			if !complete.Load() {
				st.dropped.Store(true)
			}
		})
		widgets := rx.Map(resolved, func(tok resolvedTokens) layout.Widget {
			// Props.Shaper is an explicit override; the theme's shaper is
			// the default. Same for Props.Delay over the theme's motion
			// stop.
			shaper := props.Shaper
			if shaper == nil {
				shaper = tok.shaper
			}
			delay := props.Delay
			if delay <= 0 {
				delay = tok.delay
			}
			return func(gtx layout.Context) layout.Dimensions {
				return drawTooltip(gtx, shaper, props, delay, tok, st, true)
			}
		})
		widgets(func(next layout.Widget, err error, done bool) {
			if done {
				complete.Store(true)
			}
			observe(next, err, done)
		}, sched, sub)
	}
}

// Render produces a layout.Widget for a tooltip with pre-resolved tokens
// and an explicit shown flag. Intended for golden-image testing and
// static demonstrations; production code should use Tooltip, which reads
// the shaper and the same text style off the theme. The returned layout.Widget
// performs no input handling or arbitration: pass shown=true to render
// the trigger plus the floating surface, shown=false to render only the
// trigger.
//
// style is the LabelSmall role's whole text style — typeface, weight,
// size and line height all reach the shaper, exactly as they do on the
// live path. Pass tokens.DefaultTypography.LabelSmall for the default
// desktop look. There is no density parameter: a tooltip surface wraps
// its text and sizes no control.
func Render(
	shaper *text.Shaper,
	props Props,
	shown bool,
	colors tokens.PlatformColors,
	sp tokens.SpacingScale,
	rad tokens.RadiusScale,
	style tokens.TextStyle,
) layout.Widget {
	tok := resolvedTokens{platform: colors, spacing: sp, radius: rad, style: style}
	return func(gtx layout.Context) layout.Dimensions {
		return drawStatic(gtx, shaper, props, tok, shown)
	}
}

// tooltipState holds this tooltip's arbitration set, the hover/focus
// trackers, and the dwell timer's start and latch. One instance is owned by
// each Tooltip subscription. Its address is also the tooltip's identity
// inside the Arbiter — and, because a tooltip is visible exactly while it
// holds top, that identity is the whole of its visible state: there is no
// shown flag here to fall out of step with the arbiter.
type tooltipState struct {
	arb *Arbiter

	// dropped says this tooltip's subscription is gone. It is set from
	// whichever goroutine unsubscribed — a caller that rebuilds its tooltip
	// for every value it says does so from its own stream — and read on the
	// frame goroutine, which is the one that then drops the arbitration this
	// tooltip was holding when it went. See [Arbiter].
	dropped atomic.Bool

	// entryAt is when hover/focus entry happened, zero while the trigger
	// has neither. It is both the dwell timer's start and the edge
	// detector for entry and exit.
	entryAt time.Time
	// claimed says this dwell has already had its turn at the arbiter. The
	// dwell test is a level — "entryAt + delay is in the past" stays true
	// for as long as the pointer sits still — so without a latch a tooltip
	// hidden by a later claimant would take top straight back on its next
	// layout, and the two would trade the register every frame. One dwell
	// buys one show; a fresh entry buys the next.
	claimed bool

	hov      gesture.Hover
	focusTag int
}

func newState(props Props) *tooltipState {
	arb := props.Arbiter
	if arb == nil {
		arb = NewArbiter()
	}
	return &tooltipState{arb: arb}
}

// drawTooltip runs the per-frame logic: process hover/focus events,
// update entry-time and arbitration state, paint the trigger, and (when
// shown) paint the floating surface.
func drawTooltip(
	gtx layout.Context,
	shaper *text.Shaper,
	props Props,
	delay time.Duration,
	tok resolvedTokens,
	st *tooltipState,
	live bool,
) layout.Dimensions {
	frame := gtx.Constraints.Max

	// 1. Record the trigger into a macro to measure its dims; place it on
	//    the frame, centred or where Props.Anchor says it already is. The
	//    trigger's rect is the basis for both the hit area registered for
	//    hover/focus and the surface positioning math below.
	triggerDims, triggerPos, triggerOps := placeTrigger(gtx, props, frame)
	triggerRect := image.Rectangle{Min: triggerPos, Max: triggerPos.Add(triggerDims.Size)}

	// 2. Drain hover and focus events. The drains must happen before the
	//    hit-area registration in step 4, so the gesture/focus trackers
	//    reflect this frame's state. An anchored trigger is hit-tested by
	//    nothing here: the caller owns it, and the anchor standing is the
	//    whole of what this component knows about it.
	var active bool
	switch {
	case !live:
	case props.Anchor != nil:
		active = !triggerRect.Empty()
	default:
		for {
			if _, ok := gtx.Event(key.FocusFilter{Target: &st.focusTag}); !ok {
				break
			}
		}
		hovered := st.hov.Update(gtx.Source)
		focused := gtx.Focused(&st.focusTag)
		active = hovered || focused
	}

	// 3. Dwell timer and arbitration. Both are frame state, written and
	//    read here on the goroutine that will draw the result: the timer
	//    runs on gtx.Now and the claim is a store into a plain value.
	//    Nothing polls "am I still top" — losing top is not an event this
	//    tooltip has to notice, because holding it is the only thing that
	//    makes it paint. See arbitration.go.
	if live {
		switch {
		case active && st.entryAt.IsZero():
			// Hover/focus entry: start the dwell and schedule a redraw at
			// entry+delay so we wake to show even when no other input
			// arrives.
			st.entryAt = gtx.Now
			st.claimed = false
			gtx.Execute(op.InvalidateCmd{At: st.entryAt.Add(delay)})
		case !active && !st.entryAt.IsZero():
			// Hover/focus exit: end the dwell and give up top. release is a
			// no-op for a tooltip that was already overtaken, so a
			// straggler on its way out cannot hide its successor.
			st.entryAt = time.Time{}
			st.claimed = false
			st.arb.release(st)
		}
		// The dwell has run out: take top, which hides whichever tooltip
		// held it. Guarded by claimed rather than by "am I visible" — the
		// dwell test is a level, and one dwell buys exactly one show. See
		// tooltipState.claimed.
		if active && !st.claimed && !gtx.Now.Before(st.entryAt.Add(delay)) {
			st.claimed = true
			st.arb.claim(st)
		}
	}

	// 4. Paint the trigger at the centred offset. When live and unanchored
	//    it registers the hover gesture and a focus tag clipped to the
	//    trigger rect so Enter/Leave and focus events fire for this hit
	//    area. An anchored trigger gets neither: it is the caller's own
	//    control, already hit-tested and already in the window's Tab order.
	{
		triggerOff := op.Offset(triggerPos).Push(gtx.Ops)
		if live && props.Anchor == nil {
			triggerClip := clip.Rect{Max: triggerDims.Size}.Push(gtx.Ops)
			st.hov.Add(gtx.Ops)
			event.Op(gtx.Ops, &st.focusTag)
			triggerClip.Pop()
		}
		triggerOps.Add(gtx.Ops)
		triggerOff.Pop()
	}

	// 5. Surface, only while this tooltip holds arbitration top. Visibility
	//    is read straight off the arbiter rather than mirrored in a flag
	//    beside it.
	if st.arb.isTop(st) {
		drawSurface(gtx, shaper, props, tok, triggerRect)
	}

	return layout.Dimensions{Size: frame}
}

// drawStatic is the input-free variant used by Render: skips event
// processing and arbitration, but mirrors the layout math so the static
// frame matches the live frame at shown=true.
func drawStatic(
	gtx layout.Context,
	shaper *text.Shaper,
	props Props,
	tok resolvedTokens,
	shown bool,
) layout.Dimensions {
	frame := gtx.Constraints.Max
	triggerDims, triggerPos, triggerOps := placeTrigger(gtx, props, frame)
	triggerRect := image.Rectangle{Min: triggerPos, Max: triggerPos.Add(triggerDims.Size)}

	triggerOff := op.Offset(triggerPos).Push(gtx.Ops)
	triggerOps.Add(gtx.Ops)
	triggerOff.Pop()

	if shown {
		drawSurface(gtx, shaper, props, tok, triggerRect)
	}
	return layout.Dimensions{Size: frame}
}

// placeTrigger records the trigger and says where it stands in the frame:
// centred, which is what a tooltip that wraps its own control does, or at
// Props.Anchor, which is what a caller that drew its own trigger says. An
// anchored trigger takes the anchor's size whether or not there is a
// Props.Trigger to draw at it, because the anchor is the box the pointer is
// tested against and the annotation is placed beside.
func placeTrigger(gtx layout.Context, props Props, frame image.Point) (layout.Dimensions, image.Point, op.CallOp) {
	var anchor image.Rectangle
	if props.Anchor != nil {
		anchor = props.Anchor(gtx)
	}
	triggerGtx := gtx
	triggerGtx.Constraints = layout.Constraints{Max: frame}
	if props.Anchor != nil {
		triggerGtx.Constraints = layout.Exact(anchor.Size())
	}
	macro := op.Record(gtx.Ops)
	var dims layout.Dimensions
	if props.Trigger != nil {
		dims = props.Trigger(triggerGtx)
	}
	call := macro.Stop()
	if props.Anchor != nil {
		dims.Size = anchor.Size()
		return dims, anchor.Min, call
	}
	return dims, image.Pt((frame.X-dims.Size.X)/2, (frame.Y-dims.Size.Y)/2), call
}

// drawSurface paints the rounded tooltip bubble with the text inside,
// positioned adjacent to triggerRect per props.Placement. The bubble is
// filled with the window's own plane inside a separator hairline.
//
// Everything it draws is recorded and handed to op.Defer, so the annotation
// paints above every sibling the window lays out after the trigger's slot
// (see the package doc).
func drawSurface(
	gtx layout.Context,
	shaper *text.Shaper,
	props Props,
	tok resolvedTokens,
	triggerRect image.Rectangle,
) {
	floating := op.Record(gtx.Ops)

	frame := gtx.Constraints.Max
	r := gtx.Dp(unit.Dp(tok.radius.Sm))
	padH := gtx.Dp(unit.Dp(tok.spacing.S2))
	padV := gtx.Dp(unit.Dp(tok.spacing.S1))
	gap := gtx.Dp(unit.Dp(tok.spacing.S1))

	// Pre-record the text so it can be replayed inside the surface at a known
	// offset after measuring it.
	//
	// THE TEXT FITS THE WINDOW. The surface is three quarters of the frame
	// at the widest, and never wider than the frame has room for; an address
	// longer than that is elided in its middle rather than cut at its end,
	// because the host at one end and the leaf at the other are what a
	// reader is looking for. See [github.com/vibrantgio/components/elide].
	room := min(frame.X*3/4, frame.X-2*padH-2*gap)
	textGtx := gtx
	textGtx.Constraints = layout.Constraints{Max: image.Pt(max(room, 0), frame.Y/4)}
	textGtx.Constraints.Min = image.Point{}
	fg := vgcolor.Flatten(tok.platform.Label, tok.platform.WindowBackground)
	mText := op.Record(gtx.Ops)
	textDims := typeset.Text(textGtx, shaper, elide.Middle(gtx, shaper, props.Text, tok.style, room), tok.style, fg, 1)
	textCall := mText.Stop()

	surfW := textDims.Size.X + 2*padH
	surfH := textDims.Size.Y + 2*padV
	minW := gtx.Dp(unit.Dp(24))
	minH := gtx.Dp(unit.Dp(16))
	if surfW < minW {
		surfW = minW
	}
	if surfH < minH {
		surfH = minH
	}

	midX := (triggerRect.Min.X + triggerRect.Max.X) / 2
	midY := (triggerRect.Min.Y + triggerRect.Max.Y) / 2
	above := triggerRect.Min.Y - gap - surfH
	below := triggerRect.Max.Y + gap
	before := triggerRect.Min.X - gap - surfW
	after := triggerRect.Max.X + gap
	var pos image.Point
	switch props.Placement {
	case Top:
		pos = image.Pt(midX-surfW/2, above)
	case Bottom:
		pos = image.Pt(midX-surfW/2, below)
	case Left:
		pos = image.Pt(before, midY-surfH/2)
	case Right:
		pos = image.Pt(after, midY-surfH/2)
	}

	// THE SURFACE TAKES THE SIDE WITH ROOM. A trigger near the far edge of
	// the frame leaves the side it was asked for too short to hold the
	// annotation, and the annotation takes the opposite side — but only when
	// that one holds it whole. A frame with room on neither side keeps the
	// placement the caller asked for: a caller whose frame is its trigger is
	// placing the annotation outside that frame on purpose, and flipping it
	// there would move it onto the trigger rather than away from an edge.
	switch props.Placement {
	case Top:
		if above < 0 && below+surfH <= frame.Y {
			pos.Y = below
		}
	case Bottom:
		if below+surfH > frame.Y && above >= 0 {
			pos.Y = above
		}
	case Left:
		if before < 0 && after+surfW <= frame.X {
			pos.X = after
		}
	case Right:
		if after+surfW > frame.X && before >= 0 {
			pos.X = before
		}
	}

	// AND IT STAYS INSIDE THE WINDOW'S SIDES. Above or below a trigger near
	// a side of the frame, a surface centred on that trigger runs off the
	// edge; it is shifted inward until it is inside instead, which keeps
	// what it says readable and is what the platform does with an annotation
	// against a word near a margin. It keeps the step from the window's side
	// that it keeps from its own trigger, since a surface laid flush against
	// the edge reads as a thing cut off by it. The sides only: the ends of
	// the frame above and below are the flip's business, and a caller whose
	// frame is its own trigger places the annotation outside that frame on
	// purpose.
	if props.Placement == Top || props.Placement == Bottom {
		pos.X = min(pos.X, frame.X-surfW-gap)
		pos.X = max(pos.X, gap)
	}

	surfOff := op.Offset(pos).Push(gtx.Ops)
	rect := clip.RRect{Rect: image.Rectangle{Max: image.Pt(surfW, surfH)}, SE: r, SW: r, NE: r, NW: r}
	// The hairline is the whole of a still tooltip's edge: the fill is the
	// window's own plane and the shadow belongs to whatever places it. Two
	// fills rather than a stroke, because a stroke is centred on its path
	// and spends half its coverage outside the shape; the inset fill lands
	// the separator on whole pixels at the coverage the platform recorded.
	edgePx := max(gtx.Dp(1), 1)
	innerR := max(r-edgePx, 0)
	inner := clip.RRect{
		Rect: image.Rectangle{Min: image.Pt(edgePx, edgePx), Max: image.Pt(surfW-edgePx, surfH-edgePx)},
		SE:   innerR, SW: innerR, NE: innerR, NW: innerR,
	}
	paint.FillShape(gtx.Ops, vgcolor.Flatten(tok.platform.Separator, surface.Or(props.Surface, tok.platform.WindowBackground)), rect.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, tok.platform.WindowBackground, inner.Op(gtx.Ops))
	textOff := op.Offset(image.Pt(padH, padV)).Push(gtx.Ops)
	textCall.Add(gtx.Ops)
	textOff.Pop()
	surfOff.Pop()

	op.Defer(gtx.Ops, floating.Stop())
}
