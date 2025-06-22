package async

import "context"

// ExecuteAll manage to execute the Activity with the Group and returns a future
func ExecuteAll[T any, Req any](ctx context.Context, grp *Group, req Req, executor Activity[T, Req]) *Future[T] {
	res := &Future[T]{
		resChan: make(chan T, 1),
		errChan: make(chan error, 1),
	}

	grp.wg.Add(1)

	go func() {
		defer grp.wg.Done()
		result, err := executor(ctx, req)
		if err != nil {
			res.errChan <- err
			grp.errChan <- err
			return
		}
		res.resChan <- result
	}()

	return res
}
