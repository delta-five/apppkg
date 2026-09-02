package closer

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/delta-five/apppkg/logger"
)

type closeData struct {
	name string
	fn   CloseWithContextErrorFunc
}

type closeManager struct {
	mx          sync.Mutex
	closers     []closeData
	stopTimeout time.Duration
	cancel      context.CancelFunc
	stopCalled  bool
}

func (b *closeManager) init(ctx context.Context, stopTimeout time.Duration) (context.Context, context.CancelFunc, error) {
	if stopTimeout <= 0 {
		return nil, nil, errors.New("timed out not valid")
	}
	b.mx.Lock()
	defer b.mx.Unlock()
	if b.cancel != nil {
		return nil, nil, errors.New("already initialized")
	}
	ctx, cancel := context.WithCancel(ctx)

	b.stopTimeout = stopTimeout
	b.cancel = cancel

	go func(ctx context.Context) {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		defer signal.Stop(quit)

		select {
		case <-quit:
			logger.Infof(ctx, "a signal has been received to complete the work...")
		case <-ctx.Done():
			logger.Warnf(ctx, "closer context is done")
		}

		_ = b.stop()
	}(ctx)

	return ctx, cancel, nil
}

// stop принудительно запускает остановку: помечает closer как закрытый, отменяет контекст
// и последовательно выполняет зарегистрированные closers в обратном порядке.
// Для closers задается общее ограничение на выполнение через новый контекст.
func (b *closeManager) stop() error {
	b.mx.Lock()
	if b.cancel == nil {
		return errors.New("closer not initialized")
	}
	if b.stopCalled {
		b.mx.Unlock()
		return errors.New("already stopped")
	}
	b.stopCalled = true
	b.mx.Unlock()

	b.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), b.stopTimeout)
	defer cancel()

	errs := make([]error, 0, len(b.closers))
	for _, stopper := range slices.Backward(b.closers) {
		err := stopper.fn(ctx)
		if err != nil {
			errs = append(errs, err)
			logger.Errorf(ctx, "closing of '%s' is failed: %v", stopper.name, err)
		} else {
			logger.Infof(ctx, "'%s' closed successfully", stopper.name)
		}
	}

	errs = append(errs, ctx.Err())

	return errors.Join(errs...)
}

func (b *closeManager) add(name string, fn CloseWithContextErrorFunc) error {
	if fn == nil {
		return errors.New("close function is nil")
	}

	b.mx.Lock()
	defer b.mx.Unlock()

	if b.stopCalled {
		return errors.New("close already called")
	}

	b.closers = append(b.closers, closeData{
		name: name,
		fn:   fn,
	})

	return nil
}
