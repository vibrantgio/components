package icons

import (
	"embed"
	"fmt"
	"image/color"
	"runtime"
	"slices"
	"strings"
	"sync"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/vibrantgio/components/icon"
	"github.com/vibrantgio/svg"
	"github.com/vibrantgio/svg/parser"
)

// marksFS holds the set's source. Every mark ships in every binary; which one
// a name resolves to is decided at run time, not by build tag.
//
//go:embed marks/*.svg
var marksFS embed.FS

// markDir is the directory inside marksFS the marks live in.
const markDir = "marks"

// platformSep joins a mark's name to the operating system it is drawn for,
// both in a registry key and in Resolve's answer. A name never contains it —
// names are lowercase words joined by "-" — so the two halves stay separable.
const platformSep = "@"

// Name identifies a mark by the control it belongs to.
type Name string

// The marks the set carries. Each names a control, not a picture: what the
// drawing behind one looks like is free to change and does change between
// platforms, and these are the strings call sites store.
const (
	// Sidebar is the control that shows and hides a window's sidebar. One
	// mark serves both directions — which way the control is about to go is
	// the control's to signal, and a mark that morphs cannot be recognised.
	Sidebar Name = "sidebar"

	// Disclosure is the control that opens and closes a row's children. The
	// mark is drawn as the row stands closed; a control showing an open row
	// turns it a quarter turn rather than swapping in a second drawing.
	Disclosure Name = "disclosure"

	// HistoryBack is the control that returns to what was shown before.
	HistoryBack Name = "history-back"

	// HistoryForward is the control that goes back to what the reader had
	// stepped away from.
	HistoryForward Name = "history-forward"
)

// Painter draws a mark into a square of sizePx at the current origin, in col.
// It is the shape this library's controls take for an icon slot.
type Painter func(gtx layout.Context, sizePx int, col color.NRGBA)

// Set is the icon set as one operating system sees it: every mark is present,
// and a name resolves to that system's drawing where the set carries one and
// to the drawing that serves every platform where it does not.
//
// A Set caches the ops it builds, so keep the one New returns rather than
// making a fresh one per frame.
type Set struct {
	goos  string
	reg   *icon.Registry
	names []Name

	mu    sync.Mutex
	ops   *op.Ops
	cache map[cacheKey]op.CallOp
}

// cacheKey is what a built drawing is stored under: the registry key that
// answered, and the pixel size it was built for. Colour is not part of it —
// the painter applies colour outside the recorded ops.
type cacheKey struct {
	entry string
	px    int
}

// New returns the set as the named operating system sees it. goos is a
// runtime.GOOS value; any string is accepted, and a system no mark is drawn
// for gets the fallback drawing throughout.
//
// Passing a platform explicitly is the seam this package resolves through:
// there is no build-tag fan-out, so a test or a preview can ask for another
// system's answer on this one.
func New(goos string) *Set {
	s := &Set{
		goos:  goos,
		reg:   icon.New(),
		ops:   new(op.Ops),
		cache: make(map[cacheKey]op.CallOp),
	}
	marks := mustLoadMarks()
	for _, key := range marks.keys() {
		s.reg.Register(key, icon.FromSVG(marks[key]))
		if !strings.Contains(key, platformSep) {
			s.names = append(s.names, Name(key))
		}
	}
	return s
}

// Register adds every mark in the set to r, under the key its file name
// gives it: the plain name for the drawing that serves every platform, and
// name@goos for a platform's own drawing. The icons registered are freshly
// parsed, so a registry an application owns shares nothing with this
// package's own state.
func Register(r *icon.Registry) {
	for key, parsed := range mustLoadMarks() {
		r.Register(key, icon.FromSVG(parsed))
	}
}

// Resolve reports the registry key that answers name on this set's platform:
// name@goos when the set carries that platform's own drawing, name when it
// falls back to the drawing every platform shares. ok is false for a name the
// set does not carry at all.
func (s *Set) Resolve(name Name) (key string, ok bool) {
	if k := string(name) + platformSep + s.goos; s.carries(k) {
		return k, true
	}
	if k := string(name); s.carries(k) {
		return k, true
	}
	return "", false
}

