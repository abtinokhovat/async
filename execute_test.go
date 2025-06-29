package async_test

import (
	"async"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestExecuteWithBarrier(t *testing.T) {
	activity := async.NewActivityWithReq(func(ctx context.Context, num int) error {
		time.Sleep(1 * time.Second)
		fmt.Printf("executed %d\n", num)
		return nil
	})

	grp := async.NewGroup(100, 100)

	ctx, cancel := context.WithCancel(context.Background())

	futures := make([]async.Future[async.Void], 100)
	for i := 0; i < 100; i++ {
		futures = append(futures, async.Execute(ctx, grp, i, activity))
	}

	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("cancel")
		cancel()
	}()

	grp.Wait()
}
