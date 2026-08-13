// Package coordination provides the Subject primitive for cross-widget
// coordination in Gio applications.
//
// The package codifies four invariants established by validation
// Experiments C1–C2 (their record predates the organization and is not
// preserved; the invariants below are its surviving summary):
//
//  1. One-frame lag. Subject delivery is asynchronous. Cross-widget state
//     changes are visible on the frame AFTER the emitting frame. This is
//     correct for drag, modal, and tooltip concerns and is imperceptible at
//     ≥30 fps.
//
//  2. Mutable per-widget state must be hoisted outside FRP closures. Gesture
//     accumulators (gesture.Drag, gesture.Hover, widget.Clickable) must live
//     in the owning struct, not inside rx.Map closures that are regenerated on
//     every Subject emission.
//
//  3. Buffer capacity must exceed maximum burst. For pointer-event emitters,
//     BufCapPointer prevents frame-goroutine blocking. For infrequent signals
//     (modals, tooltips) BufCapSignal suffices.
//
//  4. Intermediate emissions are silently dropped under burst. The
//     mvu.Window atomic snapshot retains only the most recent widget closure
//     before the next frame fires. Signals where every value is load-bearing
//     (undo stacks, event logs) require a different mechanism.
//
// # Subscriptions are released on Unsubscribe
//
// A fifth invariant, added by G0B.1, is not an rx property but a property of
// this package: unsubscribing a Subject subscription releases its slot, at
// once and deterministically.
//
// rx.Subject does not do that. Its subscription list reuses an entry only
// once that entry's cursor has been parked, and the only code that parks a
// cursor runs inside the subscription's own scheduled receiver task — which
// Unsubscribe cancels before it can run. So under a bare rx.Subject a slot is
// consumed for the life of the process, and, worse, the departed
// subscription's frozen cursor pins the ring buffer's window: once the
// producer has written bufCap more items it blocks forever, on whatever
// goroutine called it. For a process-global Subject emitted from the Gio
// frame goroutine that is a hung application, not a dropped signal.
//
// Subject therefore does not hand rx.Subject's subscription list to callers.
// It keeps its own registry of live subscriptions and gives each one a
// private single-subscriber rx.Subject to deliver through, so that rx's
// scheduler backoff, trampoline support and one-frame-lag semantics are all
// still rx's, while the lifetime is this package's. Detaching a subscription
// removes it from the registry and closes its private buffer, which also
// releases a producer blocked inside it.
//
// # Deprecated
//
// Deprecated: use [github.com/vibrantgio/mvu/stream.Value] instead.
//
// ADR-008 (G-G0C) retired the reason this package existed. The three concerns
// its doc names above — drag, modal and tooltip — are frame state now, owned
// by the goroutine Gio lays a frame out on and read during layout; toasts are
// messages reduced onto the model. Not one of them wanted a bus, and three of
// the four that existed turned out to have no subscriber at all. What is left
// is a genuine stream, of which the organization has exactly one
// (theme/preferences), and it cannot live here: preferences is tier 1 and
// this is tier 2, so by F5.5's rule the package lived too high.
//
// The replacement is not this code moved down a tier. G0C.5 measured it
// against rx's own [rx.Observable.Behavior] and found this wrapper both
// larger and slower: 292 lines against 14, a 64-subscriber ceiling against
// none, one full arrival cycle at 52.0 µs against 1.3 µs on an M1 Max, and —
// because
// delivery still goes through a private rx.Subject per subscription — a
// producer that a live consumer can still block, which rx.Behavior's
// conflating write cannot. The slot leak this package was written to fix was
// real; the fix was reaching for a different rx primitive, not wrapping the
// wrong one.
//
// # When this goes
//
// It has no users left in the organization at all. G0C.5 emptied the library
// side; G0C.6 moved the last two, the demo mains in prism/gallery and
// patterns/modal/gallery, onto stream.Value — so the precondition for removing
// it is already met and nothing in here is exercised by anything but its own
// tests.
//
// It is still not removed yet, and the reason is a rule rather than a
// hesitation. The org's Release protocol removes a deprecated package in the
// final major bump, alongside ADR-001's and ADR-003's alias shims, so that a
// consumer outside the organization meets every removal at one version
// boundary instead of discovering them one patch at a time. These are public
// modules and the notice you are reading shipped in prism v0.6.1; taking the
// code out in the same release would be a deprecation window zero releases
// wide, which is not a window. So: **this package is removed at components
// v1.0.0**, and nothing but that bump closes it. Until then it stays exactly
// as it is — unchanged and still tested, because a deprecated package that
// quietly rots is worse than one that works.
//
// It is deliberately not a forwarder to stream.Value. A forwarder would
// compile everywhere and change delivery policy, subscriber ceiling and
// buffering under a signature that still type-checks; ADR-008's rule is to
// break loudly rather than deprecate quietly, and its corollary is that if
// you are not breaking, you must not change behaviour either.
package coordination

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reactivego/rx"
)

