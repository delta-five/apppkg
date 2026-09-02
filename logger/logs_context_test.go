package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/delta-five/apppkg/callinfo"
	"github.com/delta-five/apppkg/logger"
)

type testCallInfo struct {
	id string
}

// Attrs возвращает атрибут request_id для проверки интеграции с logger.
func (c testCallInfo) Attrs() []slog.Attr {
	return []slog.Attr{slog.String("request_id", c.id)}
}

func TestContextAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	// Arrange: кладём CallInfo в контекст.
	ctx := callinfo.WithCallInfo(context.Background(), testCallInfo{id: "req-123"})
	// Act: логируем — атрибуты из контекста должны добавиться автоматически.
	logger.Info(ctx, "message with context attrs")

	// Assert: вывод содержит атрибут из контекста.
	out := buf.String()
	if !strings.Contains(out, "request_id=req-123") {
		t.Errorf("вывод должен содержать атрибуты из контекста, got: %s", out)
	}
}

func TestContextAttrsEmpty(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	// Act: логируем без CallInfo в контексте.
	logger.Info(context.Background(), "message without context")

	out := buf.String()
	if !strings.Contains(out, "message without context") {
		t.Errorf("вывод должен содержать сообщение, got: %s", out)
	}
	// Assert: атрибутов контекста нет.
	if strings.Contains(out, "request_id") {
		t.Errorf("вывод не должен содержать атрибуты контекста, got: %s", out)
	}
}

func TestLoggerWithAttrs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	// Arrange: экземпляр логгера с постоянным атрибутом service.
	l := logger.New(logger.LevelDebug, &buf).With("service", "api")

	// Act: логируем сообщение.
	l.Info(context.Background(), "with attr")

	// Assert: постоянный атрибут присутствует в выводе.
	out := buf.String()
	if !strings.Contains(out, "service=api") {
		t.Errorf("вывод должен содержать атрибут service=api, got: %s", out)
	}
}

func TestLevelStringUnknown(t *testing.T) {
	t.Parallel()

	// Неизвестный уровень форматируется как unknown(N).
	if got := logger.Level(999).String(); got != "unknown(999)" {
		t.Errorf("Level(999).String() = %q, want %q", got, "unknown(999)")
	}
}

func TestParseLevelError(t *testing.T) {
	t.Parallel()

	// Неподдерживаемое значение уровня возвращает ошибку.
	if _, err := logger.ParseLevel("verbose"); err == nil {
		t.Error("ParseLevel(\"verbose\") должен вернуть ошибку")
	}
}
