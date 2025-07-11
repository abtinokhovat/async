package async

import (
	"sync"
)

type resultQueue[T any] struct {
	mu      sync.Mutex
	cond    *sync.Cond
	results []TaskResult[T]
	closed  bool
	out     chan TaskResult[T]
}

// NewResultQueue creates a new queue with internal signaling
func newResultQueue[T any]() *resultQueue[T] {
	rq := &resultQueue[T]{}
	rq.cond = sync.NewCond(&rq.mu)
	return rq
}

// Close marks the queue as closed and wakes all waiting goroutines
func (rq *resultQueue[T]) Close() {
	rq.mu.Lock()
	rq.closed = true
	rq.cond.Broadcast()
	rq.mu.Unlock()
}

// Send appends a result to the queue and notifies waiting consumers
func (rq *resultQueue[T]) Send(r TaskResult[T]) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if rq.closed {
		return
	}

	rq.results = append(rq.results, r)
	rq.cond.Signal()
}

// Receive starts the dispatcher and returns a read-only result channel
// that multiple consumers can receive from concurrently.
func (rq *resultQueue[T]) Receive() <-chan TaskResult[T] {
	rq.mu.Lock()
	if rq.out != nil {
		rq.mu.Unlock()
		return rq.out
	}
	rq.out = make(chan TaskResult[T])
	rq.mu.Unlock()

	go func() {
		defer close(rq.out)
		for {
			rq.mu.Lock()
			for len(rq.results) == 0 && !rq.closed {
				rq.cond.Wait()
			}
			if len(rq.results) == 0 && rq.closed {
				rq.mu.Unlock()
				return
			}
			item := rq.results[0]
			rq.results = rq.results[1:]
			rq.mu.Unlock()

			rq.out <- item
		}
	}()

	return rq.out
}
