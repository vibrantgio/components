package coordination_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/coordination"
)

// subscribeOne subscribes to stream and returns the subscription, a channel of
// values and a channel of errors. Values and errors are non-blocking sends, so
// a caller that stops reading never wedges the receiver.
func subscribeOne[T any](stream rx.Observable[T]) (rx.Subscription, <-chan T, <-chan error) {
	values := make(chan T, 64)
	errs := make(chan error, 1)
	sub := stream.Subscribe(rx.GoroutineContext(), func(next T, err error, done bool) {
		if err != nil {
			select {
			case errs <- err:
			default:
			}
			return
		}
		if !done {
			select {
			case values <- next:
			default:
			}
		}
	})
	return sub, values, errs
}

func awaitValue[T any](t *testing.T, what string, values <-chan T, errs <-chan error) T {
	t.Helper()
	select {
	case v := <-values:
		return v
	case err := <-errs:
		t.Fatalf("%s: subscription failed: %v", what, err)
	case <-time.After(3 * time.Second):
		t.Fatalf("%s: timed out waiting for delivery", what)
	}
	var zero T
	return zero
}

// TestUnsubscribeReleasesTheSlot: a Subject must not leak a subscription slot
// on Unsubscribe. The loop deliberately runs far past MaxSubscribers with only
// ever one live subscription, because that is the shape of a long-running
// application: shells open and close, and at no instant are many subscribed at
// once.
func TestUnsubscribeReleasesTheSlot(t *testing.T) {
	obs, stream := coordination.Subject[int](coordination.BufCapSignal)

	for i := range coordination.MaxSubscribers * 3 {
		sub, values, errs := subscribeOne(stream)
		obs(i, nil, false)
		if got := awaitValue(t, "iteration", values, errs); got != i {
			t.Fatalf("iteration %d: got %d", i, got)
		}
		sub.Unsubscribe()
	}
}

// TestProducerRunsFreeAfterUnsubscribe: a departed rx subscription keeps a
// frozen cursor and the producer's ring window is pinned to the slowest
// cursor, so a producer must not block once bufCap more items are written with
// nothing subscribed. On the Gio frame goroutine that would be a hung window,
// not a dropped signal.
func TestProducerRunsFreeAfterUnsubscribe(t *testing.T) {
	obs, stream := coordination.Subject[int](coordination.BufCapSignal)

	sub, values, errs := subscribeOne(stream)
	obs(-1, nil, false)
	awaitValue(t, "priming delivery", values, errs)
	sub.Unsubscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range coordination.BufCapSignal * 20 {
			obs(i, nil, false)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("producer blocked after its only subscriber unsubscribed")
	}
}

// TestManyConcurrentSubscribers holds the ceiling: many subscriptions are live
// at once and every one of them receives the emission.
func TestManyConcurrentSubscribers(t *testing.T) {
	const n = coordination.MaxSubscribers
	obs, stream := coordination.Subject[int](coordination.BufCapSignal)

	subs := make([]rx.Subscription, 0, n)
	values := make([]<-chan int, 0, n)
	errCh := make([]<-chan error, 0, n)
	for range n {
		sub, v, e := subscribeOne(stream)
		subs = append(subs, sub)
		values = append(values, v)
		errCh = append(errCh, e)
	}
	defer func() {
		for _, s := range subs {
			s.Unsubscribe()
		}
	}()

	obs(7, nil, false)
	for i := range n {
		if got := awaitValue(t, "fan-out", values[i], errCh[i]); got != 7 {
			t.Errorf("subscriber %d: got %d, want 7", i, got)
		}
	}
}

