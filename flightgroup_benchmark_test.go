package flightgroup

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
)

// go test -bench=. -count=5 -benchmem

var (
	runtimes = 1000000
	ctx      = context.Background()
	testdata = make([]int, runtimes)
)

func init() {
	for i := range testdata {
		testdata[i] = i
	}
}

func BenchmarkFlightGroup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var (
			reader  = &mockReader{data: testdata}
			handler = &mockHandler{limit: len(testdata), sleep: time.Microsecond}
			group   = FlightGroup[int](ctx, reader, HandleFunc[int](handler.Handle), ReadTimeout(1*time.Second), HandleTimeout(time.Second), Cap(10000))
		)
		for err := range group.ErrChan {
			if err != nil {
				go group.Close()
				break
			}
		}
	}
}

func BenchmarkNative(b *testing.B) {
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
				handler.Handle(i)
			}(i)
		}

		wg.Wait()
	}
}

func BenchmarkAnts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var (
			reader  = &mockReader{data: testdata}
			handler = &mockHandler{limit: len(testdata), sleep: time.Microsecond}
		)

		pool, _ := ants.NewPool(10000)
		defer pool.Release()

		var wg sync.WaitGroup
		for {
			i, err := reader.Read()
			if err != nil {
				break
			}
			wg.Add(1)

			pool.Submit(func() {
				defer wg.Done()
				handler.Handle(i)
			})
		}

		wg.Wait()
	}
}
