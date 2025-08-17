package async_test

import (
	"context"
	"fmt"
	"github.com/abtinokhovat/async"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestProducer_Receive(t *testing.T) {
	const (
		workerCount = 5
		n           = 30
		wait        = 300 * time.Millisecond
	)

	prod := async.NewProducer[int32](workerCount)
	ctx := context.Background()

	var sum atomic.Int32

	process1 := func(ctx context.Context, num int32) (int32, error) {
		time.Sleep(wait)
		sum.Add(num)
		return num, nil
	}

	go func() {
		for i := 0; i < n; i++ {
			async.Submit(ctx, prod, process1, int32(i))
		}
	}()

	var sumResult int32 = 0
	for out := range prod.Receive() {
		sumResult += out.Value
	}

	if sumResult != sum.Load() {
		t.Errorf("sum result should be %d, but %d", sumResult, sum.Load())
	}
}

func TestProducer_ReceivePiping(t *testing.T) {
	const (
		workerCount1 = 5
		workerCount2 = 3

		wait1 = 300 * time.Millisecond
		wait2 = 150 * time.Millisecond

		n = 30
	)

	prod := async.NewProducer[int32](workerCount1)
	prod2 := async.NewProducer[string](workerCount2)
	ctx := context.Background()

	var sum atomic.Int32

	process1 := func(ctx context.Context, num int32) (int32, error) {
		time.Sleep(wait1)
		sum.Add(num)
		fmt.Printf("process 1 %d\n", num)
		return num, nil
	}

	process2 := func(ctx context.Context, num int32) (string, error) {
		time.Sleep(wait2)
		fmt.Printf("process 2 %d\n", num)
		return strconv.Itoa(int(num)), nil
	}

	for i := 0; i < n; i++ {
		async.Submit(ctx, prod, process1, int32(i))
	}

	async.Pipe(ctx, prod, prod2, process2)

	for res := range prod2.Receive() {
		fmt.Println(res.Value)
	}

	prod2.Wait()
}
