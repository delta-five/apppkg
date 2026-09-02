package closer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseManager_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*closeManager)
		fn      CloseWithContextErrorFunc
		wantErr string
	}{
		{
			// nil-функция отклоняется.
			name:    "nil function",
			fn:      nil,
			wantErr: "close function is nil",
		},
		{
			name: "success",
			fn:   func(context.Context) error { return nil },
		},
		{
			// После остановки регистрация запрещена.
			name: "already stopped",
			setup: func(b *closeManager) {
				b.stopCalled = true
			},
			fn:      func(context.Context) error { return nil },
			wantErr: "close already called",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange: менеджер с опциональной предустановкой состояния.
			b := &closeManager{}
			if tt.setup != nil {
				tt.setup(b)
			}

			// Act: регистрируем closer.
			err := b.add("test", tt.fn)

			// Assert: ошибка при невалидном вводе.
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			// Assert: успешная регистрация добавляет ровно один closer.
			require.NoError(t, err)
			require.Len(t, b.closers, 1)
			assert.Equal(t, "test", b.closers[0].name)
		})
	}
}

func TestCloseManager_Stop_ReverseOrder(t *testing.T) {
	t.Parallel()

	// Arrange: менеджер, готовый к остановке (cancel и таймаут заданы).
	b := &closeManager{
		cancel:      func() {},
		stopTimeout: time.Second,
	}

	// Регистрируем три closer'а и фиксируем порядок их вызова.
	var order []string
	require.NoError(t, b.add("first", func(context.Context) error {
		order = append(order, "first")
		return nil
	}))
	require.NoError(t, b.add("second", func(context.Context) error {
		order = append(order, "second")
		return nil
	}))
	require.NoError(t, b.add("third", func(context.Context) error {
		order = append(order, "third")
		return nil
	}))

	// Act: останавливаем.
	err := b.stop()

	// Assert: closers выполняются в порядке, обратном регистрации.
	require.NoError(t, err)
	assert.Equal(t, []string{"third", "second", "first"}, order)
}

func TestCloseManager_Stop_ContinuesOnError(t *testing.T) {
	t.Parallel()

	b := &closeManager{
		cancel:      func() {},
		stopTimeout: time.Second,
	}
	// Ожидаемая ошибка одного из closer'ов.
	errBoom := errors.New("boom")

	// Регистрируем два closer'а: один падает с ошибкой, другой — нет.
	var order []string
	require.NoError(t, b.add("ok", func(context.Context) error {
		order = append(order, "ok")
		return nil
	}))
	require.NoError(t, b.add("failing", func(context.Context) error {
		order = append(order, "failing")
		return errBoom
	}))

	// Act: ошибка одного closer'а не прерывает остановку остальных.
	err := b.stop()

	// Assert: оба closer'а вызваны, ошибка проброшена в результат.
	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, []string{"failing", "ok"}, order)
}

func TestCloseManager_Stop_AlreadyStopped(t *testing.T) {
	t.Parallel()

	// Arrange: менеджер уже помечен как остановленный.
	b := &closeManager{
		cancel:      func() {},
		stopTimeout: time.Second,
		stopCalled:  true,
	}

	// Act: повторный вызов stop.
	err := b.stop()

	// Assert: повторная остановка отклоняется.
	require.Error(t, err)
	assert.EqualError(t, err, "already stopped")
}

func TestCloseManager_Stop_Timeout(t *testing.T) {
	t.Parallel()

	// Arrange: нулевой таймаут приводит к мгновенному истечению времени.
	b := &closeManager{
		cancel:      func() {},
		stopTimeout: 0,
	}

	// Act: останавливаем без выделенного времени.
	err := b.stop()

	// Assert: контекст уже истёк — DeadlineExceeded.
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCloseManager_Init(t *testing.T) {
	t.Parallel()

	b := &closeManager{}

	// Act: первичная инициализация с валидным таймаутом.
	ctx, cancel, err := b.init(context.Background(), time.Second)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	require.NotNil(t, cancel)

	// Повторная инициализация запрещена.
	_, _, err = b.init(context.Background(), time.Second)
	require.EqualError(t, err, "already initialized")

	cancel()
}

func TestFuncToCloseFn(t *testing.T) {
	t.Parallel()

	// Arrange: фиксируем факт вызова исходной функции.
	called := false
	fn := FuncToCloseFn(func() {
		called = true
	})

	// Act: вызываем обёртку с произвольным контекстом.
	err := fn(context.Background())

	// Assert: функция вызвана, ошибки нет.
	require.NoError(t, err)
	assert.True(t, called)
}

func TestFuncErrorToCloseFn(t *testing.T) {
	t.Parallel()

	// Arrange: исходная функция возвращает ошибку.
	wantErr := errors.New("some error")
	fn := FuncErrorToCloseFn(func() error {
		return wantErr
	})

	// Act+Assert: обёртка пробрасывает ошибку без изменений.
	err := fn(context.Background())
	assert.ErrorIs(t, err, wantErr)
}

func TestAdd_NilFunc(t *testing.T) {
	// Act: регистрируем nil-функцию.
	err := Add("test", nil)
	// Assert: nil отклоняется.
	require.Error(t, err)
	assert.EqualError(t, err, "close function is nil")
}

func TestAddWithError_NilFunc(t *testing.T) {
	err := AddWithError("test", nil)
	require.Error(t, err)
	assert.EqualError(t, err, "close function is nil")
}

func TestAddWithContextError_NilFunc(t *testing.T) {
	err := AddWithContextError("test", nil)
	require.Error(t, err)
	assert.EqualError(t, err, "close function is nil")
}

func TestAdd_RegistersCloser(t *testing.T) {
	// Act: регистрируем валидную функцию.
	err := Add("test", func() {})
	// Assert: ошибок нет.
	require.NoError(t, err)
}

func TestAddCloser(t *testing.T) {
	// Act: регистрируем объект, реализующий Closer.
	err := AddCloser("test", fakeCloser{})
	require.NoError(t, err)
}

func TestAddErrorCloser(t *testing.T) {
	// Act: регистрируем объект, реализующий ErrorCloser.
	err := AddErrorCloser("test", fakeErrorCloser{})
	require.NoError(t, err)
}

func TestAddContextErrorCloser(t *testing.T) {
	// Act: регистрируем объект, реализующий ContextErrorCloser.
	err := AddContextErrorCloser("test", fakeContextErrorCloser{})
	require.NoError(t, err)
}

type fakeCloser struct{}

func (fakeCloser) Close() {}

type fakeErrorCloser struct{}

func (fakeErrorCloser) Close() error { return nil }

type fakeContextErrorCloser struct{}

func (fakeContextErrorCloser) Close(context.Context) error { return nil }
