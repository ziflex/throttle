package throttle

type (
	// options holds configuration settings for the throttler.
	options struct {
		clock Clock
	}

	Option func(opts *options)
)

func buildOptions(setters []Option) *options {
	opts := &options{}

	for _, setter := range setters {
		setter(opts)
	}

	if opts.clock == nil {
		opts.clock = &DefaultClock{}
	}

	return opts
}

// WithClock sets a custom implementation of Clock interface.
func WithClock(clock Clock) Option {
	return func(opts *options) {
		opts.clock = clock
	}
}
