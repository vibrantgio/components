// The gallery is a nested module so that prism itself does not depend on
// pulse. It is the only thing in prism that imports pulse/springbutton, and
// that single import put pulse in prism's go.mod and closed a module cycle
// (prism -> pulse -> prism). A demo may depend on layers above its parent;
// the parent must not inherit that edge.
//
// The prism requirement must stay at v0.1.0 or later. Every earlier prism
// carries gallery/ inside the prism module itself, so resolving this module
// against one of those reports an ambiguous import for this module's own
// path — found in both prism and here. v0.1.0 is the first prism that
// excludes gallery/, which is what makes this module resolvable at all.
module github.com/vibrantgio/prism/gallery

go 1.25.1

require (
	gioui.org v0.10.1
	github.com/reactivego/rx v0.3.0
	github.com/vibrantgio/ivg/raster/gio v0.1.6
	github.com/vibrantgio/prism v0.1.0
	github.com/vibrantgio/pulse v0.0.4
)

require (
	gioui.org/shader v1.0.8 // indirect
	github.com/go-text/typesetting v0.3.4 // indirect
	github.com/reactivego/scheduler v0.2.0 // indirect
	github.com/vibrantgio/ivg v0.1.4 // indirect
	github.com/vibrantgio/mvu v0.4.1 // indirect
	github.com/vibrantgio/svg v0.0.6 // indirect
	github.com/vibrantgio/traer v0.0.7 // indirect
	golang.org/x/exp/shiny v0.0.0-20260727155853-b88d891fe743 // indirect
	golang.org/x/image v0.44.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)
