package async

type Queue[T any] interface {
	Send(T)
	Receive() <-chan T
	Close()
}

type Producer2[T any] interface {
	Receive() <-chan T
}
