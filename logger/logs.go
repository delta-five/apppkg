// Package logger предоставляет глобальный структурированный логгер на основе
// log/slog с поддержкой контекста и тремя формами вывода: простое сообщение,
// форматированная строка и сообщение с парами ключ-значение.
//
// Пример использования:
//
//	// Инициализация глобального логгера (один раз при старте).
//	logger.Init(logger.LevelInfo, os.Stdout)
//
//	// Использование через пакетные функции.
//	logger.Info(ctx, "сервер запущен")
//	logger.Infof(ctx, "порт: %d", 8080)
//	logger.InfoKV(ctx, "запрос обработан", "method", "GET", "path", "/api/v1/users")
//
//	// Либо создание экземпляра для внедрения зависимостей.
//	l := logger.New(logger.LevelDebug, os.Stdout)
//	l.Info(ctx, "сообщение")
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/delta-five/apppkg/callinfo"
)

// Level — уровень логирования.
type Level slog.Level

// Константы уровней логирования.
const (
	LevelDebug Level = Level(slog.LevelDebug)
	LevelInfo  Level = Level(slog.LevelInfo)
	LevelWarn  Level = Level(slog.LevelWarn)
	LevelError Level = Level(slog.LevelError)
)

// ParseLevel преобразует строку в Level.
// Допустимые значения: "debug", "info", "warn", "error" (регистр не важен).
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error", "err":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level: %q", s)
	}
}

// MustParseLevel преобразует строку в Level. При ошибке паникует.
func MustParseLevel(s string) Level {
	lvl, err := ParseLevel(s)
	if err != nil {
		panic(err)
	}
	return lvl
}

// String возвращает строковое представление уровня.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", int(l))
	}
}

// Logger — структурированный логгер с поддержкой контекста.
type Logger struct {
	logger *slog.Logger
	level  *slog.LevelVar
}

// global — глобальный экземпляр логгера по умолчанию.
var (
	global   *Logger
	globalMu sync.RWMutex
)

// defaultGlobal возвращает логгер по умолчанию (инициализируется лениво).
// Потокобезопасна, использует double-checked locking.
func defaultGlobal() *Logger {
	globalMu.RLock()
	if global != nil {
		defer globalMu.RUnlock()
		return global
	}
	globalMu.RUnlock()

	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		lvl := &slog.LevelVar{}
		lvl.Set(slog.LevelDebug)
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
		global = &Logger{
			logger: slog.New(handler),
			level:  lvl,
		}
	}
	return global
}

// Init инициализирует глобальный логгер с указанным уровнем и каналом вывода.
// Должен быть вызван один раз при старте приложения.
func Init(level Level, writer io.Writer) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.Level(level))

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: levelVar})

	globalMu.Lock()
	defer globalMu.Unlock()
	global = &Logger{
		logger: slog.New(handler),
		level:  levelVar,
	}
}

// New создаёт новый экземпляр Logger с указанным уровнем и каналом вывода.
func New(level Level, writer io.Writer) *Logger {
	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.Level(level))

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: levelVar})

	return &Logger{
		logger: slog.New(handler),
		level:  levelVar,
	}
}

// SetLevel изменяет уровень логирования глобального логгера в рантайме.
func SetLevel(level Level) {
	defaultGlobal().setLevel(level)
}

// CurrentLevel возвращает текущий уровень глобального логгера.
func CurrentLevel() Level {
	return defaultGlobal().getLevel()
}

// setLevel изменяет уровень логирования.
func (l *Logger) setLevel(level Level) {
	l.level.Set(slog.Level(level))
}

// getLevel возвращает текущий уровень логирования.
func (l *Logger) getLevel() Level {
	return Level(l.level.Level())
}

// contextAttrs извлекает атрибуты из контекста для добавления в запись лога.
func contextAttrs(ctx context.Context) []slog.Attr {
	ci := callinfo.FromContext(ctx)
	if ci == nil {
		return nil
	}
	return ci.Attrs()
}

// log — внутренний helper для логирования.
func (l *Logger) log(ctx context.Context, level Level, msg string, logArgs []any) {
	attrs := contextAttrs(ctx)
	if len(attrs) > 0 {
		logArgs = append(logArgs, attrsToSlogArgs(attrs)...)
	}
	l.logger.Log(ctx, slog.Level(level), msg, logArgs...)
}

func (l *Logger) logf(ctx context.Context, level Level, format string, msgArgs []any, logArgs []any) {
	if l.level.Level() > slog.Level(level) {
		return
	}
	msg := fmt.Sprintf(format, msgArgs...)
	l.log(ctx, level, msg, logArgs)
}

// attrsToSlogArgs преобразует []slog.Attr в []any для передачи как args.
func attrsToSlogArgs(attrs []slog.Attr) []any {
	result := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		result = append(result, attr.Key, attr.Value)
	}
	return result
}

// With возвращает новый Logger с дополнительными атрибутами.
// Атрибуты передаются парами ключ-значение.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
		level:  l.level,
	}
}

// ---- Методы экземпляра Logger ----

// Debug логирует сообщение на уровне debug.
func (l *Logger) Debug(ctx context.Context, msg string) {
	l.log(ctx, LevelDebug, msg, nil)
}

// Debugf логирует форматированное сообщение на уровне debug.
func (l *Logger) Debugf(ctx context.Context, format string, args ...any) {
	l.logf(ctx, LevelDebug, format, args, nil)
}

