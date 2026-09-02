package app_error_test

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"

	"github.com/delta-five/apppkg/app_error"
)

func ExampleNew() {
	// Arrange: техническая ошибка.
	inner := errors.New("sql: no rows in result set")
	// Act: оборачиваем её в AppError с категорией NotFound.
	err := app_error.New("пользователь не найден", inner, app_error.NotFound)

	// Assert: все представления одной ошибки.
	fmt.Println(err.Error())
	fmt.Println(app_error.Kind(err))
	fmt.Println(app_error.LogError(err))
	fmt.Println(app_error.HTTPStatus(err))
	fmt.Println(app_error.GRPCCode(err))

	// Output:
	// пользователь не найден
	// not_found
	// sql: no rows in result set
	// 404
	// NotFound
}

func ExampleNew_missing() {
	err := app_error.New("некорректный email", errors.New("email validation: missing @"), app_error.InvalidArgument)

	fmt.Println(err.Error())
	fmt.Println(err.Kind())

	// Output:
	// некорректный email
	// invalid_argument
}

func ExampleErrorKind_HTTPStatus() {
	for _, kind := range []app_error.ErrorKind{
		app_error.InvalidArgument,
		app_error.NotFound,
		app_error.Internal,
		app_error.Unavailable,
	} {
		fmt.Printf("%s -> %d\n", kind, kind.HTTPStatus())
	}

	// Output:
	// invalid_argument -> 400
	// not_found -> 404
	// internal -> 500
	// unavailable -> 503
}

func ExampleErrorKind_GRPCCode() {
	kind := app_error.DeadlineExceeded
	fmt.Println(kind.GRPCCode())

	// Output:
	// DeadlineExceeded
}

func ExampleKind() {
	plain := errors.New("что-то пошло не так")
	fmt.Println(app_error.Kind(plain))

	appErr := app_error.New("ресурс исчерпан", plain, app_error.ResourceExhausted)
	fmt.Println(app_error.Kind(appErr))

	// Output:
	// internal
	// resource_exhausted
}

func ExampleLogError() {
	inner := errors.New("connection reset by peer")
	appErr := app_error.New("сервис недоступен", inner, app_error.Unavailable)

	fmt.Println(app_error.LogError(appErr))
	fmt.Println(app_error.LogError(inner))

	// Output:
	// connection reset by peer
	// connection reset by peer
}

func ExampleHTTPStatus() {
	err := app_error.New("не найдено", errors.New("not found"), app_error.NotFound)
	fmt.Println(app_error.HTTPStatus(err))

	fmt.Println(app_error.HTTPStatus(errors.New("plain error")))

	// Output:
	// 404
	// 500
}

func ExampleGRPCCode() {
	err := app_error.New("доступ запрещён", errors.New("forbidden"), app_error.PermissionDenied)
	fmt.Println(app_error.GRPCCode(err))

	// Output:
	// PermissionDenied
}

func Example_httpHandler() {
	// Моделируем типичный обработчик, возвращающий AppError.
	simulateError := func() error {
		return app_error.New(
			"пользователь не найден",
			fmt.Errorf("user id=42: %w", errors.New("record not found")),
			app_error.NotFound,
		)
	}

	err := simulateError()
	// HTTP-статус — пользователю, техническая ошибка — в логи.
	httpStatus := app_error.HTTPStatus(err)
	logErr := app_error.LogError(err)

	fmt.Println(httpStatus)
	fmt.Println(logErr)

	// Output:
	// 404
	// user id=42: record not found
}

func Example_grpcHandler() {
	simulateError := func() error {
		return app_error.New(
			"некорректное имя пользователя",
			errors.New("username must be 3-30 characters"),
			app_error.InvalidArgument,
		)
	}

	err := simulateError()
	grpcCode := app_error.GRPCCode(err)
	logErr := app_error.LogError(err)

	fmt.Println(grpcCode)
	fmt.Println(logErr)

	// Output:
	// InvalidArgument
	// username must be 3-30 characters
}

func Example_errorsAs() {
	process := func(id int) error {
		if id <= 0 {
			return app_error.New(
				"ID должен быть положительным",
				fmt.Errorf("invalid id: %d", id),
				app_error.InvalidArgument,
			)
		}
		return nil
	}

	err := process(-1)

	// Извлекаем AppError из цепочки и работаем с его полями по отдельности.
	var appErr *app_error.AppError
	if errors.As(err, &appErr) {
		fmt.Println("категория:", appErr.Kind())
		fmt.Println("пользователю:", appErr.Error())
		fmt.Println("в логи:", appErr.LogError())
	}

	// Output:
	// категория: invalid_argument
	// пользователю: ID должен быть положительным
	// в логи: invalid id: -1
}

func ExampleStackTrace() {
	// Ошибка со сбором стека вызовов создаётся через NewWithStack.
	err := app_error.NewWithStack("ошибка с трейсом", errors.New("причина"), app_error.Internal)
	st := app_error.StackTrace(err)
	fmt.Println(len(st) > 0)

	// Output:
	// true
}

var _ codes.Code
var _ = http.StatusOK
