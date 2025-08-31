package throttle_test

import (
	"fmt"
	"github.com/ziflex/throttle"
	"math"
	"sync"
	"testing"
	"time"
)

func seconds(fraction float64) time.Duration {
	return time.Duration(float64(time.Second) * fraction)
}

// mockClock is a test implementation of Clock for testing purposes
type mockClock struct {
	currentTime time.Time
	sleepCalls  []time.Duration
}

func (m *mockClock) Now() time.Time {
	return m.currentTime
}

func (m *mockClock) Sleep(dur time.Duration) {
	m.sleepCalls = append(m.sleepCalls, dur)
	m.currentTime = m.currentTime.Add(dur)
}

func TestWithClock(t *testing.T) {
	mock := &mockClock{currentTime: time.Now()}

	// Create throttler with custom clock
	throttler := throttle.New(1, throttle.WithClock(mock))

	// First call should not sleep
	throttler.Acquire()
	if len(mock.sleepCalls) != 0 {
		t.Fatalf("Expected no sleep calls on first acquire, got %d", len(mock.sleepCalls))
	}

	// Second call should trigger sleep since limit is 1
	throttler.Acquire()
	if len(mock.sleepCalls) != 1 {
		t.Fatalf("Expected 1 sleep call on second acquire, got %d", len(mock.sleepCalls))
	}

	// Verify the sleep duration is reasonable (should be close to 1 second)
	sleepDur := mock.sleepCalls[0]
	if sleepDur < 900*time.Millisecond || sleepDur > 1100*time.Millisecond {
		t.Fatalf("Expected sleep duration around 1 second, got %v", sleepDur)
	}
}

func TestThrottler_Do_Consistent(t *testing.T) {
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
			throttler := throttle.New(useCase.Limit)
			ts := time.Now()

			var wg sync.WaitGroup
			wg.Add(useCase.Calls)

			for range useCase.Calls {
				go func() {
					throttler.Acquire()
					calls <- time.Now()
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

func TestThrottler_Do_Sporadic(t *testing.T) {
	type Burst struct {
		Warmup  time.Duration
		Latency time.Duration
		Calls   int
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
			var buffer int

			for _, tp := range useCase.Calls {
				buffer += tp.Calls
			}

			calls := make(chan time.Time, buffer)
			throttler := throttle.New(useCase.Limit)
			ts := time.Now()

			var wg sync.WaitGroup
			wg.Add(len(useCase.Calls))

			go func() {
				for _, tpl := range useCase.Calls {
					warmup := tpl.Warmup
					latency := tpl.Latency
					callNum := tpl.Calls

					if warmup > 0 {
						time.Sleep(warmup)
					}

					for range callNum {
						throttler.Acquire()

						if latency > 0 {
							time.Sleep(latency)
						}

						calls <- time.Now()
					}

					wg.Done()
				}
			}()

			wg.Wait()
			close(calls)

			groups := map[float64]uint64{}

			for c := range calls {
				diff := c.Sub(ts)
				dur := math.Abs(math.Floor(diff.Seconds()))
				groups[dur]++

				// fmt.Println(fmt.Sprintf("Elapsed %ds", int64(dur)))
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

func TestThrottler_Do_Parallel(t *testing.T) {
	type Call struct {
		Latency time.Duration
	}

	useCases := []struct {
		Limit    uint64
		Calls    []Call
		Expected map[float64]uint64
	}{
		{
			Limit: 1,
			Calls: []Call{
				{},
				{},
				{},
				{},
				{},
			},
			Expected: map[float64]uint64{
				0: 1,
				1: 1,
				2: 1,
				3: 1,
				4: 1,
			},
		},
		{
			Limit: 5,
			Calls: []Call{
				{
					Latency: seconds(0.99),
				},
				{
					Latency: seconds(0.99),
				},
				{
					Latency: seconds(0.99),
				},
				{
					Latency: seconds(0.99),
				},
				{
					Latency: seconds(0.99),
				},
			},
			Expected: map[float64]uint64{
				0: 5,
			},
		},

		{
			Limit: 5,
			Calls: []Call{
				{
					Latency: seconds(0.1),
				},
				{
					Latency: seconds(0.1),
				},
				{
					Latency: seconds(0.1),
				},
				{
					Latency: seconds(0.1),
				},
				{
					Latency: seconds(0.1),
				},
				{},
			},
			Expected: map[float64]uint64{
				0: 5,
				1: 1,
			},
		},
	}

	for _, useCase := range useCases {
		t.Run(fmt.Sprintf("Parallel %d RPS", useCase.Limit), func(t *testing.T) {
			calls := make(chan time.Time, len(useCase.Calls))
			throttler := throttle.New(useCase.Limit)
			ts := time.Now()

			var wg sync.WaitGroup
			wg.Add(len(useCase.Calls))

			for _, tpl := range useCase.Calls {
				go func(latency time.Duration) {
					defer wg.Done()

					throttler.Acquire()

					if latency > 0 {
						time.Sleep(latency)
					}

					calls <- time.Now()
				}(tpl.Latency)
			}

			wg.Wait()
			close(calls)

			groups := map[float64]uint64{}

			for c := range calls {
				diff := c.Sub(ts)
				dur := math.Abs(math.Floor(diff.Seconds()))
				groups[dur]++

				// fmt.Println(fmt.Sprintf("Elapsed %ds", int64(dur)))
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
