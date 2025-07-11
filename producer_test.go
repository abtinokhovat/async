package async_test

import (
	"context"
	"fmt"
	"github.com/abtinokhovat/async"
	"sync/atomic"
	"testing"
	"time"
)

func TestProducer_Submit(t *testing.T) {
	const (
		workerCount = 3
		n           = 30
		wait        = 300 * time.Millisecond
	)

	prod := async.NewProducer[int](workerCount)
	ctx := context.Background()

	var sum atomic.Int32

	process1 := func(ctx context.Context, num int) (int, error) {
		fmt.Printf("process 1: %d\n", num)
		time.Sleep(wait)
		sum.Add(int32(num))
		return num, nil
	}

	for i := 0; i < n; i++ {
		async.Submit(ctx, prod, process1, i)
	}

	fmt.Println("Started Listening")
	for out := range prod.Receive() {
		fmt.Println(out.Value)
	}
}
