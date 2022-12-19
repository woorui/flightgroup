package flightgroup

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) {
	ctx := context.Background()

	testdata := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	var (
		reader  = &mockReader{data: testdata}
		handler = &mockHandler{limit: len(testdata), sleep: time.Millisecond}
		group   = FlightGroup[int](ctx, reader, HandleFunc[int](handler.Handle), Cap(1), ReadTimeout(time.Second), HandleTimeout(time.Microsecond))
	)

	// close later.
	time.AfterFunc(time.Millisecond, func() {
		t.Logf(
			"Receive Close sign, The test logic is pipe reader to handler: reader %v, handler: %v, initReader: %v",
			reader.result(),
			handler.result(),
			testdata,
		)
		group.Close()
	})

	for err := range group.ErrChan {
		t.Logf("listened to the error from ErrChan, err: %v", err)
	}

	if len(testdata) != len(reader.result())+len(handler.result()) {
		t.Fatalf(
			"test fail, reader %v, handler: %v, initReader: %v",
			reader.result(),
			handler.result(),
			testdata,
		)
	}

	t.Run("close multiple times", func(t *testing.T) {
		err := group.Close()
		if err != ErrClosed {
			t.Fatalf(
				"close group multiple times get: %v, want: %v",
				err,
				ErrClosed,
			)
		}
	})
}

type mockReader struct {
	mu    sync.Mutex
	data  []int
	sleep time.Duration
}

func (r *mockReader) Read() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	time.Sleep(r.sleep)

	if len(r.data) > 0 {
		item := r.data[0]
		r.data = r.data[1:]
		return item, nil
	}
	// time.Sleep(100 * time.Second)
	return 0, io.EOF
}

func (r *mockReader) result() []int {
	r.mu.Lock()
	defer r.mu.Unlock()

	res := make([]int, len(r.data))
	copy(res, r.data)
	return res
}

func (r *mockReader) Close() error { return nil }

type mockHandler struct {
	mu    sync.Mutex
	limit int
	sleep time.Duration
	data  []int
	err   error
}

var errLimitReached = errors.New("handler: limit reached")

func (h *mockHandler) Handle(i int) error {
	if h.err != nil {
		return h.err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	time.Sleep(h.sleep)

	h.data = append(h.data, i)
	if len(h.data) == h.limit {
		return errLimitReached
	}

	return nil
}

func (h *mockHandler) result() []int {
	h.mu.Lock()
	defer h.mu.Unlock()

	res := make([]int, len(h.data))
	copy(res, h.data)
	return res
}
