package throttle

import (
	"sync"
	"time"
)

const windowSize = time.Second

type (
	ClockOffsetProvider func(sleepDur time.Duration) time.Duration

	// Fn represents a function that returns a value of type T and an error.
	Fn[T any] func() (T, error)

	// Throttler manages the execution of operations so that they don't exceed a specified rate limit.
	Throttler[T any] struct {
		mu          sync.Mutex
		window      time.Time
		clockOffset ClockOffsetProvider
		counter     uint64
		limit       uint64
	}
)

// New creates a new instance of Throttler with a specified limit.
func New[T any](limit uint64, setters ...Option) *Throttler[T] {
	opts := buildOptions(setters)

	return &Throttler[T]{
		limit:       limit,
		clockOffset: opts.clockOffset,
	}
}

// Do executes the provided function fn if the rate limit has not been reached.
// It ensures that the operation respects the throttling constraints.
func (t *Throttler[T]) Do(fn Fn[T]) (T, error) {
	t.mu.Lock()
	t.advance()
	t.mu.Unlock()

	return fn()
}

// advance updates the throttler state, advancing the window or incrementing the counter as necessary.
func (t *Throttler[T]) advance() {
	// pass through
	if t.limit == 0 {
		return
	}

	now := time.Now()

	// if this is the first operation, initialize the window
	if t.window.IsZero() {
		t.window = now
	}

	windowDur := now.Sub(t.window)

	// if the current window has expired
	if windowDur > windowSize {
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

	sleepDur := windowSize - windowDur

	// if the limit is reached, wait until the current window expires
	// we use an optional clock offset to account for clock skew.
	time.Sleep(sleepDur + t.clockOffset(sleepDur))

	// after sleeping, reset to a new window starting now
	t.reset(time.Now())
}

// reset starts a new window from the specified start time and resets the operation counter.
func (t *Throttler[T]) reset(window time.Time) {
	t.window = window
	t.counter = 1
}
