package bunch

import (
	"context"
	"sync"
)

type mockDriver struct {
	mu     sync.Mutex
	source []int
	to     map[int]int
	errstr string
}

func newMockDriver(ints ...int) *mockDriver {
	return &mockDriver{
		mu:     sync.Mutex{},
		source: ints,
		to:     map[int]int{},
	}
}

func (c *mockDriver) Read(ctx context.Context) (<-chan int, error) {
	ch := make(chan int)
	go func() {
		for _, v := range c.source {
			ch <- v
		}
		close(ch)
	}()

	return ch, nil
}

func (c *mockDriver) Handle(ctx context.Context, n int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.to[n]++

	return nil
}

func (c *mockDriver) HandleErr(e error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errstr += e.Error()
}
