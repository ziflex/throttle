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
    throttler *throttle.Throttler[*http.Response]
}

func NewApiClient(rps uint64) *ApiClient {
    return &ApiClient{
        transport: &http.Client{},
        throttler: throttle.New[*http.Response](rps),
    }
}

func (c *ApiClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
    return c.throttler.Do(func() (*http.Response, error) {
        select {
            case <-ctx.Done():
                return nil, ctx.Err()
            default: 
                return c.transport.Do(req)
        }
    })
}
```

## Options

### Clock offset
Since client and server machines have different clocks they are probably out of sync, thus you might want to add a clock offset between the throttler's time windows.

#### Static offset
Just a static value

```go
package myapp

import (
	"time"
	"github.com/ziflex/throttle"
)

func main() {
	throttler := throttle.New[any](10, throttle.WithStaticClockOffset(time.Millisecond * 250))	
}
```

#### Dynamic offset
A function the receives the calculated sleep duration and returns an offset that is added to it:

```go
package myapp

import (
	"time"
	"github.com/ziflex/throttle"
)

func main() {
	throttler := throttle.New[any](10, throttle.WithDynamicClockOffset(func(sleepDur time.Duration) time.Duration {
        if sleepDur < (time.Millisecond * 100) {
			return time.Millisecond * 100
        }
		
		return sleepDur
	}))	
}
```