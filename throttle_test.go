package throttle_test

import (
	"fmt"
	"github.com/ziflex/throttle"
	"math"
	"sync"
	"testing"
	"time"
)

func currentTime() (time.Time, error) {
	return time.Now(), nil
}

func TestThrottler_Wait_Consistent(t *testing.T) {
	useCases := []struct {
		Limit uint64
		Calls int
	}{
		{
			Limit: 0,
			Calls: 10,
		},
		{
			Limit: 1,
			Calls: 10,
		},
		{
			Limit: 5,
			Calls: 10,
		},
		{
			Limit: 5,
			Calls: 16,
		},
		{
			Limit: 5,
			Calls: 14,
		},
	}

	for _, useCase := range useCases {
		t.Run(fmt.Sprintf("Consistent %d RPS within %d calls", useCase.Limit, useCase.Calls), func(t *testing.T) {
			calls := make(chan time.Time, useCase.Calls)
			call := func(t *throttle.Throttler[time.Time]) {
				ts, _ := t.Do(currentTime)
				calls <- ts
			}

			throttler := throttle.New[time.Time](useCase.Limit)
			ts := time.Now()
			wg := sync.WaitGroup{}
			wg.Add(useCase.Calls)

			for range useCase.Calls {
				go func() {
					call(throttler)
					wg.Done()
				}()
			}

			wg.Wait()
			close(calls)

			groups := map[float64]uint64{}

			for c := range calls {
				diff := c.Sub(ts)
				dur := math.Abs(math.Floor(diff.Seconds()))
				groups[dur]++
			}

			expected := useCase.Limit

			for _, actual := range groups {
				if expected == 0 {
					expected = uint64(useCase.Calls)
				}

				if actual > expected {
					t.Fatal(fmt.Sprintf("Expected %d per second, but got %d", expected, actual))
				}
			}
		})
	}
}

func TestThrottler_Wait_Sporadic(t *testing.T) {
	type Burst struct {
		Warmup  time.Duration
		Latency time.Duration
		Calls   int
	}

	seconds := func(fraction float64) time.Duration {
		return time.Duration(float64(time.Second) * fraction)
	}
	useCases := []struct {
		Limit    uint64
		Calls    []Burst
		Expected map[float64]uint64
	}{
		{
			Limit: 10,
			Calls: []Burst{
				{
					Warmup: seconds(0.99),
					Calls:  5,
				},
				{
					Warmup: seconds(0.99),
					Calls:  2,
				},
				{
					Warmup: seconds(0.5),
					Calls:  4,
				},
			},
			Expected: map[float64]uint64{
				0: 5,
				1: 2,
				2: 4,
			},
		},
		{
			Limit: 5,
			Calls: []Burst{
				{
					Calls:   5,
					Latency: seconds(0.255),
				},
				{
					Warmup:  seconds(0.2),
					Calls:   6,
					Latency: seconds(0.45),
				},
			},
			Expected: map[float64]uint64{
				0: 3,
				1: 3,
				2: 2,
				3: 2,
				4: 1,
			},
		},
	}

	for _, useCase := range useCases {
		t.Run(fmt.Sprintf("Sporadic %d RPS within %d calls", useCase.Limit, useCase.Calls), func(t *testing.T) {
			var totalCalls int

			for _, tp := range useCase.Calls {
				totalCalls += tp.Calls
			}

			calls := make(chan time.Time, totalCalls)
			call := func(t *throttle.Throttler[time.Time], latency time.Duration) {
				if latency > 0 {
					time.Sleep(latency)
				}

				ts, _ := t.Do(currentTime)
				calls <- ts
			}

			throttler := throttle.New[time.Time](useCase.Limit)
			ts := time.Now()
			var wg sync.WaitGroup
			wg.Add(totalCalls)

			go func() {
				for _, tpl := range useCase.Calls {
					if tpl.Warmup > 0 {
						time.Sleep(tpl.Warmup)
					}

					for range tpl.Calls {
						//ts := time.Now()
						call(throttler, tpl.Latency)
						wg.Done()
						//fmt.Println(fmt.Sprintf("Call %dms", time.Since(ts).Milliseconds()))
					}
				}
			}()

			wg.Wait()
			close(calls)

			groups := map[float64]uint64{}

			for c := range calls {
				diff := c.Sub(ts)
				dur := math.Abs(math.Floor(diff.Seconds()))
				groups[dur]++

				//fmt.Println(fmt.Sprintf("Elapsed %ds", int64(dur)))
			}

			for sec, actual := range groups {
				expected, found := useCase.Expected[sec]

				if !found {
					t.Fatal(fmt.Sprintf("Expected to have calls within %ds time range", int64(sec)))
				}

				if actual != expected {
					t.Fatal(fmt.Sprintf("Expected %d per second, but got %d", expected, actual))
				}
			}
		})
	}
}
