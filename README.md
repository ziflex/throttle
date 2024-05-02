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

func (c *ApiClient) Do(req *http.Request) (*http.Response, error) {
	c.throttler.Wait()
	
	return c.transport.Do(req)
}
```