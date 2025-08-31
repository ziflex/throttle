package throttle_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ziflex/throttle"
)

func TestNewRoundTripper(t *testing.T) {
	// Create a test server that responds immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// Create a throttled round tripper
	transport := throttle.NewRoundTripper(http.DefaultTransport, 2)

	// Create a client with the throttled transport
	client := &http.Client{Transport: transport}

	// Make multiple requests to test throttling
	start := time.Now()
	for i := 0; i < 3; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		resp.Body.Close()
	}
	elapsed := time.Since(start)

	// With a limit of 2 RPS, 3 requests should take at least 1 second
	if elapsed < time.Second {
		t.Fatalf("Expected at least 1 second for 3 requests with 2 RPS limit, got %v", elapsed)
	}
}

func TestNewRoundTripperWith(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// Create a throttler and transport separately
	throttler := throttle.New(1)
	transport := throttle.NewRoundTripperWith(http.DefaultTransport, throttler)

	// Create a client with the throttled transport
	client := &http.Client{Transport: transport}

	// Make multiple requests to test throttling
	start := time.Now()
	for i := 0; i < 2; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		resp.Body.Close()
	}
	elapsed := time.Since(start)

	// With a limit of 1 RPS, 2 requests should take at least 1 second
	if elapsed < time.Second {
		t.Fatalf("Expected at least 1 second for 2 requests with 1 RPS limit, got %v", elapsed)
	}
}

func TestThrottledRoundTripper_RoundTrip(t *testing.T) {
	// Create a test server that tracks request times
	var requestTimes []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// Create a throttled transport with limit of 2 RPS
	transport := throttle.NewRoundTripper(http.DefaultTransport, 2)

	// Create requests manually
	req1, _ := http.NewRequest("GET", server.URL, nil)
	req2, _ := http.NewRequest("GET", server.URL, nil)
	req3, _ := http.NewRequest("GET", server.URL, nil)

	// Execute requests and measure timing
	start := time.Now()

	resp1, err1 := transport.RoundTrip(req1)
	if err1 != nil {
		t.Fatalf("First request failed: %v", err1)
	}
	resp1.Body.Close()

	resp2, err2 := transport.RoundTrip(req2)
	if err2 != nil {
		t.Fatalf("Second request failed: %v", err2)
	}
	resp2.Body.Close()

	resp3, err3 := transport.RoundTrip(req3)
	if err3 != nil {
		t.Fatalf("Third request failed: %v", err3)
	}
	resp3.Body.Close()

	elapsed := time.Since(start)

	// With 2 RPS limit, 3 requests should take at least 1 second
	if elapsed < time.Second {
		t.Fatalf("Expected at least 1 second for 3 requests with 2 RPS limit, got %v", elapsed)
	}

	// Verify we got exactly 3 requests
	if len(requestTimes) != 3 {
		t.Fatalf("Expected 3 requests, got %d", len(requestTimes))
	}
}
