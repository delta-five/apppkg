package logger_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/delta-five/apppkg/logger"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	// Таблица: строковое представление уровня (регистр и синонимы).
	tests := []struct {
		name    string
		input   string
		want    logger.Level
		wantErr bool
	}{
		{name: "debug lowercase", input: "debug", want: logger.LevelDebug},
		{name: "debug uppercase", input: "DEBUG", want: logger.LevelDebug},
		{name: "debug mixed case", input: "Debug", want: logger.LevelDebug},
		{name: "info lowercase", input: "info", want: logger.LevelInfo},
		{name: "info uppercase", input: "INFO", want: logger.LevelInfo},
		{name: "warn lowercase", input: "warn", want: logger.LevelWarn},
		{name: "warning lowercase", input: "warning", want: logger.LevelWarn},
		{name: "error lowercase", input: "error", want: logger.LevelError},
		{name: "err lowercase", input: "err", want: logger.LevelError},
		{name: "error uppercase", input: "ERROR", want: logger.LevelError},
		{name: "invalid level", input: "trace", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Act: парсим строку в уровень.
			got, err := logger.ParseLevel(tt.input)
			// Assert: некорректный ввод даёт ошибку.
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseLevel(%q) expected error, got nil", tt.input)
				}
				return
			}
			// Assert: корректный ввод даёт ожидаемый уровень без ошибки.
			if err != nil {
				t.Errorf("ParseLevel(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMustParseLevel(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		// Корректный ввод возвращает уровень.
		lvl := logger.MustParseLevel("debug")
		if lvl != logger.LevelDebug {
			t.Errorf("MustParseLevel = %v, want debug", lvl)
		}
	})

	t.Run("invalid panics", func(t *testing.T) {
		t.Parallel()
		// Некорректный ввод должен вызвать панику.
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustParseLevel expected panic for invalid level")
			}
		}()
		logger.MustParseLevel("trace")
	})
}

func TestLevelString(t *testing.T) {
	t.Parallel()

	// Таблица: уровень -> строковое представление.
	tests := []struct {
		level logger.Level
		want  string
	}{
		{logger.LevelDebug, "debug"},
		{logger.LevelInfo, "info"},
		{logger.LevelWarn, "warn"},
		{logger.LevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("with level and writer", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		// Act: создаём логгер с явным уровнем.
		logger := logger.New(logger.LevelDebug, &buf)
		if logger == nil {
			t.Fatal("New() returned nil")
		}
	})

	t.Run("nil level defaults to info", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		// Act: нулевое значение Level (0 = Info) не приводит к ошибке.
		logger := logger.New(0, &buf)
		if logger == nil {
			t.Fatal("New() returned nil")
		}
	})
}

func TestInit(t *testing.T) {
	// Arrange: инициализируем глобальный логгер с буфером в качестве вывода.
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	// Act: пишем через глобальную функцию.
	ctx := context.Background()
	logger.Info(ctx, "init test")

	// Assert: сообщение попало в буфер.
	if !strings.Contains(buf.String(), "init test") {
		t.Errorf("output should contain message, got: %s", buf.String())
	}
}

func TestGlobalInfo(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	logger.Info(ctx, "test info message")

	// Assert: сообщение и уровень INFO присутствуют в выводе.
	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Errorf("output should contain message, got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("output should contain level INFO, got: %s", output)
	}
}

func TestGlobalInfof(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	// Act: логируем с форматированием.
	logger.Infof(ctx, "formatted %s %d", "test", 42)

	// Assert: форматированная строка попала в вывод.
	output := buf.String()
	if !strings.Contains(output, "formatted test 42") {
		t.Errorf("output should contain formatted string, got: %s", output)
	}
}

func TestGlobalInfoKV(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	// Act: логируем с парами ключ-значение.
	logger.InfoKV(ctx, "test kv", "key1", "value1", "key2", 42)

	// Assert: сообщение и ключи/значения присутствуют.
	output := buf.String()
	if !strings.Contains(output, "test kv") {
		t.Errorf("output should contain message, got: %s", output)
	}
	if !strings.Contains(output, "key1") || !strings.Contains(output, "value1") {
		t.Errorf("output should contain key-value pairs, got: %s", output)
	}
	if !strings.Contains(output, "key2") {
		t.Errorf("output should contain key2, got: %s", output)
	}
}

