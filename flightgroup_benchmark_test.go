package flightgroup

import (
	"context"
	"sync"
	"testing"
	"time"
)

// go test -bench=. -count=5 -benchmem

var (
	runtimes = 1000000
	ctx      = context.Background()
	testdata = make([]int, runtimes)
)

func BenchmarkGoroutines(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var (
			reader  = &mockReader{data: testdata}
			handler = &mockHandler{limit: len(testdata), sleep: time.Microsecond}
		)

		var wg sync.WaitGroup
		for {
			i, err := reader.Read()
			if err != nil {
				break
			}
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				handler.Handle(ctx, i)
			}(i)
		}

		wg.Wait()
	}
}

func BenchmarkFlightGroup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var (
			reader  = &mockReader{data: testdata}
			handler = &mockHandler{limit: len(testdata), sleep: time.Microsecond}
			group   = FlightGroup[int](ctx, reader, HandleFunc[int](handler.Handle), Timeout(1*time.Second), PoolSize(10000))
		)
		for err := range group.ErrChan {
			if err != nil {
				go group.Close()
				break
			}
		}
	}
}
