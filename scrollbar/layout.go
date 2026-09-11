package scrollbar

import (
	"image"
	"image/color"
	"math"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// fade returns the thumb's opacity multiplier in [0,1] for this frame and
// records what the frame saw, scheduling the redraws the fade needs.
//
// Activity is a change in the viewport fractions — the only signal a
// scrollbar has that its content is moving — together with hover and drag on
// its own areas. The first frame of a State's life counts as activity too,
// so a bar is opaque the moment it appears and a single-frame capture (a
// golden) never catches it mid-fade.
//
// It is a method on Style because the timings are the style's, and it writes
// through the State pointer because the timeline is the state's.
func (s Style) fade(gtx layout.Context, state *State, viewportStart, viewportEnd float32) float32 {
	if s.FadeDelay <= 0 {
		return 1
	}
	active := !state.seen ||
		viewportStart != state.start || viewportEnd != state.end ||
		state.IndicatorHovered() || state.TrackHovered() || state.Dragging()
	state.seen, state.start, state.end = true, viewportStart, viewportEnd
	if active {
		state.lastActive = gtx.Now
	}
	idle := gtx.Now.Sub(state.lastActive)
	if idle < s.FadeDelay {
		// Wake up when the delay expires; nothing to draw until then.
		gtx.Execute(op.InvalidateCmd{At: state.lastActive.Add(s.FadeDelay)})
		return 1
	}
	if s.FadeDuration <= 0 {
		return 0
	}
	t := float32(idle-s.FadeDelay) / float32(s.FadeDuration)
	if t >= 1 {
		return 0
	}
	gtx.Execute(op.InvalidateCmd{})
	return 1 - t
}

// Layout draws the scrollbar along axis and registers its gesture areas.
// viewportStart and viewportEnd describe the visible fraction of the content
// in the range [0,1] (see FromListPosition).
//
// The bar renders whenever the viewport shows less than the full content and
// renders nothing (zero dimensions) when everything fits. It occupies the
// full major axis of the incoming constraints and Width() along the minor
// axis.
//
// With a non-zero Style.FadeDelay the thumb fades out once the content stops
// moving and the pointer leaves the gutter; it keeps its size and its hit
// areas throughout, so nothing reflows and a hover brings it back. See fade.
//
// While Style.Matches holds the places of a search query's matches, the bar
// paints each of them in the track and the thumb passes under them: they are
// drawn after it, so none is ever hidden, and they keep their opacity while
// the thumb fades. Nothing is painted for a query with no matches, and the
// bar renders nothing at all while the content fits, matches or not. See
// paintMatches.
func (s Style) Layout(gtx layout.Context, state *State, axis layout.Axis, viewportStart, viewportEnd float32) layout.Dimensions {
	if viewportStart <= 0 && viewportEnd >= 1 {
		// Everything fits: no scrollbar.
		return layout.Dimensions{}
	}

	// Pin the constraints in an axis-independent way, then convert to the
	// correct representation for the current axis.
	convert := axis.Convert
	maxMajorAxis := convert(gtx.Constraints.Max).X
	gtx.Constraints.Min.X = maxMajorAxis
	gtx.Constraints.Min.Y = gtx.Dp(s.Width())
	gtx.Constraints.Min = convert(gtx.Constraints.Min)
	gtx.Constraints.Max = gtx.Constraints.Min

	// Process events against last frame's areas before reading hover state.
	state.Update(gtx, axis, viewportStart, viewportEnd)

	// Fade after the gesture areas have been updated, so this frame's hover
	// and drag state counts as activity. The track's areas are registered
	// below whatever the opacity, so a faded-out bar can still be hovered
	// back into view.
	thumbColor := s.ThumbColor
	thumbColor.A = uint8(float32(thumbColor.A)*s.fade(gtx, state, viewportStart, viewportEnd) + 0.5)

	inset := layout.Inset{
		Top:    s.TrackPadding,
		Bottom: s.TrackPadding,
		Left:   s.TrackPadding,
		Right:  s.TrackPadding,
	}

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			// Lay out the draggable track underneath the thumb.
			area := image.Rectangle{Max: gtx.Constraints.Min}
			pointerArea := clip.Rect(area)
			defer pointerArea.Push(gtx.Ops).Pop()
			state.AddDrag(gtx.Ops)

			// Stack a normal clickable area on top of the draggable area
			// to capture non-dragging clicks.
			defer pointer.PassOp{}.Push(gtx.Ops).Pop()
			defer pointerArea.Push(gtx.Ops).Pop()
			state.AddTrack(gtx.Ops)

			paint.FillShape(gtx.Ops, s.TrackColor, clip.Rect(area).Op())
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				// Work in axis-independent space.
				gtx.Constraints.Min = convert(gtx.Constraints.Min)
				gtx.Constraints.Max = convert(gtx.Constraints.Max)

				// Compute the pixel size and position of the thumb
				// within the track.
				trackLen := gtx.Constraints.Min.X
				viewStart := int(math.Round(float64(viewportStart) * float64(trackLen)))
				viewEnd := int(math.Round(float64(viewportEnd) * float64(trackLen)))
				thumbLen := max(viewEnd-viewStart, gtx.Dp(s.ThumbMinLen))
				if viewStart+thumbLen > trackLen {
					viewStart = trackLen - thumbLen
				}
				thumbDims := convert(image.Point{
					X: thumbLen,
					Y: gtx.Dp(s.ThumbMinorWidth),
				})
				radius := gtx.Dp(s.ThumbCornerRadius)

				// The thumb draws inside its own scope so that its offset
				// is popped before anything after it is drawn.
				func() {
					offset := convert(image.Pt(viewStart, 0))
					defer op.Offset(offset).Push(gtx.Ops).Pop()
					paint.FillShape(gtx.Ops, thumbColor, clip.RRect{
						Rect: image.Rectangle{Max: thumbDims},
						SW:   radius,
						NW:   radius,
						NE:   radius,
						SE:   radius,
					}.Op(gtx.Ops))

					// Register the thumb's pointer hit area.
					area := clip.Rect(image.Rectangle{Max: thumbDims})
					defer pointer.PassOp{}.Push(gtx.Ops).Pop()
					defer area.Push(gtx.Ops).Pop()
					state.AddIndicator(gtx.Ops)
				}()

				s.paintMatches(gtx, convert, trackLen, gtx.Constraints.Min.Y)

				return layout.Dimensions{Size: convert(gtx.Constraints.Min)}
			})
		},
	)
}

