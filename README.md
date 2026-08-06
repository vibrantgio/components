# prism

The component foundation of [Vibrant Gio](https://github.com/vibrantgio), a
design system for native desktop applications on macOS, Windows and Linux,
written in pure Go on [Gio](https://gioui.org). prism is where the buttons,
inputs, lists, rich text, scrollbars, icons and layout primitives live, all
styled against the `tokens` and `theme` contract that lives one tier down in
[spectrum](https://github.com/vibrantgio/spectrum).

Gio gives you drawing primitives and an event loop, not a component set. A
button is yours to write, and so are its hover, focus, press and disabled
states, its 44 dp pointer target, its Space/Enter activation and its
screen-reader label. prism writes them once. There are two API shapes, and
which one a package uses follows from what it owns:

- **Themed components** — `button`, `input` — take an
  `rx.Observable[theme.Theme]` plus a props struct and return an
  `rx.Observable[layout.Widget]`. The widget is rebuilt when the theme
  changes, so a window follows the OS between light and dark with no
  application code. Interaction state is allocated inside the component's
  `rx.Defer` scope, which is what keeps press and focus alive across the view
  rebuilds an MVU loop drives. The theme carries the whole look: colour,
  typography (the theme's shaper — see below), and `Density` — the drawn
  control is 36 dp Comfortable or 28 dp Compact, while the pointer target
  keeps the 44 dp WCAG floor by extending beyond the drawn bounds, so Compact
  shrinks the pixels, never the clickable area.
- **Immediate-mode primitives** — `list`, `richtext`, `scrollbar`, `layout` —
  take the frame's `layout.Context`, a `State` you allocate once and reuse
  across frames, and a per-frame `Style` resolved from tokens
  (`scrollbar.FromTokens`, `richtext.FromTokens`). They are what the themed
  components and cadence's patterns are built out of.

Both shapes have a pure render path — `button.Render`, `input.RenderCheckbox`,
`richtext.Render`, … — that takes resolved tokens and an explicit state struct
and draws one frame with no event handling. That is what the golden-image
tests drive. As of v0.2.0 these signatures take the same token types the live
paths do — a `tokens.TextStyle` for the role they draw text in and a
`tokens.Density` for the control height and padding — so a static caller gets
the full metrics, not sizes only. Pass `tokens.DefaultTypography.LabelLarge`
(or `.BodyLarge`) and `tokens.Comfortable` for the default desktop look. Drive
components through their theme-driven entry points (`button.Button`,
`input.TextField`, …) unless you are rendering a static frame.

## Where it sits

Tier 2 of the stack — `mvu → spectrum → prism → pulse → cadence → markdown`.
prism imports [mvu](https://github.com/vibrantgio/mvu),
[spectrum](https://github.com/vibrantgio/spectrum) — the `theme` and `tokens`
contract — and the support libraries [ivg](https://github.com/vibrantgio/ivg)
and [svg](https://github.com/vibrantgio/svg);
[pulse](https://github.com/vibrantgio/pulse),
[cadence](https://github.com/vibrantgio/cadence) and
[markdown](https://github.com/vibrantgio/markdown) are built on it —
`pulse/springbutton` is `prism/button` with a physics-driven press, `cadence`'s
table is built on `prism/list` and its modal on `prism/button`.
The [organization page](https://github.com/vibrantgio) has the full tier table.

```sh
go get github.com/vibrantgio/prism
```

Every module in the organization is on gioui.org v0.10.1,
github.com/reactivego/rx v0.3.0 and Go 1.25.1.

## Packages

| Package | |
| --- | --- |
| `bench` | `BenchFrame`, the shared per-frame benchmark harness every component's benchmarks run through. |
| `button` | The button: text or icon-only, hover/focus/press/disabled, keyboard activation, density-sized with a 44 dp pointer target; clicks arrive as a callback or as an MVU message. |
| `cache` | `FrameCache`, an op-recording cache that replays a widget's recorded draw commands on frames where its inputs have not changed. |
| `coordination` | `Subject`, the typed broadcast channel for cross-widget signals — drag, modal, tooltip — with a documented one-frame delivery lag. |
| `golden` | The organization's headless-Gio golden-image harness: `Capture`, `Render` and `PixelDiff`. Exported so callers outside prism drive one capture path instead of inlining their own. |
| `icon` | A name→icon registry holding icons in either SVG (`vibrantgio/svg`) or IVG (`vibrantgio/ivg`) form. |
| `initial` | `Value[T]`, a typed "not set yet" cell for state that cannot be computed until the first frame has laid out — instead of a magic sentinel. |
| `input` | Text field, checkbox, radio and dropdown, on the same state and props contract as `button`. |
| `keyed` | `Deferred`, a key→state registry that keeps per-row widget state attached to its item across list reorders, inserts and deletes. |
| `layout` | Spacing, inset and spacer helpers, row/column wrappers, a pill clip, and `FocusGroup` for keyboard focus across a fixed set of items. |
| `list` | Virtual-scrolling list — only the visible rows lay out. `Layout` for the bare list, `LayoutScrollbar` to draw a bar in a reserved gutter or overlaid. |
| `richtext` | The inline styled-text primitive: styled spans, wrapped paragraphs, and hyperlink spans with hover, focus ring and Tab traversal. Built directly on Gio's shaper. |
| `scrollbar` | The standalone scrollbar for any scrollable region — track, draggable thumb, click-the-track scrolling — styled from tokens. `list.LayoutScrollbar` draws this one. |

`theme`, `tokens` and `a11y` moved down into
[spectrum](https://github.com/vibrantgio/spectrum) so the theme runtime sits
beneath the components it themes. They stayed here as deprecated alias
packages for one deprecation window; **v0.2.0 deletes all three**. Import
`spectrum/theme`, `spectrum/tokens` and `spectrum/a11y`.

## Usage

The theme owns the typeface (ADR-003): every component that draws text shapes
with the theme's `Typography.Shaper()`, so there is no shaper to build and
`Props.Shaper` stays nil except as a deliberate per-instance override. The
no-gofont lint in this repository fails `go test` on any
`gioui.org/font/gofont` import, so the old fallback practice no longer merely
looks wrong — it fails the build.

Condensed from `list.go` in
[workbench/todos](https://github.com/vibrantgio/workbench/tree/master/todos) —
the smallest complete Vibrant Gio application — one row's checkbox:

```go
// Row is one todo line: a prism checkbox toggling completion, the todo text,
// and a delete icon. Every event routes through mvu.MessageOp, so the
// reducers are the only state writers.
func Row(typ Type, th rx.Observable[theme.Theme], p Palette, item Todo) layout.Widget {
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
where the modal owns the focus tag by passing its own `Clickable`. Note what
is absent: no `Shaper` prop, because the theme supplies it:

```go
submit := button.Button(th, button.Props{
	Label:     "Rename",
	Clickable: &submitClick,
	OnClick:   func(gtx layout.Context) { /* validate, then mvu.MessageOp */ },
})
```

`th` is the theme observable the window hands to your layer builder;
`spectrum/window` supplies one that tracks the OS appearance live. `Message`
is the MVU path — the component adds `mvu.MessageOp{Message: …}` to the
frame's ops and the runtime delivers it to `Update`. FRP-style applications
use `OnClick` instead, and are handed the frame's `layout.Context` so they can
still emit a message from inside the callback. Prism buttons fill the width
they are given and draw at the theme's `Density.ControlHeight`; the pointer
area extends past the drawn control to the 44 dp floor, so neighbouring
Compact controls' slop overlapping is by design (the topmost input area wins).

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

- **v0.2.0 is a breaking release.** The three alias packages — `prism/tokens`,
  `prism/theme`, `prism/a11y` — are gone; import the `spectrum/…` paths. And
  the static render surface no longer takes `tokens.TypeScale`:
  `button.Render`, `input.Render` and `input.RenderDropdown` take a
  `tokens.TextStyle` and a `tokens.Density`, `button.RenderIcon` takes a
  `tokens.Density` (it draws no text), and `richtext.FromTokens` takes a
  `tokens.TextStyle`. Old call: `…, tokens.Radius, tokens.DefaultTypeScale,
  state)`. New call: `…, tokens.Radius, tokens.DefaultTypography.LabelLarge,
  tokens.Comfortable, state)`.
- **`icon.Registry` ships empty.** Nothing populates it yet. For Material
  icons today, render `golang.org/x/exp/shiny/materialdesign/icons` data
  through `ivg/raster/gio`; `button.Props.Icon` wants a `clip.Path` painter,
  not a widget.

## License

MIT — see [LICENSE](./LICENSE).
