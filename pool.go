package async

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrMsgWorkerIsClosed          = "worker pool is closed"
	ErrMsgTerminatedDueToFastFail = "terminated due to fast fail"
)

// WorkerPoolOpts contains configuration options for the worker pool
type WorkerPoolOpts struct {
	FastFail        bool
	WorkQueueSize   int
	ErrorBufferSize int
}

// WorkerPool manages a pool of workers and handles common functionality
type WorkerPool struct {
	workerCount int
	workQueue   Queue[WorkerTask]

	wg         sync.WaitGroup
	once       sync.Once
	closeOnce  sync.Once
	close2Once sync.Once

	// error handling
	errCh chan error
	mu    sync.Mutex
	errs  []error

	// cancellation
	cancel chan struct{}
	ctx    context.Context

	// state management
	closed int64 // 0 = open, 1 = closed
	opts   WorkerPoolOpts
}

// WorkerTask represents a task that can be executed by a worker
type WorkerTask struct {
	ctx     context.Context
	execute func()
}

// NewWorkerPool creates a new worker pool with the specified configuration
func NewWorkerPool(workerCount int, opts WorkerPoolOpts) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}

	if opts.WorkQueueSize <= 0 {
		opts.WorkQueueSize = workerCount * 2
	}

	if opts.ErrorBufferSize <= 0 {
		opts.ErrorBufferSize = workerCount
	}

	pool := &WorkerPool{
		workerCount: workerCount,
		workQueue:   newQueue[WorkerTask](),
		errCh:       make(chan error, opts.ErrorBufferSize),
		cancel:      make(chan struct{}),
		ctx:         context.Background(),
		opts:        opts,
	}

	pool.start()
	return pool
}

// WithContext sets the context for the worker pool
func (p *WorkerPool) WithContext(ctx context.Context) *WorkerPool {
	p.ctx = ctx
	return p
}

// WithFastFail enables fast fail mode
func (p *WorkerPool) WithFastFail() *WorkerPool {
	p.opts.FastFail = true
	return p
}

// start initializes the worker pool
func (p *WorkerPool) start() {
	// Start workers
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// Start error handler
	go p.errorHandler()
}

// worker processes tasks from the work queue
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for task := range p.workQueue.Receive() {
		task.execute()
	}
}

// errorHandler handles errors and manages fast fail behavior
func (p *WorkerPool) errorHandler() {
	for err := range p.errCh {
		if p.opts.FastFail {
			p.once.Do(func() {
				close(p.cancel)
			})
		}

		if err != nil {
			p.mu.Lock()
			p.errs = append(p.errs, err)
			p.mu.Unlock()
		}
	}
}

// Submit submits a task to the worker pool
func (p *WorkerPool) Submit(task WorkerTask) error {
	if atomic.LoadInt64(&p.closed) == 1 {
		return errors.New(ErrMsgWorkerIsClosed)
	}

	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case <-task.ctx.Done():
		return task.ctx.Err()
	default:
		p.workQueue.Send(task)
	}

	return nil
}

// ReportError reports an error to the error handler
func (p *WorkerPool) ReportError(err error) {
	if err != nil {
		select {
		case p.errCh <- err:
		default:
			// Error channel is full, could log this
		}
	}
}

// IsCancelled returns true if the pool has been canceled due to fast fail
func (p *WorkerPool) IsCancelled() bool {
	select {
	case <-p.cancel:
		return true
	default:
		return false
	}
}

// CancelSignal returns the cancellation channel
func (p *WorkerPool) CancelSignal() <-chan struct{} {
	return p.cancel
}

// close closes the worker pool
func (p *WorkerPool) close() error {
	if !atomic.CompareAndSwapInt64(&p.closed, 0, 1) {
		return errors.New("worker pool already closed")
	}

	p.closeOnce.Do(func() {
		p.workQueue.Close()
	})

	return nil
}

// Wait waits for all workers to finish
func (p *WorkerPool) Wait() error {
	p.close()
	p.wg.Wait()
	p.close2Once.Do(func() {
		close(p.errCh)
	})

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.errs) > 0 {
		return p.errs[0]
	}

	return nil
}

// Errors returns all errors encountered
func (p *WorkerPool) Errors() []error {
	p.mu.Lock()
	defer p.mu.Unlock()

	res := make([]error, len(p.errs))
	copy(res, p.errs)
	return res
}

// IsActive returns true if the pool is still accepting tasks
func (p *WorkerPool) IsActive() bool {
	return atomic.LoadInt64(&p.closed) == 0
}

// Stats return basic statistics about the worker pool
func (p *WorkerPool) Stats() Stats {
	p.mu.Lock()
	errorCount := len(p.errs)
	p.mu.Unlock()

	return Stats{
		WorkerCount: p.workerCount,
		ErrorCount:  errorCount,
		IsActive:    p.IsActive(),
		IsCancelled: p.IsCancelled(),
	}
}

type Stats struct {
	WorkerCount int
	ErrorCount  int
	IsActive    bool
	IsCancelled bool
}
