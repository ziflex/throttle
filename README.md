# Throttle
> Dead-simple thread-safe rate limiter for Go.

`throttle` provides a thread-safe rate limiting mechanism that ensures your function calls don't exceed a specified rate limit (requests per second). Perfect for API rate limiting, database connection throttling, or any scenario where you need to control the execution rate of operations.

**Key Features:**
- 🚀 **Simple API** - Just call `Acquire()` before your operation
- 🔒 **Thread-safe** - Use from multiple goroutines safely  
- ⚡ **High Performance** - Minimal overhead and memory footprint
- 🎛️ **Configurable** - Custom clock implementations and options
- 🌐 **HTTP Helpers** - Built-in `http.RoundTripper` wrapper

Each throttler instance operates independently, allowing you to control different functions or processes with different rate limits concurrently.

## Install
```shell
go get github.com/ziflex/throttle
```

## Basic Usage

### Simple Rate Limiting
Here's the most basic example - limit operations to 5 per second:

```go
package main

import (
    "fmt"
    "time"
    "github.com/ziflex/throttle"
)

func main() {
    // Create a throttler that allows 5 operations per second
    throttler := throttle.New(5)
    
    // Perform 10 operations - they'll be spread out over time
    for i := 0; i < 10; i++ {
        throttler.Acquire() // This will block if rate limit is exceeded
        fmt.Printf("Operation %d executed at %v\n", i+1, time.Now())
    }
}
```

### No Rate Limiting
Set limit to `0` to disable throttling entirely:

```go
throttler := throttle.New(0) // No rate limiting
throttler.Acquire()          // Returns immediately
```

### Concurrent Usage
The throttler is thread-safe and works perfectly with goroutines:

```go
package main

import (
    "fmt"
    "sync"
    "github.com/ziflex/throttle"
)

func main() {
    throttler := throttle.New(3) // 3 operations per second
    var wg sync.WaitGroup
    
    // Launch 10 goroutines, all will be throttled collectively
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            throttler.Acquire()
            fmt.Printf("Goroutine %d completed\n", id)
        }(i)
    }
    
    wg.Wait()
}
```

## Real-World Examples

### API Client Rate Limiting
A common use case is throttling API requests to respect rate limits:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"
    "github.com/ziflex/throttle"
)

type ApiClient struct {
    httpClient *http.Client
    throttler  *throttle.Throttler
}

func NewApiClient(rps uint64) *ApiClient {
    return &ApiClient{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        throttler:  throttle.New(rps),
    }
}

func (c *ApiClient) Get(ctx context.Context, url string) (*http.Response, error) {
    // Acquire permission before making the request
    c.throttler.Acquire()
    
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, err
    }
    
    return c.httpClient.Do(req)
}

func main() {
    client := NewApiClient(10) // Max 10 requests per second
    
    // Make multiple API calls - they'll be automatically throttled
    for i := 0; i < 25; i++ {
        resp, err := client.Get(context.Background(), "https://api.example.com/data")
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i+1, err)
            continue
        }
        resp.Body.Close()
        fmt.Printf("Request %d completed: %s\n", i+1, resp.Status)
    }
}
```

### Database Connection Throttling
Throttle database operations to prevent overwhelming your database:

```go
package main

import (
    "database/sql"
    "fmt"
    "sync"
    "github.com/ziflex/throttle"
)

type DatabaseManager struct {
    db        *sql.DB
    throttler *throttle.Throttler
}

func NewDatabaseManager(db *sql.DB, maxQueriesPerSecond uint64) *DatabaseManager {
    return &DatabaseManager{
        db:        db,
        throttler: throttle.New(maxQueriesPerSecond),
    }
}

func (dm *DatabaseManager) Query(query string, args ...interface{}) (*sql.Rows, error) {
    dm.throttler.Acquire() // Wait for permission to execute query
    return dm.db.Query(query, args...)
}

func main() {
    // db := sql.Open(...) // Your database connection
    // dbManager := NewDatabaseManager(db, 50) // Max 50 queries per second
    
    var wg sync.WaitGroup
    
    // Execute many queries concurrently, but throttled
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            // rows, err := dbManager.Query("SELECT * FROM users WHERE id = ?", id)
            fmt.Printf("Query %d executed\n", id)
        }(i)
    }
    
    wg.Wait()
}
```

## API Reference

### Core Functions

#### `throttle.New(limit uint64, options ...Option) *Throttler`
Creates a new throttler instance with the specified rate limit.

- `limit`: Maximum number of operations per second (0 = unlimited)
- `options`: Optional configuration (see Options section)
- Returns: A new `*Throttler` instance

#### `throttler.Acquire()`
Blocks the current goroutine until the operation can be executed within the rate limit.

- **Thread-safe**: Can be called from multiple goroutines simultaneously
- **Blocking**: Will sleep if rate limit is exceeded
- **No return value**: Always succeeds (never returns an error)

### Behavior Notes

- **Rate Limit Window**: Uses a sliding window of 1 second
- **Zero Limit**: Setting `limit` to `0` disables throttling (all calls pass through immediately)
- **Thread Safety**: All methods are safe to call from multiple goroutines
- **Memory Efficient**: Minimal memory footprint, no buffering of operations
- **Precise Timing**: Uses system clock by default, customizable via options

## Options

### Clock

The `Clock` interface allows you to provide a custom timing mechanism instead of using the system clock. This is particularly useful for testing, simulation, or when you need custom timing behavior.

**Interface:**
```go
type Clock interface {
    Now() time.Time
    Sleep(dur time.Duration)
}
```

**Custom Clock Example:**
```go
package main

