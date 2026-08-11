package coordination_test

import (
	"testing"
	"time"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/components/coordination"
)

// TestSubjectReturnsBothSides verifies that Subject returns a usable
// rx.Observer[T] and rx.Observable[T]: the observer can emit, and the
// observable delivers the value to a subscriber.
func TestSubjectReturnsBothSides(t *testing.T) {
	obs, stream := coordination.Subject[int](coordination.BufCapSignal)

	done := make(chan int, 1)
	sub := stream.Subscribe(rx.GoroutineContext(), func(next int, err error, complete bool) {
		if !complete {
			done <- next
		}
	})
	defer sub.Unsubscribe()

	obs(42, nil, false)

	select {
	case got := <-done:
		if got != 42 {
			t.Errorf("expected 42, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Subject delivery")
	}
}

// TestPointerSubjectBuffers verifies that a pointer-event Subject with
// BufCapPointer capacity does not block the caller on rapid sequential
// emissions (the frame-goroutine non-blocking invariant).
func TestPointerSubjectBuffers(t *testing.T) {
	obs, _ := coordination.Subject[int](coordination.BufCapPointer)

	// Emit BufCapPointer values in tight succession without a subscriber.
	// If the buffer were too small or zero, this would block the goroutine.
	for i := range coordination.BufCapPointer {
		obs(i, nil, false)
	}
}

// TestBufCapConstants verifies the documented capacity values satisfy the
// experiment-derived minimums.
func TestBufCapConstants(t *testing.T) {
	// BufCapPointer must be ≥ 2×60 fps = 120.
	if coordination.BufCapPointer < 120 {
		t.Errorf("BufCapPointer=%d is below the 2×60fps minimum of 120", coordination.BufCapPointer)
	}
	// BufCapSignal: experiment recommended 8.
	if coordination.BufCapSignal < 8 {
		t.Errorf("BufCapSignal=%d is below the experiment-recommended minimum of 8", coordination.BufCapSignal)
	}
}
