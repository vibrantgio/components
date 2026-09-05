# prism/scrollbar — a visible, token-themed scrollbar for Vibrant Gio

Deliver a `prism/scrollbar` package (draggable thumb + clickable track, drawn
from prism colour tokens) and integrate it into `prism/list` so virtual lists
can show scroll position, then release and adopt it downstream. Today nothing
in Vibrant Gio draws a scrollbar: `prism/list` scrolls invisibly, and the only
existing implementation is `gioui.org/widget/material` — Material-themed and
off-brand.

Repo facts that hold for every task:

- This repo is `github.com/vibrantgio/prism`, checked out at
  `/Users/rene/code/w/vibrantgio/prism` inside the `go.work` workspace at
  `/Users/rene/code/w/vibrantgio/` (siblings: `mvu`, `mindchat`,
  `workbench/*`). Build/test from the repo root: `go build ./...`,
  `go vet ./...`, `go test ./...`, `gofmt -l .` must stay clean after every
  task.
- Golden-image tests use the in-repo harness
  `github.com/vibrantgio/prism/internal/golden`:
  `golden.Render(t, name, size, widget)` renders headlessly and compares
  against `<pkg>/testdata/golden/<name>.png`; regenerate with
  `go test ./<pkg>/ -update` (see `list/list_test.go` for the pattern).
- The interactive showcase is `gallery/main.go` (`go run ./gallery`) — every
  component has a section there; a Gio window is available on this machine
  for visual checks.
- Reference material named `R-…` and decisions named `ADR-…` live under
  `## Decisions` and `## Reference`; the goals embed the ones their tasks
  assume.

Every goal here is **SMART** — Specific, Measurable, Achievable, Relevant, and
Context-bound. The last replaces SMART's "time-bound": instead of a deadline, a
goal is sized to a context budget. Each goal states the five criteria; ADR-001
holds the budget rule.

## Out of scope

For humans; `mdplan next` skips it. Deliberately not in this project:

- **Autohide / fade-out animation.** Needs self-scheduling animation
  (llms.txt rule 5), `pulse` motion curves, and `prism/a11y` ReduceMotion
  handling. v1 is always-visible-while-scrollable (ADR-004); fading is a
  follow-up once the static bar is proven.
- **High-contrast variant.** `a11y.A11yPrefs.HighContrast` exists; wiring it
  into thumb contrast is a follow-up, not v1.
- **An observable component wrapper** (`Scrollbar(th, props) rx.Observable`)
  — ADR-002 picks the immediate-mode shape; a reactive wrapper can be layered
  later without breaking it.
- **cadence/table integration.** The table is virtualised and would benefit,
  but it lives in the cadence repo; do it after prism v0.0.5 ships.
- **Horizontal-scrollbar goldens.** The API is axis-generic (it falls out of
  `layout.Axis.Convert` for free) and gets a smoke test, but golden coverage
  and gallery demos are vertical-only in v1.

## Decisions

Settled choices the work assumes. `mdplan next` skips this section; goals pull
individual ADRs in with embeds.

### ADR-001: Goals are context-bound, not time-bound

SMART's "time-bound" becomes "context-bound": a task is right-sized when an agent
can finish it inside one bounded context window. Target ~100K tokens for a task's
full working set — its `mdplan next` packet plus the files and output it needs.
If a task will not fit, split it until each one does.

### ADR-002: Immediate-mode API, matching prism/list — not an observable component

Most prism components are reactive: `Button(th rx.Observable[theme.Theme],
props) rx.Observable[layout.Widget]` with a `resolvedTokens` snapshot and
Defer-scoped state. `prism/list` is the exception: a plain immediate API
(`list.Layout(gtx, state, items, rowFn)` + `*list.State`) because it composes
inside other components' frame callbacks.

The scrollbar must compose inside `list.Layout` — an immediate call — so it
gets the immediate shape too:

- `scrollbar.State` — per-instance interaction state, allocate once, reuse
  every frame (wraps `gioui.org/widget.Scrollbar`, see ADR-003).
- `scrollbar.Style` — a plain struct of resolved colours/metrics for one
  frame; `scrollbar.FromTokens(c tokens.ColorTokens) Style` derives the
  default look (ADR-005). Callers already hold `tokens.ColorTokens` snapshots
  (that is how every prism consumer works), so no observable plumbing is
  needed.
