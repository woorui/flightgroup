package bunch

import (
	"context"
	"sync"
)

type mockActor struct {
	mu     sync.Mutex
	source []int
	to     map[int]int
	errstr string
}

func newMockActor(ints ...int) *mockActor {
	return &mockActor{
		mu:     sync.Mutex{},
		source: ints,
		to:     map[int]int{},
	}
}

func (c *mockActor) Read(ctx context.Context) (<-chan int, error) {
	ch := make(chan int)
	go func() {
		for _, v := range c.source {
			ch <- v
		}
		close(ch)
	}()

	return ch, nil
}

func (c *mockActor) Handle(ctx context.Context, n int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.to[n]++

	return nil
}

func (c *mockActor) HandleErr(e error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errstr += e.Error()
}
