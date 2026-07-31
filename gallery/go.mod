// The gallery is a nested module so that prism itself does not depend on
// pulse. It is the only thing in prism that imports pulse/springbutton, and
// that single import put pulse in prism's go.mod and closed a module cycle
// (prism -> pulse -> prism). A demo may depend on layers above its parent;
// the parent must not inherit that edge.
//
// The prism requirement below still names v0.0.9, which is the last prism
// that carries gallery/ inside the prism module — so resolving this module
// standalone reports an ambiguous import for its own path. That resolves the
// moment prism v0.1.0 is published, since v0.1.0 is the first prism that
// excludes gallery/. Bump this to v0.1.0 and re-tidy after that push.
module github.com/vibrantgio/prism/gallery

go 1.25.1

require (
	gioui.org v0.10.1
	github.com/reactivego/rx v0.3.0
	github.com/vibrantgio/ivg/raster/gio v0.1.5
	github.com/vibrantgio/prism v0.0.9
	github.com/vibrantgio/pulse v0.0.3
)
