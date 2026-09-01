// Package callinfo предоставляет структуру CallInfo для передачи служебной информации
// о запросе через контекст (например, ID запроса, IP-адрес источника, имя метода)
// и добавления её в структурированный лог.
package callinfo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

// CallInfo — интерфейс служебной информации о запросе, предоставляющий
// атрибуты для добавления в структурированный лог.
type CallInfo interface {
	Attrs() []slog.Attr
}

// SimpleRequest — минимальная реализация CallInfo, хранящая ID запроса.
type SimpleRequest struct {
	id string
}

var _ CallInfo = (*SimpleRequest)(nil)

// MakeSimpleRequest создаёт SimpleRequest со сгенерированным ID запроса.
func MakeSimpleRequest() SimpleRequest {
	return SimpleRequest{
		id: GenerateID(),
	}
}

// Attrs возвращает атрибуты запроса для логирования (id).
func (r SimpleRequest) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("id", r.id),
	}
}

// ID возвращает идентификатор запроса.
func (r SimpleRequest) ID() string {
	return r.id
}

type contextKey struct{}

// WithCallInfo помещает CallInfo в контекст.
func WithCallInfo(ctx context.Context, ci CallInfo) context.Context {
	return context.WithValue(ctx, contextKey{}, ci)
}

// FromContext извлекает CallInfo из контекста.
// Возвращает nil, если информация не была сохранена.
func FromContext(ctx context.Context) CallInfo {
	ci, _ := ctx.Value(contextKey{}).(CallInfo)
	return ci
}

// GenerateID генерирует короткий уникальный ID запроса (16 hex-символов).
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