// paintMatches paints where each of Style.Matches lies inside the track:
// across the track's minor extent, MatchLen long along its major axis,
// centred on the match's fraction of the track and pushed inside the track's
// two ends so the first and the last are painted whole. trackLen and minor
// are the track's extents in pixels, in axis-independent space; convert maps
// that space to the axis being drawn.
//
// It is called after the thumb has been drawn, which is what keeps the thumb
// from ever hiding a match: the thumb passes under them rather than over
// them. And it is called with the fills at their full opacity, outside the
// fade the thumb takes, because the places of the matches stay in the track
// for as long as the query does.
//
// The current match is painted last so that it is the one that survives where
// two matches land on the same pixels.
func (s Style) paintMatches(gtx layout.Context, convert func(image.Point) image.Point, trackLen, minor int) {
	length := gtx.Dp(s.MatchLen)
	if len(s.Matches) == 0 || length <= 0 || trackLen <= 0 || minor <= 0 {
		return
	}
	length = min(length, trackLen)
	paintAt := func(fraction float32, fill color.NRGBA) {
		start := int(math.Round(float64(clamp1(fraction))*float64(trackLen))) - length/2
		start = min(max(start, 0), trackLen-length)
		rect := image.Rectangle{
			Min: convert(image.Pt(start, 0)),
			Max: convert(image.Pt(start+length, minor)),
		}
		paint.FillShape(gtx.Ops, fill, clip.Rect(rect).Op())
	}
	for i, fraction := range s.Matches {
		if i == s.Current {
			continue
		}
		paintAt(fraction, s.MatchFill)
	}
	if s.Current >= 0 && s.Current < len(s.Matches) {
		paintAt(s.Matches[s.Current], s.CurrentMatchFill)
	}
}
