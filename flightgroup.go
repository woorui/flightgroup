package flightgroup

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Reader is the interface that wraps the Read method.

// Reader reads T from underlying and return it.
type Reader[T any] interface {
	// Read can read something,
	// Put Read in for-loop for getting continuous datas.
	Read() (T, error)
}

// Handler defines how to handle data from Reader.
type Handler[T any] interface {
	// Handle handles the data.
	Handle(data T) error
}

// The HandleFunc is an adapter to allow the use of ordinary functions
// as a Handler. HandleFunc(fn) is a Handler that calls f.
type HandleFunc[T any] func(T) error

// Handle calls HandleFunc itself.
func (f HandleFunc[T]) Handle(data T) error { return f(data) }

var (
	// ErrClosed be returned after group close.
	ErrClosed = errors.New("flightgroup closed")
	// ErrReadTimeout be returned if read timeout.
	ErrReadTimeout = errors.New("flightgroup: read timeout")
)

// ErrReadTimeout be returned if read timeout.
type ErrHandleTimeout[T any] struct{ data T }

// NewErrHandleTimeout returns ErrReadTimeout with data be handled.
func NewErrHandleTimeout[T any](data T) ErrHandleTimeout[T] { return ErrHandleTimeout[T]{data} }

// Error returns error message.
func (e ErrHandleTimeout[T]) Error() string { return "flightgroup: handle timeout" }

// Group holds a reader and a handler,
// group reads data from reader, and pass in handler to handle it.
// each of handler running in a goroutinue,
// the group can only be close when all handler is finished.
type Group[T any] struct {
	flight sync.WaitGroup

	options *options
	closed  atomic.Bool

	reader  Reader[T]
	handler Handler[T]

	done chan struct{}

	// ErrChan collects errors return by Reader and Handler,
	// group don't help you to handle this errors,
	// you must listens this channel to handle then.
	ErrChan chan error
}

// FlightGroup returns Group to manage handler,
// you maybe select the group.Errchan to handle errors from Reader and Hanler.
func FlightGroup[T any](ctx context.Context, reader Reader[T], handler Handler[T], opts ...Option) *Group[T] {
	group := &Group[T]{
		reader:  reader,
		handler: handler,
		options: defaultOptions,
		ErrChan: make(chan error, 100),
		done:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(group.options)
	}

	go group.consume(
		group.options.cap,
		group.reading(ctx),
	)

	return group
}

type pipe[T any] struct {
	data T
	err  error
}

// reading continuous reads data from stream and sends them to a group woker.
func (g *Group[T]) reading(ctx context.Context) chan pipe[T] {
	ch := make(chan pipe[T])

	g.flight.Add(1)
	go func() {
		defer g.flight.Done()

		for {
			var t <-chan time.Time
			if g.options.readTimeout != 0 {
				t = time.After(g.options.readTimeout)
			}
			select {
			case <-ctx.Done():
				return
			case <-g.done:
				return
			case <-t:
				g.ErrChan <- ErrReadTimeout
			case ch <- g.read():
			}
		}
	}()

	return ch
}

// consume consumes data from read reads channel using cap goroutinues.
func (g *Group[T]) consume(cap int, ch chan pipe[T]) {
	g.flight.Add(cap)

	for i := 0; i < cap; i++ {
		go func() {
			defer g.flight.Done()

			for {
				select {
				case <-g.done:
					return
				case out := <-ch:
					if out.err != nil {
						g.ErrChan <- out.err
					} else {
						if err := g.handle(out.data); err != nil {
							g.ErrChan <- err
						}
					}
				}
			}
		}()
	}
}

func (g *Group[T]) read() pipe[T] {
	data, err := g.reader.Read()

	return pipe[T]{data, err}
}

func (g *Group[T]) handle(data T) error {
	ch := make(chan error)

	go func() {
		ch <- g.handler.Handle(data)
	}()

	var t <-chan time.Time
	if g.options.handleTimeout != 0 {
		t = time.After(g.options.handleTimeout)
	}

	select {
	case err := <-ch:
		return err
	case <-t:
		return NewErrHandleTimeout(data)
	}
}

// Close close the group, It blocked until all handlers return.
//
// Closing multiple times returns ErrClosed.
func (g *Group[T]) Close() error {
	if g.closed.Load() {
		return ErrClosed
	}
	g.closed.Store(true)

	close(g.done)

	g.flight.Wait()

	close(g.ErrChan)

	return nil
}

type Option func(o *options)

type options struct {
	cap           int
	handleTimeout time.Duration
	readTimeout   time.Duration
}

var defaultOptions = &options{
	readTimeout:   time.Second,
	handleTimeout: 10 * time.Second,
	cap:           runtime.GOMAXPROCS(-1) * 64,
}

// HandleTimeout is timereadout for handler.
func HandleTimeout(t time.Duration) Option {
	return func(o *options) {
		o.handleTimeout = t
	}
}

// ReadTimeout is timereadout for reader.
func ReadTimeout(t time.Duration) Option {
	return func(o *options) {
		o.readTimeout = t
	}
}

// Cap defines how many gorreadoutine to play handler.
func Cap(cap uint32) Option {
	return func(o *options) {
		o.cap = int(cap)
	}
}