func TestGlobalLevels(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()

	// Таблица: каждая глобальная функция пишет сообщение со своим уровнем.
	tests := []struct {
		name  string
		logFn func(ctx context.Context, msg string)
		level string
	}{
		{"Debug", logger.Debug, "DEBUG"},
		{"Info", logger.Info, "INFO"},
		{"Warn", logger.Warn, "WARN"},
		{"Error", logger.Error, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn(ctx, "level test "+tt.name)
			output := buf.String()
			if !strings.Contains(output, "level test "+tt.name) {
				t.Errorf("output should contain message, got: %s", output)
			}
			// Вывод содержит имя уровня заглавными буквами.
			if !strings.Contains(output, tt.level) {
				t.Errorf("output should contain level %s, got: %s", tt.level, output)
			}
		})
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	// Arrange: глобальный уровень Warn — сообщения ниже уровня не пишутся.
	logger.Init(logger.LevelWarn, &buf)

	ctx := context.Background()

	// Act: пишем сообщения разных уровней.
	logger.Debug(ctx, "should be filtered")
	logger.Info(ctx, "should be filtered too")
	logger.Warn(ctx, "should appear")
	logger.Error(ctx, "should also appear")

	// Assert: debug/info отфильтрованы, warn/error записаны.
	output := buf.String()
	if strings.Contains(output, "should be filtered") {
		t.Errorf("debug message should be filtered out, got: %s", output)
	}
	if !strings.Contains(output, "should appear") {
		t.Errorf("warn message should appear, got: %s", output)
	}
	if !strings.Contains(output, "should also appear") {
		t.Errorf("error message should appear, got: %s", output)
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	lvl := logger.LevelError
	logger.Init(lvl, &buf)

	ctx := context.Background()

	// С уровнем error debug-сообщения не проходят.
	logger.Debug(ctx, "debug before setlevel")
	if strings.Contains(buf.String(), "debug before setlevel") {
		t.Errorf("debug message should be filtered out at error level, got: %s", buf.String())
	}

	// Меняем уровень на debug.
	logger.SetLevel(logger.LevelDebug)

	logger.Debug(ctx, "debug after setlevel")
	if !strings.Contains(buf.String(), "debug after setlevel") {
		t.Errorf("debug message should appear after SetLevel, got: %s", buf.String())
	}
}

func TestCurrentLevel(t *testing.T) {
	var buf bytes.Buffer
	// Arrange: глобальный уровень Warn.
	logger.Init(logger.LevelWarn, &buf)

	// Act+Assert: CurrentLevel возвращает установленный уровень.
	if got := logger.CurrentLevel(); got != logger.LevelWarn {
		t.Errorf("CurrentLevel() = %v, want %v", got, logger.LevelWarn)
	}
}

func TestGlobalDebugfFormatted(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	// Act: форматированное debug-сообщение.
	logger.Debugf(ctx, "debug %s %d", "formatted", 1)

	if !strings.Contains(buf.String(), "debug formatted 1") {
		t.Errorf("output should contain formatted message, got: %s", buf.String())
	}
}

func TestGlobalWarnfFormatted(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	logger.Warnf(ctx, "warn %s", "formatted")

	if !strings.Contains(buf.String(), "warn formatted") {
		t.Errorf("output should contain formatted message, got: %s", buf.String())
	}
}

func TestGlobalErrorfFormatted(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	logger.Errorf(ctx, "error %s", "formatted")

	if !strings.Contains(buf.String(), "error formatted") {
		t.Errorf("output should contain formatted message, got: %s", buf.String())
	}
}

func TestGlobalDebugKV(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	logger.DebugKV(ctx, "debug with keys", "k1", "v1")

	if !strings.Contains(buf.String(), "debug with keys") {
		t.Errorf("output should contain message, got: %s", buf.String())
	}
	// Ключ и значение записаны в вывод.
	if !strings.Contains(buf.String(), "k1") || !strings.Contains(buf.String(), "v1") {
		t.Errorf("output should contain keys, got: %s", buf.String())
	}
}

func TestGlobalWarnKV(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	logger.WarnKV(ctx, "warn with keys", "k1", "v1", "k2", 2)

	output := buf.String()
	if !strings.Contains(output, "warn with keys") {
		t.Errorf("output should contain message, got: %s", output)
	}
	if !strings.Contains(output, "k1") {
		t.Errorf("output should contain k1, got: %s", output)
	}
}

func TestGlobalErrorKV(t *testing.T) {
	var buf bytes.Buffer
	logger.Init(logger.LevelDebug, &buf)

	ctx := context.Background()
	logger.ErrorKV(ctx, "error with keys", "err", "something went wrong")

	output := buf.String()
	if !strings.Contains(output, "error with keys") {
		t.Errorf("output should contain message, got: %s", output)
	}
	if !strings.Contains(output, "something went wrong") {
		t.Errorf("output should contain value, got: %s", output)
	}
}

func TestOutputToWriter(t *testing.T) {
	var buf bytes.Buffer
	// Arrange: экземпляр логгера, пишущий в буфер.
	logger := logger.New(logger.LevelDebug, &buf)

	ctx := context.Background()
	logger.Info(ctx, "writer output test")

	// Assert: вывод попадает в указанный writer.
	if !strings.Contains(buf.String(), "writer output test") {
		t.Errorf("output should contain message, got: %s", buf.String())
	}
}

func TestLoggerStructMethods(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	lvl := logger.LevelDebug
	logger := logger.New(lvl, &buf)

	// Проверка, что методы не паникуют и пишут в буфер.
	ctx := context.Background()
	logger.Debug(ctx, "debug")
	logger.Debugf(ctx, "debug %s", "formatted")
	logger.DebugKV(ctx, "debug kv", "key", "value")
	logger.Info(ctx, "info")
	logger.Infof(ctx, "info %s", "formatted")
	logger.InfoKV(ctx, "info kv", "key", "value")
	logger.Warn(ctx, "warn")
	logger.Warnf(ctx, "warn %s", "formatted")
	logger.WarnKV(ctx, "warn kv", "key", "value")
	logger.Error(ctx, "error")
	logger.Errorf(ctx, "error %s", "formatted")
	logger.ErrorKV(ctx, "error kv", "key", "value")

	if buf.Len() == 0 {
		t.Error("expected log output, got empty buffer")
	}
}
