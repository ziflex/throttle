# Throttle
> Dead-simple thread-safe throttler.

``throttle`` provides a thread-safe mechanism to throttle function calls, ensuring that the execution rate does not exceed a specified limit.    
Each instance operates independently, making it possible to control various functions or processes with different rate limits concurrently.

## Install
```shell
go get github.com/ziflex/throttle
```

## Quick start

```go
package myapp

import (
    "context"
    "net/http"
    "github.com/ziflex/throttle"
)

type ApiClient struct {
    transport *http.Client
    throttler *throttle.Throttler
}

func NewApiClient(rps uint64) *ApiClient {
    return &ApiClient{
        transport: &http.Client{},
        throttler: throttle.New(rps),
    }
}

func (c *ApiClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	c.throttler.Acquire()
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return c.transport.Do(req)
	}
}
```

## Options

### Clock
`Clock` type is an interface that allows you to provide custom clock mechanism that's different from the system one.   
It has just 2 methods: ``Now()`` and ``Sleep(time.Duration)``.   
It might be useful to use a custom implementation to provide a more nuanced timing mechanism. 

```go
package myapp

import (
    "time"
    "github.com/ziflex/throttle"
)

type MyClock struct {
    offset time.Duration
}

func (c *MyClock) Now() time.Time {
    return time.Now().Add(c.offset)
}

func (c *MyClock) Sleep(dur time.Duration) {
    time.Sleep(dur + c.offset)
}

func main() {
    throttler := throttle.New(10, throttle.WithClock(&MyClock{time.Millisecond * 250}))
}
```

## Helpers
### RoundTripper
The package contains a helper that wraps the standard `http.RoundTripper` interface and provides a throttling mechanism.

```go
package myapp

import (
    "context"
    "net/http"
    "github.com/ziflex/throttle"
)

func main() {
    transport := &http.Transport{}
    client := &http.Client{
        Transport: throttle.NewRoundTripper(transport, 10),
    }

    req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
    client.Do(req)
}
```