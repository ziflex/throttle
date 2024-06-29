package throttle

import (
	"sync"
	"time"
)

const windowSize = time.Second

// Throttler manages the execution of operations so that they don't exceed a specified rate limit.
type Throttler struct {
	mu      sync.Mutex
	window  time.Time
	clock   Clock
	counter uint64
	limit   uint64
}

// New creates a new instance of Throttler with a specified limit.
func New(limit uint64, setters ...Option) *Throttler {
	opts := buildOptions(setters)

	return &Throttler{
		limit: limit,
		clock: opts.clock,
	}
}

// Acquire blocks until the operation can be executed within the rate limit.
func (t *Throttler) Acquire() {
	t.mu.Lock()
	t.advance()
	t.mu.Unlock()
}

// advance updates the throttler state, advancing the window or incrementing the counter as necessary.
func (t *Throttler) advance() {
	// pass through
	if t.limit == 0 {
		return
	}

	clock := t.clock
	now := clock.Now()

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
	clock.Sleep(sleepDur)

	// after sleeping, reset to a new window starting now
	t.reset(clock.Now())
}

// reset starts a new window from the specified start time and resets the operation counter.
func (t *Throttler) reset(window time.Time) {
	t.window = window
	t.counter = 1
}
