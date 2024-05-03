package throttle

import "time"

type (
	// options holds configuration settings for the throttler.
	options struct {
		// additional time to be added when calculating sleep durations
		clockOffset ClockOffsetProvider
	}

	Option func(opts *options)
)

func buildOptions(setters []Option) *options {
	opts := &options{}

	for _, setter := range setters {
		setter(opts)
	}

	return opts
}

// WithStaticClockOffset returns an Option that sets the static clock offset in the throttler options.
// This is useful for adding extra time to the throttle's wait periods, for example, to account for clock skew.
func WithStaticClockOffset(offset time.Duration) Option {
	return func(opts *options) {
		opts.clockOffset = func() time.Duration {
			return offset
		}
	}
}

// WithDynamicClockOffset returns an Option that sets the dynamic clock offset in the throttler options.
// This is useful for adding extra time to the throttle's wait periods, for example, to account for clock skew.
func WithDynamicClockOffset(provider ClockOffsetProvider) Option {
	return func(opts *options) {
		opts.clockOffset = provider
	}
}
