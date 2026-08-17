package scrollbar

import (
	"image"
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

	thumbColor := s.ThumbColor
	if state.IndicatorHovered() || state.Dragging() {
		thumbColor = s.ThumbHoverColor
	}
	// Fade after the gesture areas have been updated, so this frame's hover
	// and drag state counts as activity. The track's areas are registered
	// below whatever the opacity, so a faded-out bar can still be hovered
	// back into view.
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

				// Draw the thumb.
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

				return layout.Dimensions{Size: convert(gtx.Constraints.Min)}
			})
		},
	)
}
