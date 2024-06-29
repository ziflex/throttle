package throttle

import (
	"net/http"
)

type throttledRoundTripper struct {
	transport http.RoundTripper
	throttler *Throttler
}

func (t *throttledRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.throttler.Acquire()

	return t.transport.RoundTrip(request)
}

func NewRoundTripper(transport http.RoundTripper, limit uint64, setters ...Option) http.RoundTripper {
	return NewRoundTripperWith(transport, New(limit, setters...))
}

func NewRoundTripperWith(transport http.RoundTripper, throttler *Throttler) http.RoundTripper {
	return &throttledRoundTripper{
		transport: transport,
		throttler: throttler,
	}
}
