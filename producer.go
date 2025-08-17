package async

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type TaskResult[T any] struct {
	Value T
	Error error
}

type ProducerOpts struct {
	FastFail      bool
	ResultBuffer  int
	WorkQueueSize int
}

type Producer[T any] struct {
	pool       *WorkerPool
	resultChan Queue[TaskResult[T]]
	opts       ProducerOpts
	once       sync.Once
}

func NewProducer[T any](workerCount int) *Producer[T] {
	return NewProducerWithOpts[T](workerCount, ProducerOpts{
		ResultBuffer:  workerCount * 2,
		WorkQueueSize: workerCount * 2,
	})
}

func NewProducerWithOpts[T any](workerCount int, opts ProducerOpts) *Producer[T] {
	if opts.ResultBuffer <= 0 {
		opts.ResultBuffer = workerCount * 2
	}

	if opts.WorkQueueSize <= 0 {
		opts.WorkQueueSize = workerCount * 2
	}

	poolOpts := WorkerPoolOpts{
		FastFail:        opts.FastFail,
		WorkQueueSize:   opts.WorkQueueSize,
		ErrorBufferSize: workerCount * 2,
	}

	return &Producer[T]{
		pool:       NewWorkerPool(workerCount, poolOpts),
		resultChan: newQueue[TaskResult[T]](),
		opts:       opts,
	}
}

func (p *Producer[T]) WithContext(ctx context.Context) *Producer[T] {
	p.pool.WithContext(ctx)
	return p
}

func (p *Producer[T]) WithFastFail() *Producer[T] {
	p.opts.FastFail = true
	p.pool.WithFastFail()
	return p
}

// Submit submits a task to the worker pool and streams the result to the producer
func Submit[In any, Out any](ctx context.Context, p *Producer[Out], executor Activity[In, Out], req In) error {
	task := WorkerTask{
		ctx: ctx,
		execute: func() {
			// check for cancellation
			select {
			case <-ctx.Done():
				p.sendResult(TaskResult[Out]{Error: ctx.Err()})
				return
			case <-p.pool.CancelSignal():
				p.sendResult(TaskResult[Out]{Error: errors.New(ErrMsgTerminatedDueToFastFail)})
				return
			default:
			}

			// execute the task
			out, err := executor(ctx, req)

			// send result
			p.sendResult(TaskResult[Out]{
				Value: out,
				Error: err,
			})

			// report error for fast fail
			if err != nil {
				p.pool.ReportError(err)
			}
		},
	}

	return p.pool.Submit(task)
}

func (p *Producer[T]) sendResult(result TaskResult[T]) {
	p.resultChan.Send(result)
}

func (p *Producer[T]) Receive() <-chan TaskResult[T] {
	p.once.Do(func() {
		go func() {
			_ = p.Wait()
		}()
	})

	return p.resultChan.Receive()
}

func (p *Producer[T]) Close() error {
	return p.pool.close()
}

func (p *Producer[T]) Wait() error {
	err := p.pool.Wait()
	p.resultChan.Close()
	fmt.Println("closing producer")
	return err
}

func (p *Producer[T]) Errors() []error {
	return p.pool.Errors()
}

func (p *Producer[T]) IsActive() bool {
	return p.pool.IsActive()
}

func (p *Producer[T]) Stats() Stats {
	return p.pool.Stats()
}
