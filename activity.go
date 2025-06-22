package async

import "context"

type Void = struct{}

type Activity[T any, Req any] func(ctx context.Context, req Req) (T, error)

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
func NewActivityWithReq[Req any](fn func(ctx context.Context, req Req) error) Activity[Void, Req] {
	return func(ctx context.Context, req Req) (Void, error) {
		return Void{}, fn(ctx, req)
	}
}

// NewActivityWithRes adapts a function with no request but a result (func(ctx) (T, error))
// into an Activity.
func NewActivityWithRes[T any](fn func(ctx context.Context) (T, error)) Activity[T, Void] {
	return func(ctx context.Context, _ Void) (T, error) {
		return fn(ctx)
	}
}