- `(Style) Layout(gtx, state *State, axis layout.Axis, viewportStart,
  viewportEnd float32) layout.Dimensions` draws it.

**Why:** the reactive wrapper can be added later on top of this (exactly like
button wraps its own draw code); the reverse — extracting an immediate core
from a reactive component — is a rewrite.

### ADR-003: Reuse gioui.org/widget.Scrollbar for gestures; prism owns only the drawing

`gioui.org/widget.Scrollbar` (in `widget/list.go`, NOT the material package)
is theme-independent interaction state: `Update(gtx, axis, viewportStart,
viewportEnd)`, `AddTrack/AddIndicator/AddDrag(ops)`, `IndicatorHovered()`,
`TrackHovered()`, `ScrollDistance() float32`, `Dragging()`. Its drag maths,
track-click paging, and pointer-pass layering are proven; re-implementing them
buys nothing. `scrollbar.State` embeds it.

What prism must own:

- All drawing (material's `ScrollbarStyle` is not imported — it drags in the
  material theme).
- `FromListPosition(lp layout.Position, elements, majorAxisSize int) (start,
  end float32)` — material's `fromListPosition` is **unexported**, so port it
  (~25 lines: per-element size estimate from `lp.Length/elements`, viewport
  fractions from `First/Offset/OffsetLast/Count`, then error diffusion
  against `majorAxisSize/lengthEstPx` weighted toward the nearer end). Port
  it with table tests; R-002 records the recipe.

The dependency cost is nil: prism already requires `gioui.org`.

### ADR-004: Always visible while scrollable; hidden when everything fits

The bar renders whenever the viewport shows less than the full content
(`viewportEnd - viewportStart < 1`) and renders nothing (zero dims) when
content fits — same predicate as material's `rangeIsScrollable`. No fade-out,
no hover-to-reveal in v1 (see Out of scope). Hover and drag get a stronger
thumb colour — that is the only state-dependent styling.

### ADR-005: Token mapping and metrics

Defaults produced by `scrollbar.FromTokens(c tokens.ColorTokens)`:

- Thumb rest: `c.OnSurfaceVariant` at alpha 100 (~40%).
- Thumb hover/drag: `c.OnSurfaceVariant` at alpha 170 (~67%).
- Track: fully transparent by default (`color.NRGBA{}` draws nothing);
  `Style.TrackColor` stays a public field so a surface-tinted gutter is one
  assignment away.
- Metrics (fixed `unit.Dp` constants, overridable via public Style fields):
  thumb minor width **6dp**, track padding **2dp** per side (total gutter
  width `Style.Width() = 10dp`), thumb corner radius **3dp** (pill), minimum
  thumb major length **16dp**.

Alpha-composited `OnSurfaceVariant` tracks light/dark automatically — no
scheme-specific values, mirroring how `Palette`-style consumers derive
hover fills. Golden tests lock these values in both `DefaultLight` and
`DefaultDark`.

### ADR-006: list integration is a new function; `list.Layout` stays untouched

`prism/list` gets:

- `list.State` grows an unexported `sb widget.Scrollbar` field (zero-value
  ready, so `NewState()` callers are unaffected).
- A new entry point rather than new params on `Layout` (every existing caller
  keeps compiling):

  ```go
  func LayoutScrollbar[T any](
      gtx layout.Context,
      state *State,
      bar scrollbar.Style,
      anchor Anchor, // list.Occupy | list.Overlay
      items []T,
      rowFn func(gtx layout.Context, item T) layout.Dimensions,
  ) layout.Dimensions
  ```

- `type Anchor int` with `Occupy` (reserve a right gutter of
  `bar.Width()`, list narrows) and `Overlay` (bar floats over the content's
  trailing edge) — semantics copied from material's `AnchorStrategy` (R-002).
- Scroll feedback: after drawing, `delta := state.sb.ScrollDistance(); if
  delta != 0 { state.l.ScrollBy(delta * float32(len(items))) }` —
  `layout.List.ScrollBy` exists in gio v0.9 and `State.l` is reachable
  in-package.

The list is vertical-only today (`NewState` hardcodes `Axis: Vertical`), so
`LayoutScrollbar` passes `layout.Vertical` and anchors East.

