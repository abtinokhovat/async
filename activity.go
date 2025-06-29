package async

import (
	"context"
	"time"
)

type Void = struct{}

type (
	ActivityOption struct {
		ScheduleToCloseTimeout time.Duration
		RetryPolicy            RetryPolicy
	}

	RetryPolicy struct {
		// Backoff interval for the first retry. If BackoffCoefficient is 1.0, then it is used for all retries.
		// If not set or set to 0, a default interval of 1s will be used.
		InitialInterval time.Duration

		// Coefficient used to calculate the next retry backoff interval.
		// The next retry interval is the previous interval multiplied by this coefficient.
		// Must be 1 or larger. Default is 2.0.
		BackoffCoefficient float64

		// Maximum backoff interval between retries. Exponential backoff leads to interval increase.
		// This value is the cap of the interval. Default is 100x of an initial interval.
		MaximumInterval time.Duration

		// Maximum number of attempts. When exceeded, the retries stop even if not expired yet.
		// If not set or set to 0, it means unlimited and rely on activity ScheduleToCloseTimeout to stop.
		MaximumAttempts int32
	}
)

type Activity[Req any, Res any] func(ctx context.Context, req Req) (Res, error)

type VoidActivity = Activity[Void, Void]

// NewVoidActivity adapts a function with no request and no result (func(ctx) error)
// into an Activity that fits the standard async model.
func NewVoidActivity(fn func(ctx context.Context) error) VoidActivity {
	return func(ctx context.Context, _ Void) (Void, error) {
		return Void{}, fn(ctx)
	}
}

// NewActivityWithReq adapts a function with a request but no result (func(ctx, req) error)
// into an Activity.
func NewActivityWithReq[Req any](fn func(ctx context.Context, req Req) error) Activity[Req, Void] {
	return func(ctx context.Context, req Req) (Void, error) {
		return Void{}, fn(ctx, req)
	}
}

// NewActivityWithRes adapts a function with no request but a result (func(ctx) (T, error))
// into an Activity.
func NewActivityWithRes[T any](fn func(ctx context.Context) (T, error)) Activity[Void, T] {
	return func(ctx context.Context, _ Void) (T, error) {
		return fn(ctx)
	}
}
