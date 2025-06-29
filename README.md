# Async Go Library

A lightweight Go library for managing asynchronous operations with futures, groups, and rate limiting.

## Features

- **Futures**: Handle asynchronous results with error handling
- **Groups**: Coordinate multiple async operations
- **Activities**: Type-safe function execution
- **Rate Limiting**: Execute activities with concurrency barriers

## Usage

### Basic Async Execution

```go
package main

import (
    "context"
    "fmt"
    "time"
    "./async"
)

func main() {
    ctx := context.Background()
    grp := async.NewGroup(10, 0) // buffer: 10, unlimited concurrency

    // Create an activity
    activity := func(ctx context.Context, req string) (string, error) {
        time.Sleep(100 * time.Millisecond) // Simulate work
        return "Processed: " + req, nil
    }

    // Execute single activity
    future := async.ExecuteAll(ctx, grp, "hello", activity)

    // Wait for completion
    if err := grp.Wait(); err != nil {
        panic(err)
    }

    // Get result
    result := future.Get(ctx)
    fmt.Println(result) // Output: Processed: hello
}
```

### Rate Limited Execution (Barrier Pattern)

Execute multiple activities with a concurrency limit:

```go
package main

import (
    "context"
    "fmt"
    "time"
    "./async"
)

func main() {
    ctx := context.Background()
    grp := async.NewGroup(100, 0) // buffer: 100, unlimited concurrency

    // Create an activity that simulates API calls
    activity := func(ctx context.Context, req int) (string, error) {
        time.Sleep(200 * time.Millisecond) // Simulate API call
        return fmt.Sprintf("API response for request %d", req), nil
    }

    // Create 20 requests
    requests := make([]int, 20)
    for i := range requests {
        requests[i] = i + 1
    }

    // Execute with barrier - only 5 concurrent executions
    futures := async.ExecuteWithBarrier(ctx, grp, requests, activity, 5)

    // Wait for all to complete
    if err := grp.Wait(); err != nil {
        panic(err)
    }

    // Process results
    for i, future := range futures {
        result := future.Get(ctx)
        fmt.Printf("Request %d: %s\n", i+1, result)
    }
}
```

### Void Activities

For functions that don't return values:

```go
package main

import (
    "context"
    "fmt"
    "./async"
)

func main() {
    ctx := context.Background()
    grp := async.NewGroup(10, 0) // buffer: 10, unlimited concurrency

    // Create a void activity
    voidActivity := async.NewVoidActivity(func(ctx context.Context) error {
        fmt.Println("Processing...")
        return nil
    })

    // Execute
    future := async.ExecuteAll(ctx, grp, async.Void{}, voidActivity)

    if err := grp.Wait(); err != nil {
        panic(err)
    }

    // Check for errors
    if err := future.Error(); err != nil {
        panic(err)
    }
}
```

### Group-Level Concurrency Control

You can also control concurrency at the group level:

```go
package main

import (
    "context"
    "fmt"
    "time"
    "./async"
)

func main() {
    ctx := context.Background()
    // Create group with max 3 concurrent operations
    grp := async.NewGroup(100, 3)

    activity := func(ctx context.Context, req int) (string, error) {
        time.Sleep(100 * time.Millisecond)
        return fmt.Sprintf("Processed %d", req), nil
    }

    // These will be limited to 3 concurrent executions by the group
    futures := make([]*async.Future[string], 10)
    for i := 0; i < 10; i++ {
        futures[i] = async.ExecuteAll(ctx, grp, i, activity)
    }

    if err := grp.Wait(); err != nil {
        panic(err)
    }

    for i, future := range futures {
        result := future.Get(ctx)
        fmt.Printf("Result %d: %s\n", i, result)
    }
}
```

## API Reference

### Functions

#### `ExecuteAll[Req, Res]`

Executes a single activity and returns a future.

```go
func ExecuteAll[Req any, Res any](
    ctx context.Context,
    grp *Group,
    req Req,
    executor Activity[Req, Res]
) *Future[Res]
```

#### `ExecuteWithBarrier[Req, Res]`

Executes multiple activities with a concurrency limit (barrier pattern).

```go
func ExecuteWithBarrier[Req any, Res any](
    ctx context.Context,
    grp *Group,
    requests []Req,
    executor Activity[Req, Res],
    concurrencyLimit int,
) []*Future[Res]
```

### Types

#### `Activity[Req, Res]`

A function type that takes a context and request, returning a result and error.

```go
type Activity[Req any, Res any] func(ctx context.Context, req Req) (Res, error)
```

#### `Future[T]`

Represents an asynchronous result.

```go
type Future[T any] struct {
    resChan chan T
    errChan chan error
}
```

#### `Group`

Coordinates multiple async operations with concurrency control.

```go
type Group struct {
    wg              sync.WaitGroup
    mu              sync.Mutex
    errChan         chan error
    semaphore       chan struct{}
    concurrencyLimit int
}

// NewGroup creates a new Group with error buffer and concurrency control
func NewGroup(buffer int, concurrencyLimit int) *Group
```

## Best Practices

1. **Always use contexts**: Pass context to handle cancellation and timeouts
2. **Check for errors**: Always call `grp.Wait()` and handle errors
3. **Use barriers for rate limiting**: Use `ExecuteWithBarrier` when you need to limit concurrent operations
4. **Group-level concurrency**: Set concurrency limits in `NewGroup` for automatic rate limiting
5. **Buffer sizes**: Choose appropriate buffer sizes for groups based on expected error volume
6. **Resource cleanup**: The library handles cleanup automatically, but ensure your activities respect context cancellation

## Examples

### API Rate Limiting

```go
// Execute 100 API calls with max 10 concurrent requests
futures := async.ExecuteWithBarrier(ctx, grp, apiRequests, apiCall, 10)
```

### Database Operations

```go
// Process database operations in batches
futures := async.ExecuteWithBarrier(ctx, grp, dbOperations, dbProcessor, 5)
```

### File Processing

```go
// Process files with limited concurrency to avoid overwhelming the system
futures := async.ExecuteWithBarrier(ctx, grp, filePaths, fileProcessor, 3)
```

### Group-Level Rate Limiting

```go
// Create group with built-in concurrency control
grp := async.NewGroup(100, 5) // Max 5 concurrent operations
// All ExecuteAll calls will respect this limit automatically
```