// DebugKV логирует сообщение с парами ключ-значение на уровне debug.
func (l *Logger) DebugKV(ctx context.Context, msg string, keys ...any) {
	l.log(ctx, LevelDebug, msg, keys)
}

// Info логирует сообщение на уровне info.
func (l *Logger) Info(ctx context.Context, msg string) {
	l.log(ctx, LevelInfo, msg, nil)
}

// Infof логирует форматированное сообщение на уровне info.
func (l *Logger) Infof(ctx context.Context, format string, args ...any) {
	l.logf(ctx, LevelInfo, format, args, nil)
}

// InfoKV логирует сообщение с парами ключ-значение на уровне info.
func (l *Logger) InfoKV(ctx context.Context, msg string, keys ...any) {
	l.log(ctx, LevelInfo, msg, keys)
}

// Warn логирует сообщение на уровне warn.
func (l *Logger) Warn(ctx context.Context, msg string) {
	l.log(ctx, LevelWarn, msg, nil)
}

// Warnf логирует форматированное сообщение на уровне warn.
func (l *Logger) Warnf(ctx context.Context, format string, args ...any) {
	l.logf(ctx, LevelWarn, format, args, nil)
}

// WarnKV логирует сообщение с парами ключ-значение на уровне warn.
func (l *Logger) WarnKV(ctx context.Context, msg string, keys ...any) {
	l.log(ctx, LevelWarn, msg, keys)
}

// Error логирует сообщение на уровне error.
func (l *Logger) Error(ctx context.Context, msg string) {
	l.log(ctx, LevelError, msg, nil)
}

// Errorf логирует форматированное сообщение на уровне error.
func (l *Logger) Errorf(ctx context.Context, format string, args ...any) {
	l.logf(ctx, LevelError, format, args, nil)
}

// ErrorKV логирует сообщение с парами ключ-значение на уровне error.
func (l *Logger) ErrorKV(ctx context.Context, msg string, keys ...any) {
	l.log(ctx, LevelError, msg, keys)
}

// ---- Глобальные функции (обёртки над defaultGlobal()) ----

// Debug логирует сообщение на уровне debug через глобальный логгер.
func Debug(ctx context.Context, msg string) {
	defaultGlobal().Debug(ctx, msg)
}

// Debugf логирует форматированное сообщение на уровне debug через глобальный логгер.
func Debugf(ctx context.Context, format string, args ...any) {
	defaultGlobal().Debugf(ctx, format, args...)
}

// DebugKV логирует сообщение с парами ключ-значение на уровне debug через глобальный логгер.
func DebugKV(ctx context.Context, msg string, keys ...any) {
	defaultGlobal().DebugKV(ctx, msg, keys...)
}

// Info логирует сообщение на уровне info через глобальный логгер.
func Info(ctx context.Context, msg string) {
	defaultGlobal().Info(ctx, msg)
}

// Infof логирует форматированное сообщение на уровне info через глобальный логгер.
func Infof(ctx context.Context, format string, args ...any) {
	defaultGlobal().Infof(ctx, format, args...)
}

// InfoKV логирует сообщение с парами ключ-значение на уровне info через глобальный логгер.
func InfoKV(ctx context.Context, msg string, keys ...any) {
	defaultGlobal().InfoKV(ctx, msg, keys...)
}

// Warn логирует сообщение на уровне warn через глобальный логгер.
func Warn(ctx context.Context, msg string) {
	defaultGlobal().Warn(ctx, msg)
}

// Warnf логирует форматированное сообщение на уровне warn через глобальный логгер.
func Warnf(ctx context.Context, format string, args ...any) {
	defaultGlobal().Warnf(ctx, format, args...)
}

// WarnKV логирует сообщение с парами ключ-значение на уровне warn через глобальный логгер.
func WarnKV(ctx context.Context, msg string, keys ...any) {
	defaultGlobal().WarnKV(ctx, msg, keys...)
}

// Error логирует сообщение на уровне error через глобальный логгер.
func Error(ctx context.Context, msg string) {
	defaultGlobal().Error(ctx, msg)
}

// Errorf логирует форматированное сообщение на уровне error через глобальный логгер.
func Errorf(ctx context.Context, format string, args ...any) {
	defaultGlobal().Errorf(ctx, format, args...)
}

// ErrorKV логирует сообщение с парами ключ-значение на уровне error через глобальный логгер.
func ErrorKV(ctx context.Context, msg string, keys ...any) {
	defaultGlobal().ErrorKV(ctx, msg, keys...)
}

// Проверка соответствия интерфейсу на этапе компиляции.
var _ interface {
	Debug(ctx context.Context, msg string)
	Debugf(ctx context.Context, format string, args ...any)
	DebugKV(ctx context.Context, msg string, keys ...any)
	Info(ctx context.Context, msg string)
	Infof(ctx context.Context, format string, args ...any)
	InfoKV(ctx context.Context, msg string, keys ...any)
	Warn(ctx context.Context, msg string)
	Warnf(ctx context.Context, format string, args ...any)
	WarnKV(ctx context.Context, msg string, keys ...any)
	Error(ctx context.Context, msg string)
	Errorf(ctx context.Context, format string, args ...any)
	ErrorKV(ctx context.Context, msg string, keys ...any)
} = (*Logger)(nil)

// KVLogFunc — сигнатура функции логирования сообщения с парами ключ-значение.
type KVLogFunc func(ctx context.Context, msg string, keys ...any)

var (
	_ KVLogFunc = InfoKV
	_ KVLogFunc = WarnKV
	_ KVLogFunc = ErrorKV
	_ KVLogFunc = DebugKV
)
