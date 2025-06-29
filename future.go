package async

import (
	"context"
)

type Future[T any] struct {
	resChan chan T
	errChan chan error
}

// Get returns the result if available, otherwise zero value of T.
// It ignores any error — group handling that.
func (a *Future[T]) Get(ctx context.Context) T {
	var zero T
	select {
	case <-ctx.Done():
		a.errChan <- ctx.Err()
		return zero
	case val := <-a.resChan:
		return val
	}
}

// Error returns the error if available, otherwise nil.
func (a *Future[T]) Error() error {
	select {
	case err := <-a.errChan:
		return err
	default:
		return nil
	}
}
