// Package app_error предоставляет тип AppError — внутреннюю ошибку приложения,
// которая разделяет сообщение для пользователя (безопасное для отображения)
// и техническую информацию для логирования.
//
// Пример использования:
//
//	// Создание ошибки с указанием категории.
//	err := app_error.New("пользователь не найден", fmt.Errorf("user id=%d: %w", id, sql.ErrNoRows), app_error.NotFound)
//
//	// Проверка категории.
//	if app_error.Kind(err) == app_error.NotFound {
//		// обработка
//	}
//
//	// Извлечение технической ошибки для логирования.
//	logger.Error(ctx, "ошибка БД", "error", app_error.LogError(err))
package app_error

import (
	"errors"
	"net/http"
	"runtime"

	runtime2 "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorKind — тип исходной проблемы (категория ошибки).
type ErrorKind int

// Константы категорий ошибок.
const (
	// InvalidArgument — некорректные параметры запроса.
	InvalidArgument ErrorKind = iota + 1
	// DeadlineExceeded — превышение времени ожидания (таймаут).
	DeadlineExceeded
	// NotFound — запрошенный ресурс не найден.
	NotFound
	// AlreadyExists — ресурс уже существует.
	AlreadyExists
	// PermissionDenied — недостаточно прав для выполнения операции.
	PermissionDenied
	// Unauthenticated — требуется аутентификация.
	Unauthenticated
	// Internal — внутренняя ошибка сервера.
	Internal
	// Unavailable — сервис временно недоступен.
	Unavailable
	// FailedPrecondition — не выполнено предусловие для операции.
	FailedPrecondition
	// ResourceExhausted — исчерпан лимит ресурсов.
	ResourceExhausted
)

// String возвращает человекочитаемое название категории ошибки.
func (k ErrorKind) String() string {
	switch k {
	case InvalidArgument:
		return "invalid_argument"
	case DeadlineExceeded:
		return "deadline_exceeded"
	case NotFound:
		return "not_found"
	case AlreadyExists:
		return "already_exists"
	case PermissionDenied:
		return "permission_denied"
	case Unauthenticated:
		return "unauthenticated"
	case Internal:
		return "internal"
	case Unavailable:
		return "unavailable"
	case FailedPrecondition:
		return "failed_precondition"
	case ResourceExhausted:
		return "resource_exhausted"
	default:
		return "unknown"
	}
}

// HTTPStatus возвращает HTTP-статус код, соответствующий категории ошибки.
func (k ErrorKind) HTTPStatus() int {
	switch k {
	case InvalidArgument:
		return http.StatusBadRequest
	case DeadlineExceeded:
		return http.StatusGatewayTimeout
	case NotFound:
		return http.StatusNotFound
	case AlreadyExists:
		return http.StatusConflict
	case PermissionDenied:
		return http.StatusForbidden
	case Unauthenticated:
		return http.StatusUnauthorized
	case Internal:
		return http.StatusInternalServerError
	case Unavailable:
		return http.StatusServiceUnavailable
	case FailedPrecondition:
		return http.StatusBadRequest
	case ResourceExhausted:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// GRPCCode возвращает gRPC-статус код, соответствующий категории ошибки.
func (k ErrorKind) GRPCCode() codes.Code {
	switch k {
	case InvalidArgument:
		return codes.InvalidArgument
	case DeadlineExceeded:
		return codes.DeadlineExceeded
	case NotFound:
		return codes.NotFound
	case AlreadyExists:
		return codes.AlreadyExists
	case PermissionDenied:
		return codes.PermissionDenied
	case Unauthenticated:
		return codes.Unauthenticated
	case Internal:
		return codes.Internal
	case Unavailable:
		return codes.Unavailable
	case FailedPrecondition:
		return codes.FailedPrecondition
	case ResourceExhausted:
		return codes.ResourceExhausted
	default:
		return codes.Unknown
	}
}

// AppError — внутренняя ошибка приложения, скрывающая техническую информацию
// от пользователя. Реализует интерфейс error, возвращая в Error() сообщение
// для пользователя. Техническая информация доступна через LogError().
type AppError struct {
	msg        string    // сообщение для пользователя (безопасно для отображения)
	err        error     // техническая ошибка для логирования
	kind       ErrorKind // категория проблемы
	stackTrace []uintptr // стек вызовов в момент создания ошибки
}

// newAppError создаёт новую ошибку AppError.
// msg — сообщение для пользователя (безопасно для отображения).
// err — техническая ошибка (пишется в логи).
// kind — категория проблемы.
// stackTrace — стек трейс.
//
// Если err уже является *AppError (в том числе обёрнутым в цепочке ошибок),
// то возвращается этот AppError как есть (без сбора StackTrace и без обёртывания).
func newAppError(msg string, err error, kind ErrorKind, stackTrace []uintptr) *AppError {
	if err == nil {
		err = errors.New(msg)
	}

	if appErr, ok := errors.AsType[*AppError](err); ok {
		return appErr
	}

	return &AppError{
		msg:        msg,
		err:        err,
		kind:       kind,
		stackTrace: stackTrace,
	}
}

// New создаёт новую ошибку AppError.
// msg — сообщение для пользователя (безопасно для отображения).
// err — техническая ошибка (пишется в логи).
// kind — категория проблемы.
//
// Если err уже является *AppError (в том числе обёрнутым в цепочке ошибок),
// то возвращается этот AppError как есть (без сбора StackTrace и без обёртывания).
func New(msg string, err error, kind ErrorKind) *AppError {
	return newAppError(msg, err, kind, nil)
}

// NewWithStack создаёт новую ошибку AppError с сохранением стека вызовов.
// msg — сообщение для пользователя (безопасно для отображения).
// err — техническая ошибка (пишется в логи).
// kind — категория проблемы.
//
// Если err уже является *AppError (в том числе обёрнутым в цепочке ошибок),
// то возвращается этот AppError как есть (без сбора StackTrace и без обёртывания).
func NewWithStack(msg string, err error, kind ErrorKind) *AppError {
	pc := make([]uintptr, 32)
	n := runtime.Callers(2, pc)

	return newAppError(msg, err, kind, pc[:n])
}

// Error возвращает сообщение для пользователя.
// Реализует интерфейс error.
func (e *AppError) Error() string {
	return e.msg
}

// Unwrap возвращает исходную техническую ошибку.
// Позволяет использовать errors.Is и errors.As для поиска в цепочке ошибок.
func (e *AppError) Unwrap() error {
	return e.err
}

// LogError возвращает техническую ошибку для записи в логи.
func (e *AppError) LogError() error {
	return e.err
}

// Kind возвращает категорию проблемы.
func (e *AppError) Kind() ErrorKind {
	return e.kind
}

// StackTrace возвращает сохранённый стек вызовов (счётчики команд).
func (e *AppError) StackTrace() []uintptr {
	return e.stackTrace
}

// ---- Вспомогательные функции для работы с AppError ----

// Kind извлекает категорию из ошибки. Если ошибка не является AppError,
// возвращает Internal (как наиболее безопасное значение по умолчанию).
func Kind(err error) ErrorKind {
	if appErr, ok := AsAppError(err); ok {
		return appErr.kind
	}
	return Internal
}

// LogError извлекает техническую ошибку из цепочки. Если ошибка не является
// AppError, возвращает саму ошибку.
func LogError(err error) error {
	if appErr, ok := AsAppError(err); ok {
		return appErr.err
	}
	return err
}

// HTTPStatus возвращает HTTP-статус код для ошибки.
// Если ошибка является AppError — по её категории; иначе, если ошибка
// является gRPC status-ошибкой — из её gRPC-кода; в остальных случаях — 500.
func HTTPStatus(err error) int {
	if appErr, ok := AsAppError(err); ok {
		return appErr.kind.HTTPStatus()
	}
	if se := status.Convert(err); se != nil {
		return runtime2.HTTPStatusFromCode(se.Code())
	}
	return http.StatusInternalServerError
}

// GRPCCode возвращает gRPC-статус код для ошибки. Если ошибка не является
// AppError, возвращает codes.Unknown.
func GRPCCode(err error) codes.Code {
	if appErr, ok := AsAppError(err); ok {
		return appErr.kind.GRPCCode()
	}
	return codes.Unknown
}

// StackTrace возвращает стек вызовов из ошибки, если она является AppError.
func StackTrace(err error) []uintptr {
	if appErr, ok := AsAppError(err); ok {
		return appErr.stackTrace
	}
	return nil
}

// AsAppError извлекает *AppError из цепочки ошибок.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// Проверка соответствия интерфейсам на этапе компиляции.
var _ error = (*AppError)(nil)
