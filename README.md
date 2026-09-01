# apppkg

Набор переиспользуемых Go-пакетов для типовых задач сервисной разработки:
обработка ошибок, служебная информация о запросе, структурированное логирование
и корректное (graceful) завершение работы приложения.

## Установка

```bash
go get github.com/delta-five/apppkg
```

Модуль требует Go 1.27+.

## Состав

| Пакет      | Назначение                                                                 |
| ---------- | -------------------------------------------------------------------------- |
| `app_error` | Ошибка приложения с разделением сообщений для пользователя и для логов      |
| `callinfo`  | Служебная информация о запросе (ID, IP, метод), передаваемая через контекст |
| `logger`    | Структурированный логгер на базе `log/slog` с поддержкой контекста          |
| `closer`    | Реестр функций остановки для graceful shutdown                             |

---

## app_error

Тип `AppError` разделяет **сообщение для пользователя** (безопасное для отображения)
и **техническую информацию** для логирования, а также хранит категорию ошибки
(аналог gRPC-кодов) и опциональный стек вызовов.

```go
package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/delta-five/apppkg/app_error"
)

func findUser(id int) error {
	return app_error.New(
		"пользователь не найден",
		fmt.Errorf("user id=%d: %w", id, sql.ErrNoRows),
		app_error.NotFound,
	)
}

func main() {
	err := findUser(42)

	// Сообщение для пользователя.
	fmt.Println(err.Error()) // пользователь не найден

	// Категория ошибки.
	fmt.Println(app_error.Kind(err)) // not_found

	// Техническая информация для логов.
	fmt.Println(app_error.LogError(err)) // user id=42: sql: no rows in result set
}
```

### Категории ошибок

| Категория             | HTTP     | gRPC                |
| --------------------- | -------- | ------------------- |
| `InvalidArgument`     | 400      | `InvalidArgument`   |
| `DeadlineExceeded`    | 504      | `DeadlineExceeded`  |
| `NotFound`            | 404      | `NotFound`          |
| `AlreadyExists`       | 409      | `AlreadyExists`     |
| `PermissionDenied`    | 403      | `PermissionDenied`  |
| `Unauthenticated`     | 401      | `Unauthenticated`   |
| `Internal`            | 500      | `Internal`          |
| `Unavailable`         | 503      | `Unavailable`       |
| `FailedPrecondition`  | 400      | `FailedPrecondition`|
| `ResourceExhausted`   | 429      | `ResourceExhausted` |

### Маппинг ошибки в HTTP/gRPC статус

```go
err := app_error.New("доступ запрещён", errors.New("forbidden"), app_error.PermissionDenied)

httpCode := app_error.HTTPStatus(err) // 403
grpcCode := app_error.GRPCCode(err)   // codes.PermissionDenied
```

### Работа через `errors.As` и стек вызовов

```go
var appErr *app_error.AppError
if errors.As(err, &appErr) {
	fmt.Println(appErr.Kind())     // категория
	fmt.Println(appErr.LogError()) // в логи
}

// Ошибка со стеком вызовов.
err = app_error.NewWithStack("внутренняя ошибка", cause, app_error.Internal)
st := app_error.StackTrace(err) // []uintptr
```

> `HTTPStatus` дополнительно умеет маппить gRPC `status.Error` в HTTP-код,
> если ошибка не является `AppError`.

---

## callinfo

Интерфейс `CallInfo` предоставляет атрибуты (`[]slog.Attr`) для добавления
служебной информации о запросе в структурированный лог. Передаётся через контекст.

```go
package main

import (
	"context"

	"github.com/delta-five/apppkg/callinfo"
	"github.com/delta-five/apppkg/logger"
)

func main() {
	req := callinfo.MakeSimpleRequest()

	ctx := callinfo.WithCallInfo(context.Background(), req)

	// Атрибуты из контекста автоматически попадут в лог.
	logger.Info(ctx, "запрос обработан")
}
```

`SimpleRequest` — минимальная реализация, хранящая только ID запроса.
Для передачи дополнительных полей реализуйте свой тип:

```go
type reqInfo struct {
	ID      string
	IP      string
	Handler string
}

func (r reqInfo) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("id", r.ID),
		slog.String("ip", r.IP),
		slog.String("handler", r.Handler),
	}
}

ctx := callinfo.WithCallInfo(context.Background(), reqInfo{
	ID:      callinfo.GenerateID(),
	IP:      "192.168.1.1",
	Handler: "/api/v1/users",
})
```

---

## logger

Структурированный логгер на базе `log/slog` с поддержкой контекста.
Поддерживает три формы записи: простое сообщение, форматированная строка
и сообщение с парами ключ-значение.

### Инициализация

```go
package main

import (
	"os"

	"github.com/delta-five/apppkg/logger"
)

func main() {
	logger.Init(logger.LevelInfo, os.Stdout)

	// Парсинг уровня из строки (удобно для конфигурации).
	level := logger.MustParseLevel("debug")
	logger.SetLevel(level)
}
```

### Логирование

```go
ctx := context.Background()

logger.Info(ctx, "сервер запущен")
logger.Infof(ctx, "порт: %d", 8080)
logger.InfoKV(ctx, "запрос обработан", "method", "GET", "path", "/api/v1/users")
logger.Error(ctx, "ошибка при обработке")
logger.Errorf(ctx, "не удалось сохранить: %v", err)
```

### Создание экземпляра для внедрения зависимостей

```go
l := logger.New(logger.LevelDebug, os.Stdout)
l.Info(ctx, "сообщение")
l.With("service", "api").Infof(ctx, "запрос №%d", 42)
```

Доступные уровни: `LevelDebug`, `LevelInfo`, `LevelWarn`, `LevelError`.
Уровень можно менять в рантайме через `logger.SetLevel`.

---

## closer

Реестр функций остановки для graceful shutdown. Функции выполняются в порядке,
обратном их регистрации, при получении сигнала `SIGINT`, `SIGTERM` или `SIGQUIT`
(либо при отмене контекста).

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/delta-five/apppkg/closer"
	"github.com/delta-five/apppkg/logger"
)

func main() {
	// Init создаёт контекст приложения и запускает обработчик сигналов ОС.
	ctx, cancel := closer.Init(context.Background(), 10*time.Second)
	defer cancel()

	// Регистрация функций остановки.
	_ = closer.Add("db", func() {
		db.Close()
	})

	_ = closer.AddWithContextError("http", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})

	// ... запуск сервера, работа приложения ...

	// Дожидаемся завершения. При получении SIGINT/SIGTERM/SIGQUIT
	// (или отмене ctx) зарегистрированные функции выполнятся автоматически.
	<-ctx.Done()
}
```

### Варианты регистрации

```go
closer.Add(name, func())                              // CloseFunc
closer.AddWithError(name, func() error)               // CloseFuncWithError
closer.AddWithContextError(name, func(ctx) error)     // CloseWithContextErrorFunc
closer.AddCloser(name, obj)                           // интерфейс Closer
closer.AddErrorCloser(name, obj)                      // интерфейс ErrorCloser
closer.AddContextErrorCloser(name, obj)               // интерфейс ContextErrorCloser
```

> `Init` задаёт общее ограничение времени на остановку всех зарегистрированных
> функций через `closeTimeout`.

---

## Разработка

```bash
go test ./...          # запуск тестов
go vet ./...           # статический анализ
task lint              # golangci-lint (см. .golangci.pipeline.yaml)
```

## Лицензия

[MIT](LICENSE)
