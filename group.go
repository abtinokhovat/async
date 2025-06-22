package async

import "sync"

type Group struct {
	wg      sync.WaitGroup
	errChan chan error
}

func NewGroup(buffer int) *Group {
	return &Group{
		wg:      sync.WaitGroup{},
		errChan: make(chan error, buffer),
	}
}

func (g *Group) Wait() error {
	g.wg.Wait()

	select {
	case errs := <-g.errChan:
		if errs != nil {
			return errs
		}
	default:
	}

	return nil
}
