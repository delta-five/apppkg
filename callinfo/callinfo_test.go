package callinfo_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/delta-five/apppkg/callinfo"
)

type ciPayload struct {
	ID      string
	IP      string
	Handler string
}

// Проверяем, что ciPayload реализует интерфейс CallInfo на этапе компиляции.
var _ callinfo.CallInfo = (*ciPayload)(nil)

func (r ciPayload) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("id", r.ID),
		slog.String("ip", r.IP),
		slog.String("handler", r.Handler),
	}
}

func TestCallInfoRoundTrip(t *testing.T) {
	t.Parallel()

	// Arrange: произвольная реализация CallInfo с несколькими полями.
	ci := &ciPayload{
		ID:      "req-123",
		IP:      "192.168.1.1",
		Handler: "/api/v1/test",
	}

	// Act: сохраняем в контекст и извлекаем обратно.
	ctx := callinfo.WithCallInfo(context.Background(), ci)
	result := callinfo.FromContext(ctx)

	// Assert: значение не потерялось.
	assert.NotNil(t, result)

	// Атрибуты полностью совпадают с исходными.
	assert.Equal(t, []slog.Attr{
		slog.String("id", ci.ID),
		slog.String("ip", ci.IP),
		slog.String("handler", ci.Handler),
	}, result.Attrs())
}

func TestCallInfoFromEmptyContext(t *testing.T) {
	t.Parallel()

	// Act: извлекаем CallInfo из пустого контекста.
	result := callinfo.FromContext(context.Background())
	// Assert: ничего не сохранено — возвращается nil.
	assert.Nil(t, result)
}
