package structures

type Semaphore struct {
	Channel chan struct{}
}

func NewSemaphore(weight int) *Semaphore {
	return &Semaphore{Channel: make(chan struct{}, weight)}
}

func (s *Semaphore) Acquire() {
	s.Channel <- struct{}{}
}

func (s *Semaphore) Release() {
	<- s.Channel
}
