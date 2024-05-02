package throttle

import (
	"sync"
	"time"
)

type (
	// Fn represents a function that returns a value of type T and an error.
	Fn[T any] func() (T, error)

	// Throttler manages the execution of operations so that they don't exceed a specified rate limit.
	Throttler[T any] struct {
		mu      sync.Mutex
		window  time.Time
		counter uint64
		limit   uint64
	}
)

// New creates a new instance of Throttler with a specified limit.
func New[T any](limit uint64) *Throttler[T] {
	return &Throttler[T]{
		limit: limit,
	}
}

// Do executes the provided function fn if the rate limit has not been reached.
// It ensures that the operation respects the throttling constraints.
func (t *Throttler[T]) Do(fn Fn[T]) (T, error) {
	t.mu.Lock()
	t.advance()
	res, err := fn()
	t.mu.Unlock()

	return res, err
}

// advance updates the throttler state, advancing the window or incrementing the counter as necessary.
func (t *Throttler[T]) advance() {
	now := time.Now()

	// if this is the first operation, initialize the window
	if t.window.IsZero() {
		t.window = now
	}

	sinceLastCall := now.Sub(t.window)

	// if the current window has expired
	if sinceLastCall > time.Second {
		// start a new window
		t.reset(now)

		return
	}

	nextCount := t.counter + 1

	// if adding another operation doesn't exceed the limit
	if t.limit >= nextCount {
		// increment the counter
		t.counter = nextCount

		return
	}

	// if the limit is reached, wait until the current window expires
	time.Sleep(time.Second - sinceLastCall)

	// after sleeping, reset to a new window starting now
	t.reset(time.Now())
}

// reset starts a new window from the specified start time and resets the operation counter.
func (t *Throttler[T]) reset(window time.Time) {
	t.window = window
	t.counter = 1
}
