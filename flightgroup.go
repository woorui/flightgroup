package flightgroup

import (
	"context"
	"errors"
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
	Handle(ctx context.Context, data T) error
}

// The HandleFunc is an adapter to allow the use of ordinary functions
// as a Handler. HandleFunc(fn) is a Handler that calls f.
type HandleFunc[T any] func(context.Context, T) error

// Handle calls HandleFunc itself.
func (f HandleFunc[T]) Handle(ctx context.Context, data T) error { return f(ctx, data) }

// ErrFlightGroupClosed be returned after group close.
var ErrFlightGroupClosed = errors.New("flightgroup: group has closed")

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

	go group.consume(group.read(ctx))

	return group
}

// read reads data from stream and sends them to a chan.
func (g *Group[T]) read(ctx context.Context) chan T {
	datch := make(chan T)

	g.flight.Add(1)
	go func() {
		defer func() {
			close(datch)
			g.flight.Done()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-g.done:
				return
			default:
			}
			data, err := g.reader.Read()
			if err != nil {
				g.ErrChan <- err
				continue
			}
			datch <- data
		}
	}()

	return datch
}

// consume consumes data from read reads channel.
func (g *Group[T]) consume(datch chan T) {
	size := g.options.poolSize

	g.flight.Add(size)

	for i := 0; i < size; i++ {
		go func() {
			defer g.flight.Done()
			for data := range datch {
				g.handleWithTimeout(data)
			}
		}()
	}
}

func (g *Group[T]) handleWithTimeout(data T) {
	// TODO: this ctx loss Value from ctx passed by externals.
	ctx := context.Background()
	if g.options.timeout != 0 {
		ctx, cancel := context.WithTimeout(ctx, g.options.timeout)
		defer cancel()

		if err := g.handler.Handle(ctx, data); err != nil {
			g.ErrChan <- err
		}
	}
}

// Close close the group, It blocked until all handlers return.
//
// Closing multiple times returns ErrFlightGroupClosed.
func (g *Group[T]) Close() error {
	if g.closed.Load() {
		return ErrFlightGroupClosed
	}
	g.closed.Store(true)

	close(g.done)

	g.flight.Wait()

	close(g.ErrChan)

	return nil
}

type Option func(o *options)

type options struct {
	timeout  time.Duration
	poolSize int
}

var defaultOptions = &options{
	timeout:  0,
	poolSize: 64,
}

// Timeout is timeout for handler.
func Timeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

// PoolSize defines how many goroutine to play handler.
func PoolSize(size uint32) Option {
	return func(o *options) {
		o.poolSize = int(size)
	}
}
