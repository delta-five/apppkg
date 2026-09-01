package app_error_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/delta-five/apppkg/app_error"
)

func TestNew(t *testing.T) {
	t.Parallel()

	// Arrange: исходная техническая ошибка.
	inner := errors.New("db: connection refused")
	// Act: оборачиваем её в AppError с категорией Unavailable.
	err := app_error.New("сервис временно недоступен", inner, app_error.Unavailable)

	// Assert: ошибка создана.
	if err == nil {
		t.Fatal("New() вернул nil")
	}
	// Пользователю отдаётся безопасное сообщение.
	if err.Error() != "сервис временно недоступен" {
		t.Errorf("Error() = %q, want %q", err.Error(), "сервис временно недоступен")
	}
	// Категория сохраняется.
	if err.Kind() != app_error.Unavailable {
		t.Errorf("Kind() = %v, want %v", err.Kind(), app_error.Unavailable)
	}
	// Техническая ошибка доступна через цепочку errors.Is.
	if !errors.Is(err, inner) {
		t.Error("errors.Is не находит исходную ошибку")
	}
	// LogError возвращает техническую ошибку для логов.
	logErr := app_error.LogError(err)
	if !errors.Is(logErr, inner) {
		t.Errorf("LogError() = %v, want %v", logErr, inner)
	}
}

func TestNew_NilInnerError(t *testing.T) {
	t.Parallel()

	// Act: передаём nil в качестве технической ошибки.
	err := app_error.New("что-то пошло не так", nil, app_error.Internal)
	if err == nil {
		t.Fatal("New() вернул nil при err == nil")
	}
	// Пользовательское сообщение сохраняется.
	if err.Error() != "что-то пошло не так" {
		t.Errorf("Error() = %q, want %q", err.Error(), "что-то пошло не так")
	}
	// При nil техническая ошибка создаётся автоматически из сообщения.
	if err.LogError() == nil {
		t.Error("LogError() вернул nil, ожидалась автоматически созданная ошибка")
	}
}

func TestNew_ExistingAppError(t *testing.T) {
	t.Parallel()

	// Arrange: исходный AppError.
	inner := app_error.New("оригинал", errors.New("inner"), app_error.NotFound)
	// Act: пытаемся обернуть AppError в новый AppError.
	wrapped := app_error.New("обёртка", inner, app_error.Internal)

	// Assert: возвращается исходный экземпляр без повторного обёртывания.
	if wrapped != inner {
		t.Error("New() должен вернуть оригинальный AppError без обёртывания")
	}
	// Оригинальная категория не затирается новым kind.
	if wrapped.Kind() != app_error.NotFound {
		t.Error("должен сохраниться оригинальный Kind")
	}
}

func TestError_UserMessageOnly(t *testing.T) {
	t.Parallel()

	err := app_error.New("доступ запрещён", errors.New("permission denied"), app_error.PermissionDenied)
	// Assert: Error() отдаёт только пользовательское сообщение, скрывая техническое.
	if got := err.Error(); got != "доступ запрещён" {
		t.Errorf("Error() = %q, want %q", got, "доступ запрещён")
	}
}

