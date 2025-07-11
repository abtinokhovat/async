package async

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrMsgTerminatedDueToFastFail = "terminated due to fast fail"
)

type task struct {
	ctx     context.Context
	execute func()
}

type WorkerGroupOpts struct {
	FastFail bool
}

type WorkerGroup struct {
	wg        sync.WaitGroup
	workQueue chan task

	once      sync.Once
	closeOnce sync.Once

	errCh chan error
	mu    sync.Mutex
	errs  []error

	cancel chan struct{}

	opts WorkerGroupOpts
}

func NewWorkerGroup(workerCount int) *WorkerGroup {
	grp := &WorkerGroup{
		workQueue: make(chan task, workerCount*2),
		errCh:     make(chan error, workerCount),
		cancel:    make(chan struct{}),
	}

	grp.workerLoop(workerCount)
	grp.errorLoop()

	return grp
}

func (g *WorkerGroup) WithFastFail() *WorkerGroup {
	g.opts.FastFail = true
	return g
}

func (g *WorkerGroup) workerLoop(count int) {
	if count > 0 {
		for i := 0; i < count; i++ {
			g.wg.Add(1)
			go g.worker(i)
		}
	}
}

func (g *WorkerGroup) errorLoop() {
	go func() {
		for err := range g.errCh {
			// signal cancel on the first-happened error
			// if fast fail was activated
			if g.opts.FastFail {
				g.once.Do(func() {
					close(g.cancel)
					// fmt.Println("termination signal sent")
				})
			}

			// add error to error array
			if err != nil {
				g.mu.Lock()
				g.errs = append(g.errs, err)
				g.mu.Unlock()
			}
		}
	}()
}

func (g *WorkerGroup) worker(id int) {
	defer g.wg.Done()

	for t := range g.workQueue {
		t.execute()
	}

	// fmt.Printf("exiting worker %d\n", id)
}

func (g *WorkerGroup) submit(t task) {
	if g.workQueue != nil {
		select {
		case g.workQueue <- t:
		}
		return
	}

	// no concurrency limit: launch a goroutine for each task
	go func() {
		t.execute()
	}()
}

// Execute manages to execute the Activity with the WorkerGroup and returns a future
func Execute[Req any, Res any](ctx context.Context, grp *WorkerGroup, executor Activity[Req, Res], req Req) Future[Res] {
	future := NewFuture[Res]()

	resolveFutureWithError := func(err error) {
		future.Resolve(*new(Res), err)
		grp.errCh <- err
	}

	// pre-execute checks for cancellation and errors
	select {
	case <-ctx.Done():
		resolveFutureWithError(ctx.Err())
		return future
	case <-grp.cancel:
		resolveFutureWithError(errors.New(ErrMsgTerminatedDueToFastFail))
		return future
	default:
	}

	// execute is the function that is going to be executed while the worker picks up this task
	execute := func() {
		select {
		case <-ctx.Done():
			resolveFutureWithError(ctx.Err())
			return
		case <-grp.cancel:
			resolveFutureWithError(errors.New(ErrMsgTerminatedDueToFastFail))
			return
		default:
		}

		res, err := executor(ctx, req)
		future.Resolve(res, err)
		if err != nil {
			grp.errCh <- err
		}
	}

	grp.submit(task{
		ctx:     ctx,
		execute: execute,
	})

	return future
}

func (g *WorkerGroup) close() {
	g.closeOnce.Do(func() {
		close(g.workQueue)
	})
}

func (g *WorkerGroup) Wait() error {
	// close the work queue - no more tasks can be submitted
	g.close()

	// wait for all workers to finish processing existing tasks
	g.wg.Wait()

	// now that all workers are done, close an error channel
	// this will terminate the error collector goroutine
	close(g.errCh)

	// return first happened error
	g.mu.Lock()
	var err error
	if len(g.errs) > 0 {
		err = g.errs[0]
	}
	g.mu.Unlock()

	return err
}

func (g *WorkerGroup) Errors() []error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.errs
}
