# AGENTS.md — prism

The component library of the Vibrant Gio design system: buttons, inputs,
lists, rich text, scrollbars, icons, layout primitives, and the `theme` and
`tokens` contract the rest of the stack styles against.

**Layer.** Tier 2 of ADR-001's stack, `mvu → spectrum → prism → pulse →
cadence → markdown`. It imports mvu and the support libraries ivg and svg;
pulse, cadence and markdown import it. Spectrum imports it too today — the
inversion that goal G-B3 corrects by moving the token and theme contract
down into spectrum.

**Read the canonical guide before you write code against this module.** It is
the organization's one agent guide — the module inventory with current tags,
the application skeleton, the MVU loop and rx semantics, typography, and the
pitfalls that are not guessable. It lives exactly once, in `vibrantgio/.github`,
and this file links it rather than copying it:

    https://raw.githubusercontent.com/vibrantgio/.github/master/llms.txt

**Modules.** `github.com/vibrantgio/prism` at the repository root, and one
nested module: `gallery/` (`github.com/vibrantgio/prism/gallery`).
Nested-module tags carry the directory as a prefix — `gallery/v0.1.2`, not
`v0.1.2`.

**Build and test.** From the repository root, and again inside each nested
module directory — `./...` does not cross a module boundary:

    go build ./... && go test ./...

**Golden images.** Tests in eight packages compare rendered output against
PNGs committed under `testdata/golden/`. `prism/golden` is the harness they
use, and since F5.5 it is the organization's only one: pulse, cadence,
markdown and the workbench applications import it too, so a change to it
moves every stored image in the org, not just prism's. Regenerate all of
them, not only this repo's, before believing a change here is pixel-neutral.
When a change legitimately moves pixels, regenerate them within the same
change, look at what came out, and say so in the commit. From the repository
root:

    go test ./button ./golden ./icon ./input ./layout ./list ./richtext ./scrollbar -golden.update

Both halves of that line matter. `go test` cannot tell that an unfamiliar
flag is boolean, so a flag placed before the packages swallows them: `go
test -golden.update ./...` tests whatever package the repository root
holds, not `./...`. And `./...` cannot stand in for the list — this module
has test packages that store no goldens, and a test binary rejects a flag
it never declared.

**A green CI run does not say these images matched.** The harness answers a
failed `headless.NewWindow` with `t.Skipf`, and a skipped test passes, so
the pixels and the build status are independent facts. Until F5.4 nothing
could tell them apart: the workflow ran plain `go test`, which never prints
a skip, and downloading a run log needs admin rights on the repository. The
test step is verbose now, and the step after it publishes the verdict as a
workflow annotation — and annotations, unlike logs, are public: the
`check-runs` endpoint for the commit returns them with no token at all.
Read it before treating green as a golden-image gate, and expect the answer
to be that they skipped. The runner installs the GL *development* headers,
where gio's own Linux CI also installs the drivers — `libegl-mesa0`,
`libglx-mesa0`, `libgl1-mesa-dri`, `mesa-libgallium`, `libgbm1`, `libegl1`,
`mesa-vulkan-drivers` — without which there is no EGL display to initialise
and no Vulkan ICD for gio's fallback context to find.

**A golden test pins its faces; application code does not.** Every golden and
pixel test here builds its shaper with
`tokens.DefaultTypography.DeterministicShaper()` — the default typography's
faces and nothing else, system fonts off, so the stored PNGs are the same on
every machine. Applications call `Shaper()` instead, which falls back to the
platform's own fonts so that text outside Roboto and Roboto Mono still
resolves. The two are not interchangeable: a golden written against
`Shaper()` passes on the machine that wrote it and fails on one with a
different font set, which is the failure the split constructor exists to make
impossible. When a test genuinely needs a glyph the default faces lack, widen
the collection rather than reach for the system —
`tokens.DefaultTypography.WithFaces(notosansmono.FontFace()).DeterministicShaper()`
— and assert the shaper resolved the rune rather than storing it as pixels.
