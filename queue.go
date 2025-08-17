package async

import (
	"sync"
)

type queue[T any] struct {
	mu      sync.Mutex
	cond    *sync.Cond
	results []T
	closed  bool
	out     chan T
}

// newQueue creates a go-routine safe expandable queue
func newQueue[T any]() *queue[T] {
	rq := &queue[T]{}
	rq.cond = sync.NewCond(&rq.mu)
	return rq
}

// Close marks the queue as closed and wakes all waiting goroutines
func (rq *queue[T]) Close() {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if rq.closed {
		return
	}

	rq.closed = true
	rq.cond.Broadcast()
}

// Send appends a result to the queue and notifies waiting consumers
func (rq *queue[T]) Send(r T) {
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
func (rq *queue[T]) Receive() <-chan T {
	rq.mu.Lock()
	if rq.out != nil {
		rq.mu.Unlock()
		return rq.out
	}
	rq.out = make(chan T)
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
