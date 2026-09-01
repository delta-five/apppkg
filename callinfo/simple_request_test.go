package callinfo_test

import (
	"encoding/hex"
	"testing"

	"github.com/delta-five/apppkg/callinfo"
)

func TestMakeSimpleRequest(t *testing.T) {
	t.Parallel()

	// Act: создаём SimpleRequest со сгенерированным ID.
	r := callinfo.MakeSimpleRequest()

	// Assert: ID должен быть непустым.
	if r.ID() == "" {
		t.Error("ID должен быть непустым")
	}

	// Attrs возвращает ровно один атрибут — id.
	attrs := r.Attrs()
	if len(attrs) != 1 {
		t.Fatalf("Attrs() вернул %d атрибутов, want 1", len(attrs))
	}
	if attrs[0].Key != "id" {
		t.Errorf("Attrs()[0].Key = %q, want %q", attrs[0].Key, "id")
	}
	// Значение атрибута совпадает с ID запроса.
	if attrs[0].Value.String() != r.ID() {
		t.Errorf("Attrs()[0].Value = %q, want %q", attrs[0].Value.String(), r.ID())
	}
}

func TestGenerateID(t *testing.T) {
	t.Parallel()

	// Act: генерируем ID.
	id := callinfo.GenerateID()

	// Assert: длина 16 символов (8 байт в hex).
	if len(id) != 16 {
		t.Errorf("GenerateID() длина = %d, want 16", len(id))
	}

	// ID — корректная hex-строка.
	if _, err := hex.DecodeString(id); err != nil {
		t.Errorf("GenerateID() вернул не hex-строку: %v", err)
	}

	// Два подряд сгенерированных ID должны различаться.
	other := callinfo.GenerateID()
	if id == other {
		t.Error("GenerateID() вернул одинаковые значения подряд")
	}
}
