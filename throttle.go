package throttle

import (
	"sync"
	"time"
)

type Throttler struct {
	mu      sync.Mutex
	window  time.Time
	counter uint64
	limit   uint64
}

func New(limit uint64) *Throttler {
	t := new(Throttler)
	t.limit = limit

	return t
}

func (t *Throttler) Wait() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	// if first call
	if t.window.IsZero() {
		t.window = now
	}

	sinceLastCall := now.Sub(t.window)

	// if we are past the current window
	// start a new one and exit
	if sinceLastCall > time.Second {
		t.reset(now)

		return
	}

	nextCount := t.counter + 1

	// if we are in the limit and there is enough time left to process next operation
	// we increase the counter and move on
	if t.limit >= nextCount {
		t.counter = nextCount

		return
	}

	leftInWindow := time.Second - sinceLastCall

	// otherwise wait for the next window
	time.Sleep(leftInWindow)

	// new window
	t.reset(time.Now())
}

func (t *Throttler) reset(window time.Time) {
	t.window = window
	t.counter = 1
}