// Has reports whether the set carries a mark under name.
func (s *Set) Has(name Name) bool {
	_, ok := s.Resolve(name)
	return ok
}

// Names lists every mark in the set, in order. Platform variants are not
// separate entries: a name appears once however many drawings stand behind it.
func (s *Set) Names() []Name {
	return slices.Clone(s.names)
}

// GOOS reports the operating system this set resolves for.
func (s *Set) GOOS() string { return s.goos }

// Mark returns a painter for name, or nil when the set carries no such mark.
// A painter is cheap: it looks the mark up once and closes over the set's
// shared cache, so building one per frame costs nothing worth avoiding.
func (s *Set) Mark(name Name) Painter {
	entry, ok := s.Resolve(name)
	if !ok {
		return nil
	}
	return func(gtx layout.Context, sizePx int, col color.NRGBA) {
		if sizePx <= 0 {
			return
		}
		call := s.drawing(entry, sizePx)
		paint.ColorOp{Color: col}.Add(gtx.Ops)
		call.Add(gtx.Ops)
	}
}

// carries reports whether the set holds a drawing under the registry key.
func (s *Set) carries(key string) bool {
	_, ok := s.reg.Icon(key)
	return ok
}

var (
	hostOnce sync.Once
	hostSet  *Set
)

// host is the set as the running process's operating system sees it, built on
// first use so that importing this package parses nothing.
func host() *Set {
	hostOnce.Do(func() { hostSet = New(runtime.GOOS) })
	return hostSet
}

// Mark returns a painter for name as the host operating system draws it, or
// nil when the set carries no such mark.
func Mark(name Name) Painter { return host().Mark(name) }

// Has reports whether the set carries a mark under name.
func Has(name Name) bool { return host().Has(name) }

// Names lists every mark in the set, in order.
func Names() []Name { return host().Names() }

// Resolve reports the registry key that answers name on the host operating
// system. See Set.Resolve.
func Resolve(name Name) (key string, ok bool) { return host().Resolve(name) }

// markSet is the parsed source keyed by registry key.
type markSet map[string]*svg.Icon

// keys returns the set's registry keys in order.
func (m markSet) keys() []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

// loadMarks parses every embedded mark into a fresh set of icons keyed by
// registry key. Each call parses again: the drawings are mutated while their
// ops are built, so two owners must not share one.
func loadMarks() (markSet, error) {
	entries, err := marksFS.ReadDir(markDir)
	if err != nil {
		return nil, err
	}
	out := make(markSet, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		key, err := registryKey(e.Name())
		if err != nil {
			return nil, err
		}
		f, err := marksFS.Open(markDir + "/" + e.Name())
		if err != nil {
			return nil, err
		}
		parsed, err := parser.NewParser(parser.StrictErrorMode).ParseStream(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out[key] = parsed
	}
	return out, nil
}

// mustLoadMarks is loadMarks over source that ships inside the binary. A
// failure means a mark file in this package is malformed or misnamed, which
// is a broken build rather than a runtime condition — the package's own tests
// parse and name-check every file, so this cannot fire on a tree that passes
// them.
func mustLoadMarks() markSet {
	set, err := loadMarks()
	if err != nil {
		panic("icons: " + err.Error())
	}
	return set
}

// registryKey turns a mark's file name into its registry key: "sidebar.svg"
// into "sidebar", "sidebar.darwin.svg" into "sidebar@darwin".
func registryKey(file string) (string, error) {
	base := strings.TrimSuffix(file, ".svg")
	name, goos, hasVariant := strings.Cut(base, ".")
	if err := checkName(name); err != nil {
		return "", fmt.Errorf("%s: %w", file, err)
	}
	if !hasVariant {
		return name, nil
	}
	if goos == "" || strings.Contains(goos, ".") {
		return "", fmt.Errorf("%s: expected <name>.svg or <name>.<goos>.svg", file)
	}
	return name + platformSep + goos, nil
}

// checkName enforces the naming rule: lowercase ASCII words joined by "-".
func checkName(name string) error {
	if name == "" {
		return fmt.Errorf("empty mark name")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
		default:
			return fmt.Errorf("mark name %q: only lowercase letters, digits and %q are allowed", name, "-")
		}
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("mark name %q: %q is a word separator, not an edge", name, "-")
	}
	return nil
}
