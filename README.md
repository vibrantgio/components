# prism

The component foundation of [Vibrant Gio](https://github.com/vibrantgio), a
design system for native desktop applications on macOS, Windows and Linux,
written in pure Go on [Gio](https://gioui.org). prism is where the buttons,
inputs, lists, rich text, scrollbars, icons and layout primitives live —
together with the `tokens` and `theme` contract every layer above it styles
against.

Gio gives you drawing primitives and an event loop, not a component set. A
button is yours to write, and so are its hover, focus, press and disabled
states, its 44 dp hit target, its Space/Enter activation and its screen-reader
label. prism writes them once. There are two API shapes, and which one a
package uses follows from what it owns:

- **Themed components** — `button`, `input` — take an
  `rx.Observable[theme.Theme]` plus a props struct and return an
  `rx.Observable[layout.Widget]`. The widget is rebuilt when the theme
  changes, so a window follows the OS between light and dark with no
  application code. Interaction state is allocated inside the component's
  `rx.Defer` scope, which is what keeps press and focus alive across the view
  rebuilds an MVU loop drives.
- **Immediate-mode primitives** — `list`, `richtext`, `scrollbar`, `layout` —
  take the frame's `layout.Context`, a `State` you allocate once and reuse
  across frames, and a per-frame `Style` resolved from tokens
  (`scrollbar.FromTokens`, `richtext.FromTokens`). They are what the themed
  components and cadence's patterns are built out of.

Both shapes have a pure render path — `button.Render`, `input.RenderCheckbox`,
`richtext.Render`, … — that takes resolved tokens and an explicit state struct
and draws one frame with no event handling. That is what the golden-image
tests drive, and what to use for static rendering.

## Where it sits

Tier 2 of the stack — `mvu → spectrum → prism → pulse → cadence → markdown`.
prism imports [mvu](https://github.com/vibrantgio/mvu) and the support
libraries [ivg](https://github.com/vibrantgio/ivg) and
[svg](https://github.com/vibrantgio/svg);
[pulse](https://github.com/vibrantgio/pulse),
[cadence](https://github.com/vibrantgio/cadence) and
[markdown](https://github.com/vibrantgio/markdown) are built on it —
`pulse/springbutton` is `prism/button` with a physics-driven press, `cadence`'s
table is built on `prism/list` and its modal on `prism/button`, and every
cadence pattern takes its visual values from this repository's tokens.
The [organization page](https://github.com/vibrantgio) has the full tier table.

```sh
go get github.com/vibrantgio/prism
```

Every module in the organization is on gioui.org v0.10.1,
github.com/reactivego/rx v0.3.0 and Go 1.25.1.

## Packages

| Package | |
| --- | --- |
| `a11y` | OS accessibility preferences — reduce motion, high contrast, larger text — polled and published as an `rx.Observable[A11yPrefs]`. |
| `bench` | `BenchFrame`, the shared per-frame benchmark harness every component's benchmarks run through. |
| `button` | The button: text or icon-only, hover/focus/press/disabled, keyboard activation, 44 dp hit target; clicks arrive as a callback or as an MVU message. |
| `cache` | `FrameCache`, an op-recording cache that replays a widget's recorded draw commands on frames where its inputs have not changed. |
| `coordination` | `Subject`, the typed broadcast channel for cross-widget signals — drag, modal, tooltip — with a documented one-frame delivery lag. |
| `icon` | A name→icon registry holding icons in either SVG (`vibrantgio/svg`) or IVG (`vibrantgio/ivg`) form. |
| `initial` | `Value[T]`, a typed "not set yet" cell for state that cannot be computed until the first frame has laid out — instead of a magic sentinel. |
| `input` | Text field, checkbox, radio and dropdown, on the same state and props contract as `button`. |
| `keyed` | `Deferred`, a key→state registry that keeps per-row widget state attached to its item across list reorders, inserts and deletes. |
| `layout` | Spacing, inset and spacer helpers, row/column wrappers, a pill clip, and `FocusGroup` for keyboard focus across a fixed set of items. |
| `list` | Virtual-scrolling list — only the visible rows lay out. `Layout` for the bare list, `LayoutScrollbar` to draw a bar in a reserved gutter or overlaid. |
| `richtext` | The inline styled-text primitive: styled spans, wrapped paragraphs, and hyperlink spans with hover, focus ring and Tab traversal. Built directly on Gio's shaper. |
| `scrollbar` | The standalone scrollbar for any scrollable region — track, draggable thumb, click-the-track scrolling — styled from tokens. `list.LayoutScrollbar` draws this one. |
| `theme` | `Theme`: one `rx.Observable` per token category, so a consumer subscribes to just the categories it reads. `Default()` and `AutoLightDark()` construct one. |
| `tokens` | The typed design values: colour scales and semantic `ColorTokens`, the MD3 type scale, and the 4-pt spacing, radius, elevation and motion scales. |

`theme` and `tokens` move down into
[spectrum](https://github.com/vibrantgio/spectrum) in Phase B, so the theme
runtime sits beneath the components it themes. Both packages remain here as
type aliases and re-exported variables, so `prism/theme` and `prism/tokens`
keep working unchanged.

## Usage

Build one shaper per window at layer scope and thread it down — never per
frame, and never shared between windows, because a shaper owns caches and is
not safe for concurrent use:

```go
shaper := text.NewShaper(text.WithCollection(style.FontFaces()))
```

Then, condensed from `list.go` in
[workbench/todos](https://github.com/vibrantgio/workbench/tree/master/todos) —
the smallest complete Vibrant Gio application — the virtual list and one row's
checkbox:

```go
// List renders the todos inside a rounded pane using the prism virtual list.
func List(shaper *text.Shaper, th rx.Observable[theme.Theme], model Model) layout.Widget {
	listState := prismlist.NewState() // allocate once; reused every frame
	rows := make([]layout.Widget, len(model.List))
	for i := range model.List {
		rows[i] = Row(shaper, th, model.List[i])
	}
	return func(gtx layout.Context) layout.Dimensions {
		return prismlist.Layout(gtx, listState, rows,
			func(gtx layout.Context, row layout.Widget) layout.Dimensions {
				return layout.UniformInset(Padding).Layout(gtx, row)
			})
	}
}

// Row is one todo line: a prism checkbox toggling completion, the todo text,
// and a delete icon. Every event routes through mvu.MessageOp, so the
// reducers are the only state writers.
func Row(shaper *text.Shaper, th rx.Observable[theme.Theme], item Todo) layout.Widget {
	// th is a static snapshot (rx.Of), so First() resolves synchronously.
	cb, _ := input.Checkbox(th, input.CheckboxProps{
		Description: "completed",
		Checked:     item.Completed,
		Message:     ToggleTodo{Id: item.Id},
	}).First()
	// ... the label and the delete icon, then a Flex row over the three.
}
```

A button is the same shape — this one is from `watchlist/renamemodal.go`,
where the modal owns the focus tag by passing its own `Clickable`:

```go
submit := button.Button(th, button.Props{
	Label:     "Rename",
	Clickable: &submitClick,
	Shaper:    shaper,
	OnClick:   func(gtx layout.Context) { /* validate, then mvu.MessageOp */ },
})
```

`th` is the theme observable the window hands to your layer builder;
`spectrum/window` supplies one that tracks the OS appearance live. `Message`
is the MVU path — the component adds `mvu.MessageOp{Message: …}` to the
frame's ops and the runtime delivers it to `Update`. FRP-style applications
use `OnClick` instead, and are handed the frame's `layout.Context` so they can
still emit a message from inside the callback. Prism buttons fill the width
they are given and are at least 44 dp tall, so a fixed-size button is laid out
inside a box with constrained width.

The **gallery** shows every component in every visual state, with live
interaction:

```sh
go run github.com/vibrantgio/prism/gallery@latest   # or: cd gallery && go run .
```

It is a nested module — `github.com/vibrantgio/prism/gallery`, whose tags
carry the directory as a prefix (`gallery/v0.1.2`, not `v0.1.2`) — so that
prism itself does not depend on pulse, which the gallery's spring-button page
needs.

## For coding assistants

Read the canonical guide before writing code against this module — the module
inventory with current tags, the application skeleton, MVU and rx semantics,
typography, and the pitfalls that are not guessable:

<https://raw.githubusercontent.com/vibrantgio/.github/master/llms.txt>

[`AGENTS.md`](./AGENTS.md) in this repository has the build, test and
golden-image commands.

## Status

Honest about what does not work yet:

- **The typeface.** Vibrant Gio ships Roboto, but there is no typography token
  yet: `button`, `input.TextField` and `input.Dropdown` each build a Go-fonts
  shaper for themselves when `Props.Shaper` is nil, and nothing warns you —
  the application renders, in the wrong typeface. Always pass
  `text.NewShaper(text.WithCollection(style.FontFaces()))`. (`richtext` takes
  its shaper positionally, so there is nothing to forget there; `checkbox` and
  `radio` draw no text and need none.) Phase C moves the typeface into the
  theme token and removes the fallbacks.
- **The layering.** `spectrum`, the theme runtime, still imports prism for the
  token contract, so the theme sits above what it themes. Phase B moves
  `theme` and `tokens` down into spectrum and leaves deprecated aliases here.
- **`theme.AutoLightDark()` reads the clock, not the OS** — hours 7–17 light,
  otherwise dark. For real OS dark-mode and accent tracking use spectrum's
  `system.LiveTheme`.
- **`icon.Registry` ships empty.** Nothing populates it yet. For Material icons
  today, render `golang.org/x/exp/shiny/materialdesign/icons` data through
  `ivg/raster/gio`; `button.Props.Icon` wants a `clip.Path` painter, not a
  widget.
- **`a11y` on Linux returns all-false** — no reliable cross-desktop API without
  desktop-environment-specific dependencies. macOS and Windows report real
  preferences.

## License

MIT — see [LICENSE](./LICENSE).