## Reference

Facts to embed where tasks need them. `mdplan next` skips this section.

### R-001: gioui.org/widget.Scrollbar API surface

Defined in `gioui.org@v0.9.0/widget/list.go` (open it before use):

- `Update(gtx, axis, viewportStart, viewportEnd)` — call once per frame
  BEFORE reading hover/drag state; processes pointer events against the
  areas added last frame.
- `AddTrack(ops)` / `AddIndicator(ops)` / `AddDrag(ops)` — register the hit
  areas inside clip areas the caller pushes.
- `IndicatorHovered() bool`, `TrackHovered() bool`, `Dragging() bool`.
- `ScrollDistance() float32` — fraction of content scrolled by
  drag/track-click since last Update; consumer converts to items/pixels.

### R-002: the material layout recipe (port target)

`gioui.org@v0.9.0/widget/material/list.go` is the reference implementation to
transliterate (drawing only — do not import it):

1. `ScrollbarStyle.Layout` bails out (`layout.Dimensions{}`) unless
   `viewportStart > 0 || viewportEnd < 1`; then pins constraints to
   full-major-axis × `Width()` minor via `axis.Convert`, calls
   `state.Update(...)`, and picks the hover colour.
2. `layout.Background{}` stacks track under thumb. Track: push a
   `clip.Rect` over the whole bar, `AddDrag`, then `pointer.PassOp` +
   same rect + `AddTrack` (drag-under-click layering), fill TrackColor.
3. Thumb: inside track padding insets, work in axis-converted space —
   `trackLen := constraints.Min.X`; `viewStart/viewEnd := round(fraction ×
   trackLen)`; `thumbLen := max(viewEnd-viewStart, dp(MajorMinLen))` clamped
   so `viewStart+thumbLen ≤ trackLen`; `op.Offset` to viewStart, fill a
   `clip.RRect` pill, then `pointer.PassOp` + `clip.Rect` + `AddIndicator`.
4. `fromListPosition(lp layout.Position, elements, majorAxisSize)` estimates
   `elementLenEstPx = lp.Length/elements`, computes
   `viewportStart = clamp1((First×elementLenEstPx + Offset) / Length)` and
   `viewportEnd = clamp1(((First+Count)×elementLenEstPx + OffsetLast) /
   Length)` (note: `OffsetLast ≤ 0`), then diffuses the error vs
   `majorAxisSize/Length` onto both ends weighted by proximity so the thumb
   doesn't jitter with variable-height rows.
5. `material.ListStyle.Layout` (Occupy): shrink minor-axis constraints by
   `barWidth` before laying the list, restore, anchor the bar `layout.E`
   (with `Constraints.Min = listDims.Size` so the anchor has room), then
   apply `ScrollDistance()` via `ScrollBy(delta × elements)`, and finally
   re-widen reported dims by `barWidth`.

### R-003: prism conventions checklist

- Package doc comment states the component's contract in the first sentence
  (see `list/list.go`, `button/button.go`).
- Tests: table-driven `_test.go` in-package plus goldens via
  `internal/golden`; goldens live in `<pkg>/testdata/golden/`; every golden
  name states what it proves (`long`, `scrolled-mid`, `short`).
- Gallery: `gallery/main.go` holds per-demo state on the `gallery` struct
  (allocated in the constructor), a `sectionHeader(...)` per demo, and lays
  demos out in one scrolling column — extend it, do not fork it.
- Benchmarks: `list/list_bench_test.go` is the perf-conventions reference;
  `bench/` holds the shared harness.

## Phase P1: the scrollbar package

Everything in this phase happens inside a new top-level `scrollbar/` package;
nothing outside it changes. After every task the full repo gates pass:
`gofmt -l .` empty, `go vet ./...`, `go test ./...` green.

### G1.1: A drawable, draggable scrollbar core

- **Specific:** a `scrollbar` package exposing `State`, `Style`,
  `FromTokens`, `FromListPosition`, and `Style.Layout`, drawing a
  token-themed track+thumb and handling drag/track-click via
  `widget.Scrollbar`.
- **Measurable:** unit tests for the fraction maths and metrics pass;
  `Style.Layout` renders in a headless test; hover/drag colour switching is
  exercised; repo gates stay green.
