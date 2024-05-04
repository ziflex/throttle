package throttle

import "time"

type Clock interface {
	Now() time.Time
	Sleep(dur time.Duration)
}

type DefaultClock struct{}

func (c *DefaultClock) Now() time.Time {
	return time.Now()
}

func (c *DefaultClock) Sleep(dur time.Duration) {
	time.Sleep(dur)
}
