package async_test

import (
	"context"
	"fmt"
	"github.com/abtinokhovat/async"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecute_Wait(t *testing.T) {
	const (
		workerCount = 5
		n           = 30
		wait        = 100 * time.Millisecond
	)

	grp := async.NewWorkerGroup(workerCount)
	ctx := context.Background()

	var sum atomic.Int32

	process1 := func(ctx context.Context, num int) (int, error) {
		time.Sleep(wait)
		sum.Add(int32(num))
		return num, nil
	}

	start := time.Now()
	for i := 0; i < n; i++ {
		async.Execute(ctx, grp, process1, i)
	}

	_ = grp.Wait()
	elapsed := time.Since(start)

	var totalCount int32 = n * (n - 1) / 2
	if sum.Load() != totalCount {
		t.Errorf("expected sum to be 435, got %d", sum.Load())
	}

	delay := 50 * time.Millisecond
	var totalElapsed = n * wait / workerCount
	if elapsed < totalElapsed-delay || elapsed > totalElapsed+delay {
		t.Errorf("expected execution time to be around %v+%v, got %v", totalElapsed, delay, elapsed)
	}
}

func TestExecute_WaitError(t *testing.T) {
	const (
		workerCount      = 5
		n                = 30
		wait             = 100 * time.Millisecond
		processWithError = 5
	)

	grp := async.NewWorkerGroup(workerCount)
	ctx := context.Background()

	process1 := func(ctx context.Context, num int) (int, error) {
		time.Sleep(wait)
		if num == processWithError {
			return 0, fmt.Errorf("random error for %d", num)
		}

		fmt.Printf("[%v] processing %d\n", time.Now().Format(time.RFC3339), num)
		return num, nil
	}

	for i := 0; i < n; i++ {
		async.Execute(ctx, grp, process1, i)
	}

	err := grp.Wait()
	if err == nil {
		t.Error("expected error, got nil")
	}
	fmt.Printf("error: %v\n", err)

	errs := grp.Errors()
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	const (
		workerCount = 5
		n           = 30
		wait        = 100 * time.Millisecond
	)

	grp := async.NewWorkerGroup(workerCount)
	ctx, cancel := context.WithCancel(context.Background())

	process1 := func(ctx context.Context, num int) (int, error) {
		time.Sleep(wait)
		fmt.Printf("[%v] processing %d\n", time.Now().Format(time.RFC3339), num)
		return num, nil
	}

	// randomly cancel other task contexts
	go func() {
		time.Sleep(wait * 2)
		cancel()
	}()

	for i := 0; i < n; i++ {
		async.Execute(ctx, grp, process1, i)
	}

	err := grp.Wait()
	if err == nil {
		t.Error("expected error, got nil")
	}
	fmt.Printf("error: %v\n", err)

	errs := grp.Errors()
	fmt.Println(len(errs))
}

func TestExecute_FastFail(t *testing.T) {
	const (
		workerCount      = 5
		n                = 30
		wait             = 100 * time.Millisecond
		processWithError = 2
	)

	grp := async.NewWorkerGroup(workerCount).WithFastFail()
	ctx := context.Background()

	process1 := func(ctx context.Context, num int) (int, error) {
		time.Sleep(wait)
		if num == processWithError {
			return 0, fmt.Errorf("random error for %d", num)
		}

		fmt.Printf("[%v] processing %d\n", time.Now().Format(time.RFC3339), num)
		return num, nil
	}

	for i := 0; i < n; i++ {
		async.Execute(ctx, grp, process1, i)
	}

	err := grp.Wait()
	if err == nil {
		t.Error("expected error, got nil")
	}
	fmt.Printf("error: %v\n", err)

	errs := grp.Errors()
	if len(errs) <= 1 {
		t.Errorf("expected all tasks after number 5 to, but this did not happened")
	}
}