import (
    "fmt"
    "time"
    "github.com/ziflex/throttle"
)

// CustomClock adds an offset to all timing operations
type CustomClock struct {
    offset time.Duration
}

func (c *CustomClock) Now() time.Time {
    return time.Now().Add(c.offset)
}

func (c *CustomClock) Sleep(dur time.Duration) {
    time.Sleep(dur + c.offset)
}

func main() {
    // Create throttler with custom clock that adds 250ms to all operations
    customClock := &CustomClock{offset: 250 * time.Millisecond}
    throttler := throttle.New(5, throttle.WithClock(customClock))
    
    for i := 0; i < 3; i++ {
        start := time.Now()
        throttler.Acquire()
        fmt.Printf("Operation %d: %v\n", i+1, time.Since(start))
    }
}
```

**Testing Clock Example:**
```go
package main

import (
    "sync"
    "time"
    "github.com/ziflex/throttle"
)

// MockClock for deterministic testing
type MockClock struct {
    mu   sync.Mutex
    time time.Time
}

func NewMockClock(start time.Time) *MockClock {
    return &MockClock{time: start}
}

func (m *MockClock) Now() time.Time {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.time
}

func (m *MockClock) Sleep(dur time.Duration) {
    m.mu.Lock()
    m.time = m.time.Add(dur)
    m.mu.Unlock()
}

func (m *MockClock) Advance(dur time.Duration) {
    m.mu.Lock()
    m.time = m.time.Add(dur)
    m.mu.Unlock()
}

// Use in tests for predictable timing
func ExampleWithMockClock() {
    mockClock := NewMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    throttler := throttle.New(2, throttle.WithClock(mockClock))
    
    // Your test logic here
    throttler.Acquire() // Will use mock timing
}
```

## Helpers

### HTTP RoundTripper

The package provides convenient helpers for HTTP client throttling that integrate seamlessly with Go's standard `http` package.

#### `throttle.NewRoundTripper(transport http.RoundTripper, limit uint64, options ...Option) http.RoundTripper`

Wraps any `http.RoundTripper` with throttling functionality.

**Basic HTTP Client Example:**
```go
package main

import (
    "fmt"
    "net/http"
    "github.com/ziflex/throttle"
)

func main() {
    // Create a throttled HTTP client (max 10 requests per second)
    client := &http.Client{
        Transport: throttle.NewRoundTripper(http.DefaultTransport, 10),
        Timeout:   30 * time.Second, // Add timeout for safety
    }
    
    // All requests through this client will be throttled
    for i := 0; i < 20; i++ {
        resp, err := client.Get("https://api.example.com/endpoint")
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i+1, err)
            continue
        }
        resp.Body.Close()
        fmt.Printf("Request %d: %s\n", i+1, resp.Status)
    }
}
```

#### `throttle.NewRoundTripperWith(transport http.RoundTripper, throttler *Throttler) http.RoundTripper`

Use an existing throttler instance for more control:

```go
package main

import (
    "net/http"
    "time"
    "github.com/ziflex/throttle"
)

func main() {
    // Create a shared throttler instance
    sharedThrottler := throttle.New(5)
    
    // Create multiple clients that share the same rate limit
    client1 := &http.Client{
        Transport: throttle.NewRoundTripperWith(http.DefaultTransport, sharedThrottler),
    }
    
    client2 := &http.Client{
        Transport: throttle.NewRoundTripperWith(http.DefaultTransport, sharedThrottler),
    }
    
    // Both clients share the 5 requests/second limit
    go makeRequests(client1, "Client 1")
    go makeRequests(client2, "Client 2")
    
    time.Sleep(5 * time.Second)
}

func makeRequests(client *http.Client, name string) {
    for i := 0; i < 10; i++ {
        resp, _ := client.Get("https://api.example.com/endpoint")
        if resp != nil {
            resp.Body.Close()
            fmt.Printf("%s - Request %d completed\n", name, i+1)
        }
    }
}
```

### Custom Transport with Options
```go
package main

