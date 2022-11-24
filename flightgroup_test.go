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
		handler = &mockHandler{limit: len(testdata)}
		group   = FlightGroup[int](ctx, reader, HandleFunc[int](handler.Handle))
	)

	// close later.
	time.AfterFunc(time.Microsecond, func() {
		t.Logf(
			"The test logic is pipe reader to handler: reader %v, handler: %v, initReader: %v",
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
		if err != ErrFlightGroupClosed {
			t.Fatalf(
				"close group multiple times get: %v, want: %v",
				err,
				ErrFlightGroupClosed,
			)
		}
	})
}

type mockReader struct {
	mu   sync.Mutex
	data []int
}

func (r *mockReader) Read() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.data) > 0 {
		item := r.data[0]
		r.data = r.data[1:]
		return item, nil
	}
	return 0, io.EOF
}

func (h *mockReader) result() []int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.data
}

func (r *mockReader) Close() error { return nil }

type mockHandler struct {
	mu    sync.Mutex
	limit int
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

	h.data = append(h.data, i)
	if len(h.data) == h.limit {
		return errLimitReached
	}

	return nil
}

func (h *mockHandler) result() []int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.data
}
