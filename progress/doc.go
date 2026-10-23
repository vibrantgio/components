// Package progress provides the Vibrant Gio progress signal: the signal
// showing how far a task the system is doing has come. This package draws the
// platform's indeterminate indicator, the signal for a task whose end is not
// known. It is read and never operated: it takes no pointer and no keyboard
// focus, and it leaves when the task ends.
//
// The indicator is a 16 dp wheel of eight grey strokes, the darkest stroke
// stepping clockwise about every 115 ms, as the reference's "What the
// indeterminate indicator measures" reads it off System Settings. The wheel
// turns on Gio's frame clock: each frame draws the step the clock is at and
// requests the frame at the next step. A frame with no clock, which is what a
// headless golden lays out, draws the step [Props.Phase] names and requests no
// frame, so a stored image is the same on every run.
//
// [Indicator] draws the wheel alone. Two helpers place it with the
// application's own text, a short phrase naming the task and ending with an
// ellipsis, set at 13 sp in the label colour:
//
//   - [Region] is the shape for a region whose content is still arriving: the
//     indicator above the text, centred both ways in the region's frame as one
//     group, [Gap] between them.
//   - [Row] is the shape for a row: the text at the row's leading end and the
//     indicator at its trailing end, both on the row's middle.
package progress