// TestSubscriberLimitNamesItself: overrunning the limit must say what ran out
// and who is holding it, rather than surfacing on an innocent bystander.
func TestSubscriberLimitNamesItself(t *testing.T) {
	_, stream := coordination.Subject[int](coordination.BufCapSignal)

	subs := make([]rx.Subscription, 0, coordination.MaxSubscribers)
	for range coordination.MaxSubscribers {
		sub, _, _ := subscribeOne(stream)
		subs = append(subs, sub)
	}
	defer func() {
		for _, s := range subs {
			s.Unsubscribe()
		}
	}()

	_, _, errs := subscribeOne(stream)
	var err error
	select {
	case err = <-errs:
	case <-time.After(3 * time.Second):
		t.Fatal("the subscription past the limit neither succeeded nor reported")
	}

	if !errors.Is(err, coordination.ErrSubscriberLimit) {
		t.Fatalf("error is not ErrSubscriberLimit: %v", err)
	}
	msg := err.Error()
	// What ran out, and the size of the thing that ran out.
	for _, want := range []string{"Subject subscriber limit", "slots are held by live subscriptions", "Unsubscribe"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message does not mention %q:\n%s", want, msg)
		}
	}
	// Who holds them: this test file is the call site of every slot.
	if !strings.Contains(msg, "lifetime_test.go") {
		t.Errorf("message does not name the holders' call site:\n%s", msg)
	}
}

// TestSlotFreedByLimitIsReusable checks the limit is a ceiling on live
// subscriptions and nothing more: release one and the next caller gets in.
func TestSlotFreedByLimitIsReusable(t *testing.T) {
	obs, stream := coordination.Subject[int](coordination.BufCapSignal)

	subs := make([]rx.Subscription, 0, coordination.MaxSubscribers)
	for range coordination.MaxSubscribers {
		sub, _, _ := subscribeOne(stream)
		subs = append(subs, sub)
	}
	defer func() {
		for _, s := range subs[1:] {
			s.Unsubscribe()
		}
	}()

	subs[0].Unsubscribe()

	sub, values, errs := subscribeOne(stream)
	defer sub.Unsubscribe()
	obs(11, nil, false)
	if got := awaitValue(t, "after a slot was freed", values, errs); got != 11 {
		t.Errorf("got %d, want 11", got)
	}
}

// TestCompletionReachesEverySubscriber checks the producer's done signal still
// terminates every live subscription, and that a subscriber arriving after
// completion is completed rather than left hanging on a dead Subject.
func TestCompletionReachesEverySubscriber(t *testing.T) {
	obs, stream := coordination.Subject[int](coordination.BufCapSignal)

	const n = 3
	dones := make([]chan error, n)
	subs := make([]rx.Subscription, n)
	for i := range n {
		done := make(chan error, 1)
		dones[i] = done
		subs[i] = stream.Subscribe(rx.GoroutineContext(), func(_ int, err error, complete bool) {
			if complete {
				select {
				case done <- err:
				default:
				}
			}
		})
	}
	defer func() {
		for _, s := range subs {
			s.Unsubscribe()
		}
	}()

	sentinel := errors.New("producer finished")
	obs(0, sentinel, true)

	for i := range n {
		select {
		case err := <-dones[i]:
			if !errors.Is(err, sentinel) {
				t.Errorf("subscriber %d completed with %v, want %v", i, err, sentinel)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("subscriber %d never completed", i)
		}
	}

	late := make(chan error, 1)
	lateSub := stream.Subscribe(rx.GoroutineContext(), func(_ int, err error, complete bool) {
		if complete {
			select {
			case late <- err:
			default:
			}
		}
	})
	defer lateSub.Unsubscribe()
	select {
	case err := <-late:
		if !errors.Is(err, sentinel) {
			t.Errorf("late subscriber completed with %v, want %v", err, sentinel)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("a subscriber arriving after completion was left hanging")
	}
}

// TestUnsubscribeDeliversNoCompletion guards the seam between the leg's
// private buffer and the caller: detaching closes that buffer, and the caller
// must not see that close as an ordinary completion of the Subject.
func TestUnsubscribeDeliversNoCompletion(t *testing.T) {
	_, stream := coordination.Subject[int](coordination.BufCapSignal)

	completed := make(chan struct{}, 1)
	sub := stream.Subscribe(rx.GoroutineContext(), func(_ int, _ error, complete bool) {
		if complete {
			select {
			case completed <- struct{}{}:
			default:
			}
		}
	})
	time.Sleep(50 * time.Millisecond)
	sub.Unsubscribe()

	select {
	case <-completed:
		t.Fatal("Unsubscribe delivered a completion to the observer")
	case <-time.After(250 * time.Millisecond):
	}
}
