package closer

import (
	"context"
	"errors"
	"time"
)

var manager closeManager

// Init инициализирует глобальный closer. Должен быть вызван один раз при старте
// приложения до вызова Add/AddWithContext.
func Init(ctx context.Context, closeTimeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel, err := manager.init(ctx, closeTimeout)
	if err != nil {
		panic(err)
	}
	return ctx, cancel
}

// CloseAll запускает остановку всех зарегистрированных функций
// в порядке, обратном их регистрации.
func CloseAll() error {
	return manager.stop()
}

func Done() <-chan struct{} {
	return manager.done()
}

// CloseFunc — функция остановки без контекста.
type (
	CloseFunc func()

	// CloseFuncWithError — функция остановки, возвращающая ошибку.
	CloseFuncWithError func() error

	// CloseWithContextErrorFunc — функция остановки, принимающая контекст
	// с ограничением по времени и возвращающая ошибку.
	CloseWithContextErrorFunc func(context.Context) error

	// Closer — интерфейс объекта с методом остановки Close().
	Closer interface {
		Close()
	}

	// ErrorCloser — интерфейс объекта с методом остановки Close() error.
	ErrorCloser interface {
		Close() error
	}

	// ContextErrorCloser — интерфейс объекта с методом остановки
	// Close(ctx context.Context) error.
	ContextErrorCloser interface {
		Close(ctx context.Context) error
	}
)

// FuncErrorToCloseFn преобразует CloseFuncWithError (func() error)
// в CloseWithContextErrorFunc, игнорируя переданный контекст.
func FuncErrorToCloseFn(fn CloseFuncWithError) CloseWithContextErrorFunc {
	return func(_ context.Context) error {
		return fn()
	}
}

// FuncToCloseFn создает обертку для вызова `func()` как функцию остановки
func FuncToCloseFn(fn func()) CloseWithContextErrorFunc {
	return func(_ context.Context) error {
		fn()
		return nil
	}
}

// Add регистрирует функцию остановки
func Add(name string, closeFunc CloseFunc) error {
	if closeFunc == nil {
		return errors.New("close function is nil")
	}
	return manager.add(name, FuncToCloseFn(closeFunc))
}

// AddWithError регистрирует функцию остановки
func AddWithError(name string, closeFunc CloseFuncWithError) error {
	if closeFunc == nil {
		return errors.New("close function is nil")
	}
	return manager.add(name, FuncErrorToCloseFn(closeFunc))
}

// AddWithContextError регистрирует функцию остановки, принимающую контекст с таймаутом.
func AddWithContextError(name string, closeFunc CloseWithContextErrorFunc) error {
	if closeFunc == nil {
		return errors.New("close function is nil")
	}
	return manager.add(name, closeFunc)
}

// AddCloser регистрирует объект, реализующий интерфейс Closer.
func AddCloser(name string, c Closer) error {
	return Add(name, c.Close)
}

// AddErrorCloser регистрирует объект, реализующий интерфейс ErrorCloser.
func AddErrorCloser(name string, c ErrorCloser) error {
	return AddWithError(name, c.Close)
}

// AddContextErrorCloser регистрирует объект, реализующий интерфейс ContextErrorCloser.
func AddContextErrorCloser(name string, c ContextErrorCloser) error {
	return AddWithContextError(name, c.Close)
}