// BufCapPointer is the recommended producer-side buffer depth for Subjects
// that emit on every pointer event (drag, hover). Sized to ~2×60 fps so the
// frame goroutine is never blocked under burst pointer events.
const BufCapPointer = 128

// BufCapSignal is the recommended producer-side buffer depth for infrequent
// coordination signals (modal depth, focus owner, tooltip arbitration).
const BufCapSignal = 8

// MaxSubscribers is the number of subscriptions a Subject will hold at once.
//
// It is a leak detector, not a backpressure control. Slots are released on
// Unsubscribe, so a well-behaved application never approaches it however many
// shells it opens and closes over its lifetime; reaching it means
// subscriptions are being created and never unsubscribed. The limit exists so
// that shows up as ErrSubscriberLimit — which names the holders — rather than
// as unbounded goroutine and buffer growth.
//
// The value is deliberately well above any plausible live subscriber count
// (rx's own default subscription capacity is 32) and well above the eight this
// package allowed before G0B.1. Each live subscription costs one goroutine and
// one ring of bufCap items, so it is not free, merely cheap.
const MaxSubscribers = 64

// ErrSubscriberLimit is returned to a subscriber that arrives when a Subject
// already holds MaxSubscribers live subscriptions. It is delivered as the
// subscription's error rather than panicking, matching how rx reports
// ErrOutOfSubjectSubscriptions. The wrapped message names the limit and the
// call sites holding the slots.
var ErrSubscriberLimit = errors.Join(rx.Err, errors.New("coordination: Subject subscriber limit reached"))

// Subject creates a typed broadcast channel for cross-widget coordination.
// The Observer side is held by one producer; the Observable side may be
// subscribed by up to MaxSubscribers concurrent consumers, and a slot is
// returned as soon as its subscription is unsubscribed.
//
// bufCap is the producer-side buffer depth, applied per subscription. Use
// BufCapPointer for signals emitted on every pointer event; use BufCapSignal
// for infrequent signals.
//
// Deprecated: use [github.com/vibrantgio/mvu/stream.Value], which lives at
// tier 0 where every layer can reach it, costs no goroutine while nobody is
// watching, has no subscriber ceiling, and cannot be blocked by a consumer
// that stops draining. See this package's doc for the measurements.
func Subject[T any](bufCap int) (rx.Observer[T], rx.Observable[T]) {
	h := &hub[T]{bufCap: bufCap}
	return h.emit, h.subscribe
}

// hub is the subscription registry behind Subject. It owns the list of live
// legs; the producer fans out to a snapshot of that list taken under the
// mutex and released before any delivery, so that a subscriber departing
// while the producer is blocked inside its buffer can always make progress.
type hub[T any] struct {
	bufCap int

	mu     sync.Mutex
	legs   []*leg[T]
	closed bool
	err    error
}

// leg is one live subscription: a private single-subscriber rx.Subject that
// carries values to it, plus the provenance the limit error reports.
type leg[T any] struct {
	send     rx.Observer[T]
	site     string
	since    time.Time
	detached atomic.Bool
}

// emit is the Observer side. It is held by one producer, as rx.Subject's is.
func (h *hub[T]) emit(next T, err error, done bool) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	if done {
		h.closed, h.err = true, err
	}
	legs := make([]*leg[T], len(h.legs))
	copy(legs, h.legs)
	if done {
		h.legs = nil
	}
	h.mu.Unlock()

	for _, l := range legs {
		if l.detached.Load() {
			continue
		}
		l.send(next, err, done)
	}
}

