package async

import (
	"context"
	"errors"
)

type WorkerGroupOpts struct {
	FastFail bool
}

type WorkerGroup struct {
	pool *WorkerPool
	opts WorkerGroupOpts
}

type WorkerGroupOptsFunc func(*WorkerGroupOpts)

func NewWorkerGroup(workerCount int, wopts ...WorkerGroupOptsFunc) *WorkerGroup {
	workerOpts := WorkerGroupOpts{}

	for _, opt := range wopts {
		opt(&workerOpts)
	}

	poolOpts := WorkerPoolOpts{
		FastFail:        workerOpts.FastFail,
		WorkQueueSize:   workerCount * 2,
		ErrorBufferSize: workerCount,
	}

	return &WorkerGroup{
		pool: NewWorkerPool(workerCount, poolOpts),
	}
}

func withFastFail() WorkerGroupOptsFunc {
	return func(o *WorkerGroupOpts) {
		o.FastFail = true
	}
}

func (g *WorkerGroup) Wait() error {
	return g.pool.Wait()
}

func (g *WorkerGroup) Errors() []error {
	return g.pool.Errors()
}

// Execute manages to execute the Activity with the WorkerGroup and returns a future
func Execute[In any, Out any](ctx context.Context, grp *WorkerGroup, executor Activity[In, Out], req In) Future[Out] {
	future := NewFuture[Out]()

	resolveFutureWithError := func(err error) {
		future.Resolve(*new(Out), err)
		grp.pool.ReportError(err)
	}

	// pre-execute checks for cancellation and errors
	select {
	case <-ctx.Done():
		resolveFutureWithError(ctx.Err())
		return future
	case <-grp.pool.CancelSignal():
		resolveFutureWithError(errors.New(ErrMsgTerminatedDueToFastFail))
		return future
	default:
	}

	// create the task
	task := WorkerTask{
		ctx: ctx,
		execute: func() {
			select {
			case <-ctx.Done():
				resolveFutureWithError(ctx.Err())
				return
			case <-grp.pool.CancelSignal():
				resolveFutureWithError(errors.New(ErrMsgTerminatedDueToFastFail))
				return
			default:
			}

			res, err := executor(ctx, req)
			future.Resolve(res, err)
			if err != nil {
				grp.pool.ReportError(err)
			}
		},
	}

	// submit the task
	if err := grp.pool.Submit(task); err != nil {
		resolveFutureWithError(err)
	}

	return future
}