func TestKindFunction(t *testing.T) {
	t.Parallel()

	// Таблица соответствия: ошибка -> ожидаемая категория.
	tests := []struct {
		name string
		err  error
		want app_error.ErrorKind
	}{
		{
			name: "InvalidArgument",
			err:  app_error.New("bad request", errors.New("bad"), app_error.InvalidArgument),
			want: app_error.InvalidArgument,
		},
		{
			name: "DeadlineExceeded",
			err:  app_error.New("timeout", errors.New("timeout"), app_error.DeadlineExceeded),
			want: app_error.DeadlineExceeded,
		},
		{
			name: "NotFound",
			err:  app_error.New("not found", errors.New("missing"), app_error.NotFound),
			want: app_error.NotFound,
		},
		{
			name: "AlreadyExists",
			err:  app_error.New("exists", errors.New("dup"), app_error.AlreadyExists),
			want: app_error.AlreadyExists,
		},
		{
			name: "Internal",
			err:  app_error.New("internal", errors.New("internal"), app_error.Internal),
			want: app_error.Internal,
		},
		{
			name: "plain error returns Internal as default",
			err:  errors.New("plain error"),
			want: app_error.Internal,
		},
		{
			name: "nil error returns Internal",
			err:  nil,
			want: app_error.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Act+Assert: категория извлекается из цепочки ошибок.
			got := app_error.Kind(tt.err)
			if got != tt.want {
				t.Errorf("Kind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogErrorFunction(t *testing.T) {
	t.Parallel()
	inner := errors.New("underlying error")
	appErr := app_error.New("user msg", inner, app_error.Internal)

	t.Run("app_error извлекает внутреннюю ошибку", func(t *testing.T) {
		t.Parallel()
		if got := app_error.LogError(appErr); !errors.Is(got, inner) {
			t.Errorf("LogError() = %v, want %v", got, inner)
		}
	})

	t.Run("обычная ошибка возвращается как есть", func(t *testing.T) {
		t.Parallel()
		plain := errors.New("plain")
		if got := app_error.LogError(plain); !errors.Is(got, plain) {
			t.Errorf("LogError() = %v, want %v", got, plain)
		}
	})

	t.Run("nil возвращает nil", func(t *testing.T) {
		t.Parallel()
		if got := app_error.LogError(nil); got != nil {
			t.Errorf("LogError() = %v, want nil", got)
		}
	})
}

func TestErrorKind_String(t *testing.T) {
	t.Parallel()

	// Таблица: категория -> строковое представление (включая неизвестные значения).
	tests := []struct {
		kind app_error.ErrorKind
		want string
	}{
		{app_error.InvalidArgument, "invalid_argument"},
		{app_error.DeadlineExceeded, "deadline_exceeded"},
		{app_error.NotFound, "not_found"},
		{app_error.AlreadyExists, "already_exists"},
		{app_error.PermissionDenied, "permission_denied"},
		{app_error.Unauthenticated, "unauthenticated"},
		{app_error.Internal, "internal"},
		{app_error.Unavailable, "unavailable"},
		{app_error.FailedPrecondition, "failed_precondition"},
		{app_error.ResourceExhausted, "resource_exhausted"},
		{app_error.ErrorKind(0), "unknown"},
		{app_error.ErrorKind(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorKind_HTTPStatus(t *testing.T) {
	t.Parallel()

	// Таблица: категория -> HTTP-статус.
	tests := []struct {
		kind app_error.ErrorKind
		want int
	}{
		{app_error.InvalidArgument, http.StatusBadRequest},
		{app_error.DeadlineExceeded, http.StatusGatewayTimeout},
		{app_error.NotFound, http.StatusNotFound},
		{app_error.AlreadyExists, http.StatusConflict},
		{app_error.PermissionDenied, http.StatusForbidden},
		{app_error.Unauthenticated, http.StatusUnauthorized},
		{app_error.Internal, http.StatusInternalServerError},
		{app_error.Unavailable, http.StatusServiceUnavailable},
		{app_error.FailedPrecondition, http.StatusBadRequest},
		{app_error.ResourceExhausted, http.StatusTooManyRequests},
		{app_error.ErrorKind(0), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v->%d", tt.kind, tt.want), func(t *testing.T) {
			t.Parallel()
			if got := tt.kind.HTTPStatus(); got != tt.want {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestErrorKind_GRPCCode(t *testing.T) {
	t.Parallel()

	// Таблица: категория -> gRPC-код.
	tests := []struct {
		kind app_error.ErrorKind
		want codes.Code
	}{
		{app_error.InvalidArgument, codes.InvalidArgument},
		{app_error.DeadlineExceeded, codes.DeadlineExceeded},
		{app_error.NotFound, codes.NotFound},
		{app_error.AlreadyExists, codes.AlreadyExists},
		{app_error.PermissionDenied, codes.PermissionDenied},
		{app_error.Unauthenticated, codes.Unauthenticated},
		{app_error.Internal, codes.Internal},
		{app_error.Unavailable, codes.Unavailable},
		{app_error.FailedPrecondition, codes.FailedPrecondition},
		{app_error.ResourceExhausted, codes.ResourceExhausted},
		{app_error.ErrorKind(0), codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			t.Parallel()
			if got := tt.kind.GRPCCode(); got != tt.want {
				t.Errorf("GRPCCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPStatusFunction(t *testing.T) {
	t.Parallel()

	// AppError маппится в HTTP-статус по категории.
	appErr := app_error.New("not found", errors.New("missing"), app_error.NotFound)
	if got := app_error.HTTPStatus(appErr); got != http.StatusNotFound {
		t.Errorf("HTTPStatus(appErr) = %d, want %d", got, http.StatusNotFound)
	}

	// Обычная ошибка -> 500 (Internal Server Error).
	plainErr := errors.New("plain")
	if got := app_error.HTTPStatus(plainErr); got != http.StatusInternalServerError {
		t.Errorf("HTTPStatus(plainErr) = %d, want %d", got, http.StatusInternalServerError)
	}

	// nil-ошибка -> 500.
	if got := app_error.HTTPStatus(nil); got != http.StatusInternalServerError {
		t.Errorf("HTTPStatus(nil) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestGRPCCodeFunction(t *testing.T) {
	t.Parallel()

	// AppError маппится в gRPC-код по категории.
	appErr := app_error.New("invalid", errors.New("bad"), app_error.InvalidArgument)
	if got := app_error.GRPCCode(appErr); got != codes.InvalidArgument {
		t.Errorf("GRPCCode(appErr) = %v, want %v", got, codes.InvalidArgument)
	}

	// Обычная ошибка -> codes.Unknown.
	plainErr := errors.New("plain")
	if got := app_error.GRPCCode(plainErr); got != codes.Unknown {
		t.Errorf("GRPCCode(plainErr) = %v, want %v", got, codes.Unknown)
	}

	// nil-ошибка -> codes.Unknown.
	if got := app_error.GRPCCode(nil); got != codes.Unknown {
		t.Errorf("GRPCCode(nil) = %v, want %v", got, codes.Unknown)
	}
}

func TestErrorWrapping(t *testing.T) {
	t.Parallel()

	// Arrange: AppError, обёрнутый стандартным fmt.Errorf.
	inner := errors.New("inner")
	appErr := app_error.New("user msg", inner, app_error.InvalidArgument)
	wrapped := fmt.Errorf("wrapped: %w", appErr)

	// errors.Is проходит всю цепочку до исходной ошибки.
	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is не проходит через цепочку wrapped -> AppError -> inner")
	}

	// errors.As извлекает AppError из цепочки.
	var target *app_error.AppError
	if !errors.As(wrapped, &target) {
		t.Error("errors.As не извлекает AppError из цепочки")
	} else if target.Kind() != app_error.InvalidArgument {
		t.Errorf("Kind() = %v, want %v", target.Kind(), app_error.InvalidArgument)
	}
}

func TestStackTrace(t *testing.T) {
	t.Parallel()

	// Arrange: ошибка со сбором стека вызовов.
	err := app_error.NewWithStack("test stack", errors.New("cause"), app_error.Internal)

	// Пакетная функция возвращает стек из AppError.
	st := app_error.StackTrace(err)
	if len(st) == 0 {
		t.Fatal("StackTrace() вернул пустой срез")
	}

	// Метод экземпляра возвращает тот же стек.
	stDirect := err.StackTrace()
	if len(stDirect) == 0 {
		t.Fatal("AppError.StackTrace() вернул пустой срез")
	}

	// Для обычной ошибки стек отсутствует (nil).
	plainErr := errors.New("plain")
	if got := app_error.StackTrace(plainErr); got != nil {
		t.Error("StackTrace(plainErr) должен вернуть nil")
	}
}

func TestAsAppError(t *testing.T) {
	t.Parallel()

	// AppError извлекается из самой себя.
	appErr := app_error.New("test", errors.New("cause"), app_error.Internal)
	got, ok := app_error.AsAppError(appErr)
	if !ok {
		t.Fatal("AsAppError не нашёл AppError")
	}
	if got != appErr {
		t.Error("AsAppError вернул не ту ошибку")
	}

	// Обычная ошибка не является AppError.
	plainErr := errors.New("plain")
	_, ok = app_error.AsAppError(plainErr)
	if ok {
		t.Error("AsAppError(plainErr) должен вернуть false")
	}
}