// subscribe is the Observable side.
func (h *hub[T]) subscribe(observe rx.Observer[T], scheduler rx.Scheduler, subscriber rx.Subscriber) {
	h.mu.Lock()
	switch {
	case h.closed:
		err := h.err
		h.mu.Unlock()
		terminate(observe, scheduler, subscriber, err)
		return
	case len(h.legs) >= MaxSubscribers:
		err := fmt.Errorf("%w: %d of %d slots are held by live subscriptions; a slot is released only by Unsubscribe. Held by:\n%s",
			ErrSubscriberLimit, len(h.legs), MaxSubscribers, h.holdersLocked())
		h.mu.Unlock()
		terminate(observe, scheduler, subscriber, err)
		return
	}
	send, stream := rx.Subject[T](0, 0, h.bufCap, 1)
	l := &leg[T]{send: send, site: subscribeSite(), since: time.Now()}
	h.legs = append(h.legs, l)
	h.mu.Unlock()

	// rx owns delivery: the leg's own receiver task runs on the subscriber's
	// scheduler with rx's backoff, and registers its cancellation on
	// subscriber. Detach is registered after it, so by the time we close the
	// leg its receiver is already cancelled and the downstream observer sees
	// no spurious completion.
	stream(observe, scheduler, subscriber)
	subscriber.OnUnsubscribe(func() { h.detach(l) })
}

// detach releases l's slot and closes its buffer. Closing matters as much as
// removing: a producer parked inside a full leg buffer only returns once that
// buffer leaves the active state.
func (h *hub[T]) detach(l *leg[T]) {
	if l.detached.Swap(true) {
		return
	}
	h.mu.Lock()
	for i, x := range h.legs {
		if x == l {
			h.legs = append(h.legs[:i], h.legs[i+1:]...)
			break
		}
	}
	h.mu.Unlock()

	var zero T
	l.send(zero, nil, true)
}

// holdersLocked renders the live subscription sites, most-held first, for the
// limit error. Callers must hold h.mu.
func (h *hub[T]) holdersLocked() string {
	type row struct {
		site   string
		count  int
		oldest time.Duration
	}
	order := make([]string, 0, len(h.legs))
	byRow := make(map[string]*row, len(h.legs))
	now := time.Now()
	for _, l := range h.legs {
		r, ok := byRow[l.site]
		if !ok {
			r = &row{site: l.site}
			byRow[l.site] = r
			order = append(order, l.site)
		}
		r.count++
		if age := now.Sub(l.since); age > r.oldest {
			r.oldest = age
		}
	}
	var b strings.Builder
	for _, site := range order {
		r := byRow[site]
		fmt.Fprintf(&b, "  %d× %s (oldest %s)\n", r.count, r.site, r.oldest.Round(time.Millisecond))
	}
	return strings.TrimRight(b.String(), "\n")
}

// terminate delivers err to a subscriber that never got a slot, on its own
// scheduler, exactly as rx.Subject reports ErrOutOfSubjectSubscriptions.
func terminate[T any](observe rx.Observer[T], scheduler rx.Scheduler, subscriber rx.Subscriber, err error) {
	runner := scheduler.Schedule(func() {
		if subscriber.Subscribed() {
			var zero T
			observe(zero, err, true)
		}
	})
	subscriber.OnUnsubscribe(runner.Cancel)
}

// subscribeSite names the code that took the slot: the nearest call frames
// that are neither rx's plumbing nor this package's. Subscription is rare, so
// walking the stack here costs nothing that matters, and it is the difference
// between an error that names the leak and one that surfaces on a bystander.
func subscribeSite() string {
	var pcs [64]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	parts := make([]string, 0, 3)
	for len(parts) < 3 {
		f, more := frames.Next()
		if f.Function == "" && !more {
			break
		}
		if !strings.Contains(f.Function, "github.com/reactivego/") &&
			!strings.Contains(f.Function, "github.com/vibrantgio/components/coordination.") {
			parts = append(parts, fmt.Sprintf("%s (%s:%d)", shortFunc(f.Function), shortFile(f.File), f.Line))
		}
		if !more {
			break
		}
	}
	if len(parts) == 0 {
		return "unknown call site"
	}
	return strings.Join(parts, " ← ")
}

// shortFunc drops the module path from a fully qualified function name,
// keeping the last package element and the symbol.
func shortFunc(fn string) string {
	if i := strings.LastIndex(fn, "/"); i >= 0 {
		return fn[i+1:]
	}
	return fn
}

// shortFile keeps the last two path elements of a source file.
func shortFile(file string) string {
	if i := strings.LastIndex(file, "/"); i > 0 {
		if j := strings.LastIndex(file[:i], "/"); j >= 0 {
			return file[j+1:]
		}
	}
	return file
}
