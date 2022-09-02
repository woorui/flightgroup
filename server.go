package bunch

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Actor defines how to play with server.
type Actor[T any] interface {
	// Read reads sth to channel.
	Read(context.Context) (<-chan T, error)
	// Handle Handle sth.
	Handle(context.Context, T) error
	// HandleErr defines how to handle error.
	HandleErr(error)
}

// ErrServerClosed represents a completed Shutdown
var ErrServerClosed = errors.New("trike: server closed")

// Server provides stage for actor playing.
type Server[T any] struct {
	options    *options
	appCtx     context.Context    // appCtx is used to control the lifecycle of the Server.
	cancelFunc context.CancelFunc // cancelFunc to signal the server should stop requesting messages.
	actor      Actor[T]
}

// NewServer returns a server.
func NewServer[T any](actor Actor[T], option ...Option) *Server[T] {
	options := defaultOptions

	for _, o := range option {
		o(options)
	}

	appCtx, appCancel := context.WithCancel(options.ctx)

	return &Server[T]{
		options:    options,
		appCtx:     appCtx,
		cancelFunc: appCancel,
		actor:      actor,
	}
}

// Serve() continue to listen until Shutdown is called on the Server.
func (srv *Server[T]) Serve() error {
	var wg sync.WaitGroup

	msgch, err := srv.actor.Read(srv.appCtx)
	if err != nil {
		return err
	}

	worker := make(chan struct{}, srv.options.handlerNum)

	for {
		select {
		case <-srv.appCtx.Done():
			goto END
		case msg, ok := <-msgch:
			if !ok {
				goto END
			}
			worker <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				var err error
				if srv.options.timeout > 0 {
					userCtx, cancel := context.WithTimeout(context.Background(), srv.options.timeout)
					defer cancel()

					err = srv.actor.Handle(userCtx, msg)
				} else {
					err = srv.actor.Handle(context.Background(), msg)
				}
				if err != nil {
					srv.actor.HandleErr(err)
				}
				<-worker
			}()
		}
	}

END:
	wg.Wait()
	close(worker)

	return ErrServerClosed
}

// Shutdown gracefully shuts down the Server by letting any messages in
// flight finish processing.
func (srv *Server[T]) Shutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-srv.appCtx.Done():
		return ErrServerClosed
	default:
	}

	srv.cancelFunc()

	return nil
}

type options struct {
	ctx        context.Context
	timeout    time.Duration
	handlerNum uint32
}

var defaultOptions = &options{
	ctx:        context.Background(),
	timeout:    0,
	handlerNum: 10,
}

type Option func(o *options)

// HandlerNum defines how many goroutine to handle it.
func Context(ctx context.Context) Option {
	return func(o *options) {
		o.ctx = ctx
	}
}

// Timeout is timeout for handling.
func Timeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

// HandlerNum defines how many goroutine to handle it.
func HandlerNum(size uint32) Option {
	return func(o *options) {
		o.handlerNum = size
	}
}
