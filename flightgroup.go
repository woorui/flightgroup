package flightgroup

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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

// ErrFlightGroupClosed be returned after group close.
var ErrFlightGroupClosed = errors.New("flightgroup: group has closed")

// Group holds a reader and a handler,
// group reads data from reader, and pass in handler to handle it.
// each of handler running in a goroutinue,
// the group can only be close when all handler is finished.
type Group[T any] struct {
	flight sync.WaitGroup

	closed atomic.Bool

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
func FlightGroup[T any](ctx context.Context, reader Reader[T], handler Handler[T]) *Group[T] {
	group := &Group[T]{
		reader:  reader,
		handler: handler,
		ErrChan: make(chan error, 100),
		done:    make(chan struct{}),
	}

	group.run(ctx)

	return group
}

// run reads data from stream and handle them,
// it returns an error channel for handler returns.
func (g *Group[T]) run(ctx context.Context) {
	g.flight.Add(1)

	go func() {
		defer g.flight.Done()
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
			g.flight.Add(1)
			go func() {
				defer g.flight.Done()
				if err := g.handler.Handle(data); err != nil {
					g.ErrChan <- err
				}
			}()
		}

	}()
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
