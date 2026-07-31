package icon

import "github.com/vibrantgio/svg"

// Kind identifies the icon format.
type Kind uint8

const (
	KindSVG Kind = iota
	KindIVG
)

// Icon holds an icon in either SVG or IVG format.
type Icon struct {
	kind    Kind
	svgIcon *svg.Icon
	ivgData []byte
}

// FromSVG wraps a parsed SVG icon.
func FromSVG(i *svg.Icon) Icon {
	return Icon{kind: KindSVG, svgIcon: i}
}

// FromIVG wraps raw IVG bytes.
func FromIVG(data []byte) Icon {
	return Icon{kind: KindIVG, ivgData: data}
}

// Kind returns the icon's format.
func (i Icon) Kind() Kind { return i.kind }

// SVG returns the parsed SVG icon. Only valid when Kind() == KindSVG.
func (i Icon) SVG() *svg.Icon { return i.svgIcon }

// IVG returns the raw IVG bytes. Only valid when Kind() == KindIVG.
func (i Icon) IVG() []byte { return i.ivgData }

// Registry maps names to icons.
type Registry struct {
	m map[string]Icon
}

// New creates an empty Registry.
func New() *Registry {
	return &Registry{m: make(map[string]Icon)}
}

// Register adds an icon under the given name.
func (r *Registry) Register(name string, icon Icon) {
	r.m[name] = icon
}

// Icon resolves an icon by name.
func (r *Registry) Icon(name string) (Icon, bool) {
	icon, ok := r.m[name]
	return icon, ok
}