import (
    "net/http"
    "time"
    "github.com/ziflex/throttle"
)

type LoggingClock struct {
    throttle.DefaultClock
}

func (c *LoggingClock) Sleep(dur time.Duration) {
    fmt.Printf("Throttling for %v\n", dur)
    c.DefaultClock.Sleep(dur)
}

func main() {
    // Custom transport with logging
    transport := &http.Transport{
        MaxIdleConns:        10,
        IdleConnTimeout:     30 * time.Second,
    }
    
    // Throttled client with custom clock
    client := &http.Client{
        Transport: throttle.NewRoundTripper(
            transport, 
            2, // 2 requests per second
            throttle.WithClock(&LoggingClock{}),
        ),
    }
    
    // Use the client...
}
```

## Advanced Use Cases

### Rate Limiting Background Workers
```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/ziflex/throttle"
)

type TaskProcessor struct {
    throttler *throttle.Throttler
}

func NewTaskProcessor(tasksPerSecond uint64) *TaskProcessor {
    return &TaskProcessor{
        throttler: throttle.New(tasksPerSecond),
    }
}

func (tp *TaskProcessor) ProcessTask(task string) error {
    tp.throttler.Acquire() // Ensure we don't exceed rate limit
    
    // Simulate task processing
    fmt.Printf("Processing task: %s at %v\n", task, time.Now())
    time.Sleep(100 * time.Millisecond) // Simulate work
    
    return nil
}

func main() {
    processor := NewTaskProcessor(3) // 3 tasks per second max
    
    tasks := []string{"task1", "task2", "task3", "task4", "task5", "task6"}
    
    for _, task := range tasks {
        if err := processor.ProcessTask(task); err != nil {
            log.Printf("Failed to process %s: %v", task, err)
        }
    }
}
```

### File Processing with Rate Limiting
```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "github.com/ziflex/throttle"
)

func processFile(filename string, linesPerSecond uint64) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    throttler := throttle.New(linesPerSecond)
    scanner := bufio.NewScanner(file)
    lineNum := 0
    
    for scanner.Scan() {
        throttler.Acquire() // Rate limit line processing
        lineNum++
        
        line := scanner.Text()
        // Process the line (API call, database insert, etc.)
        fmt.Printf("Processed line %d: %s\n", lineNum, line)
    }
    
    return scanner.Err()
}
```

### Multiple Service Rate Limits
```go
package main

import (
    "github.com/ziflex/throttle"
)

type ServiceManager struct {
    apiThrottler   *throttle.Throttler  // For API calls
    dbThrottler    *throttle.Throttler  // For database operations  
    emailThrottler *throttle.Throttler  // For sending emails
}

func NewServiceManager() *ServiceManager {
    return &ServiceManager{
        apiThrottler:   throttle.New(100), // 100 API calls/sec
        dbThrottler:    throttle.New(50),  // 50 DB operations/sec
        emailThrottler: throttle.New(5),   // 5 emails/sec
    }
}

func (sm *ServiceManager) CallAPI() {
    sm.apiThrottler.Acquire()
    // Make API call...
}

func (sm *ServiceManager) QueryDatabase() {
    sm.dbThrottler.Acquire()
    // Execute database query...
}

func (sm *ServiceManager) SendEmail() {
    sm.emailThrottler.Acquire()
    // Send email...
}
```

## Performance Characteristics

- **Low Overhead**: Minimal CPU and memory usage
- **No Goroutine Leaks**: No background goroutines created
- **Precise Timing**: Accurate rate limiting using system clock
- **Scalable**: Works efficiently with hundreds of goroutines
- **Zero Allocations**: No memory allocations during normal operation (after initialization)

## Thread Safety

All `Throttler` methods are thread-safe and can be called concurrently from multiple goroutines. The throttler uses a mutex internally to ensure consistency.

```go
// Safe to use from multiple goroutines
throttler := throttle.New(10)

go func() { throttler.Acquire() }() // ✅ Safe
go func() { throttler.Acquire() }() // ✅ Safe
go func() { throttler.Acquire() }() // ✅ Safe
```

## FAQ

**Q: What happens when limit is 0?**  
A: Setting limit to 0 disables throttling completely. All `Acquire()` calls return immediately.

**Q: How precise is the rate limiting?**  
A: The throttler uses a sliding window approach with 1-second windows, providing accurate rate limiting for most use cases.

**Q: Can I change the rate limit after creation?**  
A: No, the rate limit is immutable after creating the throttler. Create a new instance if you need a different limit.

**Q: Does this create background goroutines?**  
A: No, the throttler doesn't create any background goroutines. It only uses the calling goroutine.

**Q: What's the memory footprint?**  
A: Very small - just a few fields per throttler instance with no buffering or queuing.

## License

MIT License - see [LICENSE](LICENSE) file for details.