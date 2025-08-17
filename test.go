package async

import (
	"context"
	"sync"
)

func testProducer[In any, Out any](ctx context.Context, in <-chan In, activity Activity[In, Out]) <-chan Out {
	const workerCount = 5

	outCh := make(chan Out)
	errCh := make(chan error, 1)

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for batch := range in {
				select {
				case <-ctx.Done():
					return
				default:
				}

				out, err := activity(ctx, batch)

				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}

				outCh <- out
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(outCh)
	}()

	return outCh
}
