# AGENTS.md — prism

The component library of the VibrantGio design system: buttons, inputs,
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

**Golden images.** Tests in seven packages compare rendered output against
PNGs committed under `testdata/golden/`. When a change legitimately moves
pixels, regenerate them within the same change, look at what came out, and
say so in the commit. From the repository root:

    go test ./button ./input ./internal/golden ./layout ./list ./richtext ./scrollbar -golden.update

Both halves of that line matter. `go test` cannot tell that an unfamiliar
flag is boolean, so a flag placed before the packages swallows them: `go
test -golden.update ./...` tests whatever package the repository root
holds, not `./...`. And `./...` cannot stand in for the list — this module
has test packages that store no goldens, and a test binary rejects a flag
it never declared.
