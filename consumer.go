package async

import (
	"context"
	"fmt"
)

func Pipe[In any, Out any](
	ctx context.Context,
	in Producer2[TaskResult[In]],
	p *Producer[Out],
	activity Activity[In, Out],
) *Producer[Out] {
	go func() {
		fmt.Println("starting")
		for taskRes := range in.Receive() {
			if taskRes.Error != nil {
				p.sendResult(TaskResult[Out]{Error: taskRes.Error})
				continue
			}

			err := Submit(ctx, p, activity, taskRes.Value)
			if err != nil {
				p.sendResult(TaskResult[Out]{Error: err})
			}
		}
		fmt.Println("exiting")
	}()

	return p
}
