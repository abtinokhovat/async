package async

import (
	"context"
	"sync/atomic"
)

type result[T any] struct {
	val *T
	err error
}

type Future[T any] struct {
	resolved int64 // 0 = not resolved, 1 = resolved
	res      *result[T]
	resChan  chan *result[T]
}

// NewFuture creates a new Future instance
func NewFuture[T any]() Future[T] {
	return Future[T]{
		resChan: make(chan *result[T], 1),
	}
}

// Resolve sets the result and marks the future as resolved
// This should only be called once per Future, but is safe for concurrent calls
func (f *Future[T]) Resolve(val T, err error) {
	if atomic.CompareAndSwapInt64(&f.resolved, 0, 1) {
		f.res = &result[T]{val: &val, err: err}
		f.resChan <- f.res
		close(f.resChan)
	}
}

// Get returns the result if available, otherwise blocks until resolved or context is cancelled.
// After the first resolution, subsequent calls return the cached result immediately.
func (f *Future[T]) Get(ctx context.Context) T {
	// Fast path: check if already resolved
	if atomic.LoadInt64(&f.resolved) == 1 && f.res != nil && f.res.val != nil {
		return *f.res.val
	}

	// Slow path: wait for resolution
	select {
	case <-ctx.Done():
		var zero T
		return zero
	case res := <-f.resChan:
		if res != nil && res.val != nil {
			return *res.val
		}

		var zero T
		return zero
	}
}

// Error returns the error if available, otherwise blocks until resolved or context is cancelled.
// After the first resolution, subsequent calls return the cached error immediately.
func (f *Future[T]) Error(ctx context.Context) error {
	if atomic.LoadInt64(&f.resolved) == 1 && f.res != nil {
		return f.res.err
	}

	// Slow path: wait for resolution
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-f.resChan:
		if res != nil {
			return res.err
		}
		return nil
	}
}

// IsResolved returns true if the future has been resolved
func (f *Future[T]) IsResolved() bool {
	return atomic.LoadInt64(&f.resolved) == 1
}

// TryGet returns the result and a boolean indicating if the future is resolved
// This is a non-blocking operation
func (f *Future[T]) TryGet() (T, error, bool) {
	if atomic.LoadInt64(&f.resolved) == 1 && f.res != nil {
		val := *f.res.val
		return val, f.res.err, true
	}
	var zero T
	return zero, nil, false
}
