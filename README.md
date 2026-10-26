# components

The component foundation of [Vibrant Gio](https://github.com/vibrantgio), a
design system for native desktop applications on macOS, Windows and Linux,
written in pure Go on [Gio](https://gioui.org). components is where the buttons,
inputs, lists, rich text, scrollbars, icons and layout primitives live, all
styled against the `tokens` and `theme` contract that lives one tier down in
[theme](https://github.com/vibrantgio/theme).

Gio gives you drawing primitives and an event loop, not a component set. A
button is yours to write, and so are its hover, focus, press and disabled
states, its pointer target, its Space/Enter activation and its
screen-reader label. components writes them once. There are two API shapes, and
which one a package uses follows from what it owns:

- **Themed components** — `button`, `input`, `chip`, `picker` and the
  others with a live entry point — take an
  `rx.Observable[theme.Theme]` plus a props struct and return an
  `rx.Observable[layout.Widget]`. The widget is rebuilt when the theme
  changes, so a window follows the OS between light and dark with no
  application code. Interaction state is allocated inside the component's
  `rx.Defer` scope, which is what keeps press and focus alive across the view
  rebuilds an MVU loop drives. The theme carries the whole look: colour,
  typography (the theme's shaper — see below), and `Density` — a control is
  at least the platform's regular control height, 24 dp, at Comfortable and its
  small one, 19 dp, at Compact, taller where its type role needs it, and the
  drawn control is also the pointer target, so
  Compact moves what a pointer lands on with the pixels.
- **Immediate-mode primitives** — `list`, `paragraph`, `scrollbar`, `layout` —
  take the frame's `layout.Context`, a `State` you allocate once and reuse
  across frames, and a per-frame `Style` resolved from tokens
  (`scrollbar.FromTokens`, `paragraph.FromTokens`). They are what the themed
  components are built out of.

Both shapes have a pure render path — `button.Render`, `input.RenderCheckbox`,
`paragraph.Render`, … — that takes resolved tokens and an explicit state struct
and draws one frame with no event handling. That is what the golden-image
tests drive. These signatures take the same token types the live
paths do — a `tokens.TextStyle` for the role they draw text in and a
`tokens.Density` for the control height and padding — so a static caller gets
the full metrics, not sizes only. Pass `tokens.DefaultTypography.SystemFont`
(or `.Body` for reading text) and `tokens.Comfortable` for the default desktop look. Drive
components through their theme-driven entry points (`button.Button`,
`input.TextField`, …) unless you are rendering a static frame.

## Where it sits

The tier above theme in the stack. components imports
[mvu](https://github.com/vibrantgio/mvu),
[theme](https://github.com/vibrantgio/theme) — the `theme` and `tokens`
contract, with `color` and `typeset` — and the support libraries
[ivg](https://github.com/vibrantgio/ivg) and
[svg](https://github.com/vibrantgio/svg); its module also requires
[font](https://github.com/vibrantgio/font). The
[organization page](https://github.com/vibrantgio) has the full tier table.

```sh
go get github.com/vibrantgio/components
```

## Packages

| Package | |
| --- | --- |
| `alert` | The status signal for a situation: a rounded box — a leading status symbol, a title and an arbitrary body slot — standing in the page flow until the situation resolves. It stands on the content's fill inside the platform's separator hairline, its title in the platform's label colour, and the status is carried by the symbol alone, in the platform's system colour for it: systemGreen, systemOrange, systemRed, systemBlue. Info is systemBlue and never the accent. It holds words about the situation, never a control: an action on the situation stands beside it. |
| `attachment` | The recipe every floating surface places itself with; not a component, and it draws nothing. It holds the available room, set at the window by `WithAvailableRoom` and moved, with the `Surfaces` a layout collects, by every layout that places a child at an offset (`AvailableRoomAt`, `PlaceSurfacesAt`, `PlacingFlex`), and the arbitration of placement against fit inside it: `TakeSide` keeps the preferred side while the room holds the surface whole there and otherwise gives it the opposite side, `ShiftInward` and `Inward` keep it inside the room along the other axis, and `SideRoom` is the length a surface taller than its side scrolls in. |
| `badge` | The inline annotation: the system's own word about a thing, set at the size of its own type and off the control-height scale entirely. It speaks as a word, a count or a symbol — a word or a count filled with the platform's system colour for the status under white, the platform's own count badge; a symbol standing bare in that system colour, or on a disc of it under a white sign. Five values differing in hue alone (`Neutral`, which has no system colour of its own and is filled with systemGray, or drawn bare in the secondary label colour, and `Success`/`Warning`/`Error`/`Info` for the four statuses) and no emphasis axis, because a badge is read rather than used. `Props.OnDismiss` adds the close mark, drawn 8 dp across with a 24 dp pointer target centred on it. |
| `breadcrumb` | The control for going back up the hierarchy: a chevron-separated row of location segments — each step a link, the last where you are. Every segment is drawn in the platform's `Label` colour and the chevrons between them in `SecondaryLabel`, and a segment is clickable exactly when its own `OnClick` is set. `Breadcrumb` takes the trail when the stream is built; `Trail` takes it per frame, for a path that changes as the user navigates, and routes each click by the segment's own key rather than by the position it stood in. No overflow behaviour: every segment renders, on one row, and a trail deeper than its constraint is clipped rather than collapsed. |
| `bench` | `BenchFrame`, the shared per-frame benchmark harness every component's benchmarks run through. |
| `button` | The button: text or icon-only, in three emphases (`Filled`, `Tonal`, `Ghost`; `Props.Fill` and `Props.Foreground` let a caller set the filled button's colours), hover/focus/press/disabled, keyboard activation, density-sized, the drawn control its own pointer target; clicks arrive as a callback or as an MVU message. `Props.Icon` is a painter, `func(gtx layout.Context, sizePx int, col color.NRGBA)`; with no `Title` it makes the button a square icon-only control. `RenderBordered`, `BorderedFace` and `BorderedSegments` draw the bordered button the toolbar uses and a row of joined segments; `Render` and `RenderIcon` are the pure paths. |
| `button/gallery` | A runnable page showing the button in every visual state in both appearances. |
| `chip` | The chip: the small control content offers, drawn as the platform draws an ordinary small control and defined by its purpose — `Assist`, `Filter` (the one that toggles), `Input` (carries a dismiss mark) and `Suggestion`. `Chip` is the live path, `Render` the pure one, `Resolve` its colours for a purpose and state. |
| `composite` | Paints an overlay over what the frame has already drawn, composited in encoded sRGB as the platform composites it: `Flatten` paints an overlay over mixed content, and `Frame`, called first in a window's root layout, records the frame it composites against. Where the surface beneath is one known fill, `theme/color.Flatten` is the answer instead. |
| `cache` | `FrameCache`, an op-recording cache that replays a component's recorded draw commands on frames where its inputs have not changed. |
| `coordination` | **Deprecated** — use [`mvu/stream`](https://github.com/vibrantgio/mvu)`.Value`. `Subject`, a typed broadcast channel for signals between components, with `BufCapPointer` and `BufCapSignal` buffer depths; a subscription's slot is released on unsubscribe, and `MaxSubscribers` live subscriptions at once is its leak limit. |
| `elide` | Shortens a line of text to the width it has: `Middle` keeps the start and the end of an address on either side of one `Ellipsis`, measured with the same role and shaper the draw uses. |
| `gallery` | A nested module, the runnable gallery: every component in every visual state, with live interaction, and the page of everything at once. |
| `gallery/inventory` | Draws every family of the design system — foundations, components, patterns and a prose sample — as one column of labelled sections, each a pure function of the `PlatformColors` it is handed. |
| `gallery/palette` | Draws the platform's colour set: every name AppKit answers for, with its value in both appearances, and where a name is a coverage rather than an opaque colour. |
| `golden` | The headless-Gio golden-image harness: `Capture`, `Render`, `Compare` and `PixelDiff` for a frame; `StaticTheme`, `Window` and `WindowGolden` for whole windows of themed layers; `CoverageRow` and `FitCorner` for measuring edges and corner radii off a capture. |
| `icon` | A name→icon registry holding icons in either SVG (`vibrantgio/svg`) or IVG (`vibrantgio/ivg`) form. |
| `icon/gallery` | A runnable page drawing one SVG and one IVG icon side by side. |
| `icons` | The design system's own icon set: symbols drawn on one grid at one weight, addressed by a `Name` that says what the symbol depicts, and resolved to the host platform's drawing at run time. `Mark` returns the painter a control's icon slot takes, `Names` lists the set, and `Register` fills an `icon.Registry` with it; `New` builds the set for a given platform. |
| `initial` | `Value[T]`, a typed "not set yet" cell for state that cannot be computed until the first frame has laid out — instead of a magic sentinel. |
| `input` | Text field, search field, checkbox, radio and dropdown, on the same state and props contract as `button`; the dropdown is `picker`'s form trigger under input's name. The search field is the text field with two slots added to its structure — the looking glass that names it and the clear mark that empties it — both spent out of the field's own width, and clearing reports the empty query the way a keystroke reports any other. |
| `keyed` | `Deferred`, a key→state registry that keeps per-row state attached to its item across list reorders, inserts and deletes. |
| `layout` | Spacing, inset and spacer helpers, row/column wrappers, a pill clip, and `FocusGroup` for keyboard focus across a fixed set of items. |
| `list` | Virtual-scrolling list — only the visible rows lay out. `Layout` for the bare list, `LayoutScrollbar` to draw a bar in a lane of its own, which exists only while the rows overflow, or overlaid; `LayoutSelectable` and `LayoutSelectableScrollbar` make the list one focus target whose selection the arrow keys, Home and End move, telling each row whether it is the selected one so the caller draws the selection. |
| `pagination` | The control for moving between numbered pages of content: a row of numbered page cells flanked by prev/next chevrons. Only the current page carries a fill: the platform's accent under the text that reads on it. Every other cell is unfilled — a page the control can take the reader to is drawn in the platform's link colour, any other page number in its label colour — so the row says "this one" with the one filled cell and in nothing else. The cells are drawn natively rather than through `button`, whose Comfortable padding would truncate a page digit inside a density-sized square. `Props.Page` and `PageCount` are plain ints, so a page change rebuilds the stream; every page in `[1, PageCount]` renders, with no ellipsis collapse. |
| `picker` | The pick-one-from-many control: a trigger showing the chosen item's title and a menu offering the others. `Field` is the form variant, the platform's pop-up button; `Toolbar` the chrome variant; `Menu` the open surface; `RenderField`, `RenderToolbar` and `RenderMenu` the pure paths. |
| `pointershape` | Declares the shape the platform's pointer takes over a region — the arrow, the I-beam, the hand — bounded to that region: `Over` for a rectangle, `OverSize` for a size at the origin. |
| `progress` | The progress signal for a task whose end is not known: the platform's indeterminate indicator, a 16 dp wheel of eight grey strokes turning on Gio's frame clock. `Indicator` draws the wheel, `Region` and `Row` place it. It takes no pointer and no focus. |
| `paragraph` | Content as a run of styled text wrapped into lines: styled spans, spans set on a rounded chip, and hyperlink spans with hover, focus ring and Tab traversal. Built directly on Gio's shaper. |
| `scrollarea` | A horizontally scrolling viewport for content that must not be reflowed to fit — a preformatted block, an over-wide diagram. The child lays out at its natural width; content past an edge is clipped at the edge, and an optional bar is drawn at the area's own bottom edge, with a lane of its own when the bar stands and the content overflows. It claims the horizontal axis only, so it nests inside a vertical `list` without fighting it. |
| `scrollbar` | The standalone scrollbar for any scrollable region — track, draggable thumb, click-the-track scrolling — styled from tokens. Absent while the content fits, and faded out a second after it stops moving, as the desktop platforms' overlay bars are. `Style.Always()` is the standing bar: an 11 dp thumb on a pill track, 3 dp inside a 17 dp lane. `list.LayoutScrollbar` draws this one. |
| `toast` | The status signal for an event: a small surface at the floating level that appears when a notification is raised and leaves by itself after a set time. Its fill is the window background, as every floating surface's on this platform is, and its message is in the platform's label colour. Its status shows in the leading edge, in the platform's system colour for it, and never in the fill. It draws one toast and no queue: receiving, placing, stacking and timing notifications, and the shadow a placed toast casts, are its caller's. |
| `tooltip` | The signal naming a control or explaining another signal on demand: a small annotation at the floating level beside its trigger, appearing by itself after a short delay on hover or focus and leaving when they do. Its fill is the window background inside the platform's separator hairline, with its text in the platform's label colour, the same in both appearances, so it reads as an annotation about the thing rather than as a pane. It holds text only, never a control: anything the user must operate is the job for a popover. Arbitration is frame state rather than a bus — `Props.Arbiter` names the set, one per window, and a tooltip is visible exactly while it holds that set's top, so the claim a finished dwell makes *is* the previous tooltip's dismissal. A nil `Arbiter` gets the tooltip one of its own, so sharing one is the explicit act. `DefaultDelay` resolves from the token motion scale's `DurXSlow` stop (500 ms), and the live form re-times from the theme's `Motion` observable. It is placed by `attachment`. |

## Usage

The theme owns the typeface: every component that draws text shapes
with the theme's `Typography.Shaper()`, so there is no shaper to build and
`Props.Shaper` stays nil except as a deliberate per-instance override. The
no-gofont lint in this repository fails `go test` on any
`gioui.org/font/gofont` import, so a component cannot fall back to gofont.

One row's checkbox in a todo list, condensed:

```go
// Row is one todo line: a components checkbox toggling completion, the todo text,
// and a delete icon. Every event routes through mvu.MessageOp, so the
// reducers are the only state writers.
func Row(typ Type, th rx.Observable[theme.Theme], p Palette, item Todo) layout.Widget {
	// th is a static snapshot (rx.Of), so First() resolves synchronously.
	cb, _ := input.Checkbox(th, input.CheckboxProps{
		Description: "completed",
		Checked:     item.Completed,
		Message:     ToggleTodo{Id: item.Id},
	}).First()
	// ... the todo text and the delete icon, then a Flex row over the three.
}
```

A button is the same shape — this one is from a rename dialog, where the
dialog owns the focus tag by passing its own `Clickable`. Note what
is absent: no `Shaper` prop, because the theme supplies it:

```go
submit := button.Button(th, button.Props{
	Title:     "Rename",
	Clickable: &submitClick,
	OnClick:   func(gtx layout.Context) { /* validate, then mvu.MessageOp */ },
})
```

`th` is the theme observable the window hands to your layer builder;
`theme/system.LiveTheme` supplies one that tracks the OS appearance live. `Message`
is the MVU path — the component adds `mvu.MessageOp{Message: …}` to the
frame's ops and the runtime delivers it to `Update`. FRP-style applications
use `OnClick` instead, and are handed the frame's `layout.Context` so they can
still emit a message from inside the callback. Buttons fill the width
they are given and are at least the theme's `Density.ControlHeight` tall, and the
drawn control is the pointer target too: a control claims the pixels it shows and no more, so two
controls side by side never contend for the same press.

The **gallery** shows every component in every visual state, with live
interaction:

```sh
go run github.com/vibrantgio/components/gallery@latest   # or: cd gallery && go run .
```

It is a nested module — `github.com/vibrantgio/components/gallery`, whose tags
carry the directory as a prefix — so that components itself does not depend on
effects, which the gallery's spring-button page needs.

## For coding assistants

Read this repository's [AGENTS.md](./AGENTS.md) before you write code against this module.

Build and test the root module with `go build ./...` and `go test ./...`.

To regenerate golden images, name the test packages and put the flag after them.
This module defines the flag `golden.update` in `golden/golden.go`.
Placed before the packages, `go test` tests the current directory only.
Name only packages that store golden images, because a test binary rejects a flag it never declared.

```sh
go test ./button -golden.update
```

## License

MIT — see [LICENSE](./LICENSE).