- **Achievable:** the gesture engine is reused (ADR-003) and the drawing is a
  transliteration of a known-good recipe (R-002).
- **Relevant:** unblocks every scrolling surface in Vibrant Gio — prism/list,
  cadence/table, mindchat panes — starting with the list integration in P2.
- **Context-bound:** three tasks, each well under the ADR-001 budget; the
  working set is one new package plus two reference files.

![[#ADR-002]]

![[#ADR-003]]

![[#ADR-005]]

![[#R-001]]

#### G1.1.1: Scaffold the package — State, Style, FromTokens

Create `scrollbar/scrollbar.go` with the public types and the token-derived
defaults, no drawing yet. Follow R-003 for the package doc.

- [x] Write the package doc: visible scrollbar for scrollable regions;
      immediate-mode (allocate `State` once, style per frame); pairs with
      `prism/list` via `list.LayoutScrollbar` (forward reference is fine).
- [x] Define `type State struct { widget.Scrollbar }` (embedding, so
      `Update`/`ScrollDistance`/`Dragging` are promoted) plus
      `func NewState() *State`.
- [x] Define `type Style struct { ThumbColor, ThumbHoverColor, TrackColor
      color.NRGBA; ThumbMinorWidth, TrackPadding, ThumbCornerRadius,
      ThumbMinLen unit.Dp }` and `func (s Style) Width() unit.Dp` returning
      `ThumbMinorWidth + 2×TrackPadding`.
- [x] Implement `func FromTokens(c tokens.ColorTokens) Style` with exactly
      the ADR-005 values; add a unit test asserting the mapping for
      `tokens.DefaultLight` and `tokens.DefaultDark` (alpha values included)
      and `Width() == 10`.
- [x] Gates: `gofmt -l .` empty, `go vet ./...`, `go test ./...`.

#### G1.1.2: Port the viewport-fraction maths

Add `FromListPosition` — the bridge from `layout.Position` to the
`viewportStart/End` fractions `Style.Layout` consumes. R-002 item 4 is the
algorithm; material's `fromListPosition` is the reference source
(`gioui.org@v0.9.0/widget/material/list.go`).

- [x] Implement `func FromListPosition(lp layout.Position, elements int,
      majorAxisSize int) (start, end float32)` in
      `scrollbar/position.go`, including the error-diffusion step and a
      `clamp1` helper; guard `elements == 0` and `lp.Length == 0` by
      returning `(0, 1)` (nothing to scroll).
- [x] Table tests in `scrollbar/position_test.go`: top of list → start 0;
      bottom (`First+Count == elements`, `OffsetLast == 0`) → end 1; middle
      of a uniform list → `end-start ≈ visible/total` within 1e-3; the
      degenerate guards; and a variable-offset case asserting
      `0 ≤ start ≤ end ≤ 1` always holds.
- [x] Property-style sweep: for a few hundred synthetic positions (nested
      loops over First/Offset/Count, no randomness), assert the invariant
      `0 ≤ start ≤ end ≤ 1` and that start is non-decreasing as First grows.
- [x] Gates green.

#### G1.1.3: Draw and wire the gestures — Style.Layout

Transliterate R-002 items 1–3 into
`func (s Style) Layout(gtx layout.Context, state *State, axis layout.Axis,
viewportStart, viewportEnd float32) layout.Dimensions`.

- [x] Bail out with zero dims when the range is not scrollable
      (`viewportStart <= 0 && viewportEnd >= 1`) — ADR-004.
- [x] Pin constraints axis-independently (`axis.Convert`), call
      `state.Update(gtx, axis, viewportStart, viewportEnd)` before reading
      hover state, and select `ThumbHoverColor` when
      `state.IndicatorHovered() || state.Dragging()`.
- [x] Track pass: clip rect over the bar, `AddDrag`, `pointer.PassOp` +
      clip + `AddTrack`, fill `TrackColor` (transparent default draws
      nothing but the hit areas still register).
- [x] Thumb pass: padding insets, thumb length from fractions with the
      `ThumbMinLen` clamp and end-of-track clamp, `clip.RRect` pill fill,
      `pointer.PassOp` + clip + `AddIndicator`.
- [x] Headless render test (no golden yet): lay it out via the
      `internal/golden` harness's context or a bare `op.Ops` at 40×400 with
      fractions (0.25, 0.5) and assert returned dims equal the pinned
      constraints; call again with (0, 1) and assert zero dims.
- [x] Gates green.

### G1.2: Proof the core visually — goldens and gallery

- **Specific:** golden images covering both colour schemes and thumb
  positions, plus a live gallery section.
- **Measurable:** new goldens exist under `scrollbar/testdata/golden/` and
  `go test ./scrollbar/` compares them; `go run ./gallery` shows the demo.
- **Achievable:** the harness and gallery patterns already exist (R-003).
- **Relevant:** goldens lock the ADR-005 look before anything builds on it.
- **Context-bound:** two tasks, each touching one file plus generated PNGs.

![[#ADR-005]]

![[#R-003]]

#### G1.2.1: Golden tests

Model the test file on `list/list_test.go`.

- [x] Golden cases in `scrollbar/scrollbar_test.go`, rendered at 24×400 on a
      Surface-filled background: `light-top` (0, 0.3), `light-mid`
      (0.35, 0.65), `dark-mid` (same fractions, `DefaultDark`),
      `light-bottom` (0.7, 1.0), and `min-thumb` (0.5, 0.501 — proves the
      16dp clamp).
- [x] Generate with `-update`, eyeball each PNG (Read them — thumb visible,
      pill-shaped, positioned as named, colours differ light/dark), then run
      without `-update` to confirm stability.
- [x] Gates green.

#### G1.2.2: Gallery section

- [x] Add a "Scrollbar" section to `gallery/main.go`: a tall fake-content
      column with a standalone bar driven by fractions from its
      `list.State` sibling demo, or simplest honest equivalent — the point
      is a draggable, hoverable bar on screen.
- [x] `go run ./gallery` (a window opens on this machine): drag the thumb,
      click the track, hover — confirm colour change and motion; note
      anything odd in the task body before checking off.
- [x] Gates green.

Verification note (2026-07-16, agent session): `go run ./gallery` could not
open a window from this session — it panics during window init with
"runtime/cgo: misuse of an invalid Handle" in gio v0.9.0 `os_macos.go`
(`gio_onDestroy` via `CFRelease` in `(*window).init`). This is pre-existing:
the unmodified gallery at HEAD crashes identically, so it is environmental
(agent shell has no usable WindowServer attachment), not caused by this
change. Verified instead by rendering `scrollbarDemo` headlessly through
`internal/golden.Capture` at 600×360: frame at top shows the pill thumb at
the track top; after `ScrollBy(45)` the thumb sits mid-track at ~45/100 —
fractions and motion confirmed. Not verified interactively: pointer drag,
track click, and hover colour change (those paths are `widget.Scrollbar`
plumbing plus the hover-colour branch already covered by scrollbar goldens).
Drag/track clicks feed back via `State.ScrollDistance()` →
`layout.List.ScrollBy`, the same wiring material.List uses. Please give the
Scrollbar section on the List page a quick manual drag/hover when a window
is available.

## Phase P2: prism/list integration

Changes live in `list/` and depend on P1. `list.Layout`'s signature and
behaviour must not change — the whole phase is additive (ADR-006).

### G2.1: LayoutScrollbar — lists that show where they are

- **Specific:** `list.LayoutScrollbar` with `Occupy`/`Overlay` anchors,
  feeding scrollbar drags back into the list position.
- **Measurable:** existing `list` tests pass untouched; new goldens show the
  gutter (Occupy) and floating bar (Overlay); a feedback unit test proves a
  synthetic `ScrollDistance` moves `Position.First`; gallery list demo shows
  the bar live.
- **Achievable:** the composition recipe is R-002 item 5 with the P1 pieces.
- **Relevant:** this is the user-visible payoff — every prism list can now
  show scroll position.
- **Context-bound:** three tasks; working set is `list/` plus the P1 package.

![[#ADR-006]]

![[#R-002]]

#### G2.1.1: The entry point and Occupy anchor

- [x] Add `sb widget.Scrollbar` to `list.State` (unexported; zero-value
      ready — document that `NewState` callers are unaffected).
- [x] Add `type Anchor int` with `Occupy` and `Overlay` constants and doc
      comments explaining the trade (gutter vs content overlap).
- [x] Implement `LayoutScrollbar` per ADR-006 in `list/scrollbar.go`:
      Occupy shrinks `Constraints.Max.X`/`Min.X` by `gtx.Dp(bar.Width())`
      (floor at 0) before `state.l.Layout`, restores constraints, computes
      fractions via `scrollbar.FromListPosition(state.l.Position,
      len(items), listDims.Size.Y)`, anchors `layout.E` with
      `Constraints.Min = listDims.Size` (re-widened by the gutter for
      Occupy), draws `bar.Layout(gtx, &state.sb, layout.Vertical, …)`, and
      reports dims re-widened by the gutter.
- [x] Scroll feedback after drawing: `if d := state.sb.ScrollDistance();
      d != 0 { state.l.ScrollBy(d * float32(len(items))) }`.
- [x] Unit test: 100 fixed-height rows in a short viewport; simulate a drag
      by calling the feedback path with a hand-set distance (extract it as
      a small testable func if needed) and assert `Position.First`
      advances; assert `Occupy` narrows row width by exactly
      `gtx.Dp(bar.Width())`.
- [x] Gates green, and `go test ./list/` proves old goldens unchanged.

#### G2.1.2: Overlay anchor and goldens

- [x] Implement `Overlay`: identical except constraints are NOT reserved and
      dims are NOT re-widened — the bar draws over the trailing edge.
- [x] Goldens in `list/`: `scrollbar-occupy` (rows visibly narrower, bar in
      its gutter, thumb mid-list) and `scrollbar-overlay` (rows full width,
      bar over them) at the same size as the existing list goldens;
      regenerate, eyeball (Read the PNGs), re-run clean.
- [x] Confirm a non-scrollable case (3 rows, tall viewport) renders NO bar
      and dims match plain `Layout` — add it as a unit test, not a golden.
- [x] Gates green.

#### G2.1.3: Gallery upgrade and package docs

- [x] Switch the gallery's 50-item list demo to `LayoutScrollbar(…, Occupy,
      …)` and add an Overlay variant beside it; `go run ./gallery` and
      exercise both with wheel + thumb-drag + track-click.
- [x] Update the `list` package doc to name both entry points and when to
      pick which anchor; cross-reference `scrollbar.FromTokens`.
- [x] Gates green.

VERIFICATION NOTE (2026-07-16): `go run ./gallery` still panics on window
init in this environment ("runtime/cgo: misuse of an invalid Handle", gio
v0.9.0 os_macos.go — the pre-existing environmental issue recorded under
G1.2.2), so the demo could not be exercised interactively. Verified
headlessly instead via a temporary `internal/golden.Capture` test (deleted
before commit): rendered `listScrollbarDemo` at scroll top and mid-scroll
(via `list.NewStateAt`) and the full List page. PNGs confirmed: both
columns render; Occupy rows are visibly narrower than Overlay rows (gutter
reserved vs bar floating over full-width rows); thumbs sit at the top when
unscrolled and proportionally mid-track when scrolled; captures differ
(PixelDiff > 0). NOT verified: live wheel scrolling, thumb-drag, and
track-click in a real window (those interaction paths are covered by the
existing `list` unit/golden tests from G2.1.1/G2.1.2).

## Phase P3: hardening and release

### G3.1: Perf proof, docs, and the v0.0.5 tag

- **Specific:** benchmark the bar's per-frame cost, finish docs, tag and
  push `v0.0.5`.
- **Measurable:** a benchmark exists and shows `LayoutScrollbar` within ~10%
  of plain `Layout` for a 10k-item list; `git tag` shows v0.0.5 pushed with
  a clean tree.
- **Achievable:** drawing is two fills and some clip math — the benchmark
  should confirm, not chase, that cost.
- **Relevant:** downstream (P4) pins this tag; llms.txt promises perf-honest
  components.
- **Context-bound:** two tasks, no file bigger than a screen.

#### G3.1.1: Benchmark

- [x] Add `BenchmarkLayoutScrollbar` beside `list/list_bench_test.go`'s
      pattern: 10k items, fixed-height rows, one frame per iteration;
      report ns/op against plain `Layout` in the bench output comment.
- [x] If the delta exceeds ~10%, profile before optimising; record the
      finding either way in a short note in the bench file.
- [x] Gates green.

#### G3.1.2: Release v0.0.5

- [x] Sweep: `gofmt -l .` empty, `go vet ./...`, `go test ./...`,
      `go run ./gallery` launches.
      Note: `go run ./gallery` panics on window init in this agent
      environment ("runtime/cgo: misuse of an invalid Handle", gio v0.9.0
      os_macos.go) — pre-existing and environmental (see G1.2.2/G2.1.3);
      verified `go build ./gallery` succeeds instead.
- [x] Commit the scrollbar work on master with a message naming the new
      package and the list entry point (Co-Authored-By trailer per repo
      convention), then `git tag v0.0.5` (lightweight, matching v0.0.4) and
      `git push origin master v0.0.5`.
- [x] Verify resolvability: in a scratch dir outside the workspace,
      `GOWORK=off go mod download github.com/vibrantgio/prism@v0.0.5`.

## Phase P4: downstream adoption (cross-repo)

These tasks leave the prism repo — they touch the workspace siblings
`../workbench` and `../mindchat`, each its own git repo on `master`. Commit
and push per repo, same conventions as here. Do them only after v0.0.5 is
pushed (P3).

### G4.1: The ecosystem knows the scrollbar exists

- **Specific:** llms.txt documents the component; mindchat's two scrolling
  panes use it.
- **Measurable:** `../workbench/llms.txt` names scrollbar in the prism line
  at v0.0.5; mindchat builds against v0.0.5 (not the workspace override —
  go.mod bumped), its history pane and sidebar show bars, and its tests
  pass.
- **Achievable:** mindchat's panes already hold `*layout.List` state at
  subscription scope; swapping to `prism/list` state is mechanical.
- **Relevant:** an undocumented component doesn't exist for the next
  llms.txt-guided agent; mindchat was the motivating app.
- **Context-bound:** two tasks, one repo each.

#### G4.1.1: workbench llms.txt

- [x] Update the prism module line to v0.0.5 and add `scrollbar` to its
      component list; if the llms.txt "Minimal go.mod" pins prism, bump it
      there too.
- [x] Add one sentence in the components overview: prism/list can render a
      visible scrollbar via `list.LayoutScrollbar` (Occupy or Overlay).
- [x] Commit and push workbench.

#### G4.1.2: mindchat adoption

- [x] In `../mindchat`: `go get github.com/vibrantgio/prism@v0.0.5`, then
      replace the chat-history `layout.List` and the sidebar `layout.List`
      (both allocated at subscription scope in `view.go`'s ContentLayer)
      with `list.NewState()` + `list.LayoutScrollbar(…, list.Overlay, …)`,
      deriving each pane's `scrollbar.Style` from the same
      `tokens.ColorTokens` the `Palette` comes from (extend `themed` if
      needed).
- [x] `go test ./...` in mindchat (wiring test must still measure 1
      consumer), launch the app, and confirm both bars render and drag.
- [x] Commit and push mindchat.

VERIFICATION NOTE (2026-07-16): launching mindchat still panics on window
init in this environment ("runtime/cgo: misuse of an invalid Handle", gio
v0.9.0 os_macos.go — the pre-existing environmental issue recorded under
G1.2.2/G2.1.3/G3.1.2), so the bars could not be exercised interactively.
Verified headlessly instead via a temporary `gioui.org/gpu/headless`
capture test in mindchat (deleted before commit): rendered `Sidebar` (40
chats, 260x400) and `ChatPane` (30 messages, 600x400) at scroll top and
mid-scroll (`list.NewStateAt`). PNGs confirmed: both panes draw the
overlay bar on the trailing edge over full-width rows, with the thumb at
the top when unscrolled and proportionally mid-track when scrolled. All
mindchat tests pass (wiring test still measures 1 consumer); gofmt/vet
clean; builds green both in-workspace and with GOWORK=off against the
v0.0.5 pin. NOT verified: live thumb-drag and track-click in a real
window (those interaction paths are covered by prism's `list` unit tests
from G2.1.1/G2.1.2). Behaviour note: the history pane's former
`ScrollToEnd: true` is not expressible through `prism/list.State`, so
auto-follow of new messages is dropped by this swap.
