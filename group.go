package async

import (
	"context"
	"sync"
)

type task struct {
	ctx     context.Context
	execute func()
}

type Group struct {
	wg        sync.WaitGroup
	workQueue chan task
	once      sync.Once

	errCh chan error
	mu    sync.Mutex
	errs  []error
}

func NewGroup(bufferSize, workerCount int) *Group {
	grp := &Group{
		workQueue: make(chan task, bufferSize),
		errCh:     make(chan error, 100),
	}

	grp.workerConstructorLoop(workerCount)
	grp.errorCollectorLoop()

	return grp
}

func (g *Group) workerConstructorLoop(count int) {
	if count > 0 {
		for i := 0; i < count; i++ {
			go g.worker()
		}
	}
}

func (g *Group) errorCollectorLoop() {
	go func() {
		for err := range g.errCh {
			// add error to error array
			if err != nil {
				g.mu.Lock()
				g.errs = append(g.errs, err)
				g.mu.Unlock()
			}
		}
	}()
}

func (g *Group) worker() {
	for t := range g.workQueue {
		t.execute()
	}
}

func (g *Group) submit(t task) {
	g.wg.Add(1)
	if g.workQueue != nil {
		g.workQueue <- t
		return
	}

	// no concurrency limit: launch a goroutine for each task
	go func() {
		t.execute()
		g.wg.Done()
	}()
}

// Execute manages to execute the Activity with the Group and returns a future
func Execute[Req any, Res any](ctx context.Context, grp *Group, req Req, executor Activity[Req, Res]) Future[Res] {
	res := &Future[Res]{
		resChan: make(chan Res, 1),
		errChan: make(chan error, 1),
	}

	if ctx.Err() != nil {
		res.errChan <- ctx.Err()
		grp.errCh <- ctx.Err()
		return *res
	}

	grp.submit(task{
		ctx: ctx,
		execute: func() {
			defer grp.wg.Done()
			result, err := executor(ctx, req)
			if err != nil {
				res.errChan <- err
				grp.errCh <- err
			} else {
				res.resChan <- result
			}
		},
	})

	return *res
}

func (g *Group) Wait() {
	g.once.Do(func() { close(g.workQueue) }) // close only once
	g.wg.Wait()
}
