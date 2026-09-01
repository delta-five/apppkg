package app_error_test

import (
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/delta-five/apppkg/app_error"
)

// Проверка fallback-ветки HTTPStatus для gRPC status-ошибок,
// которые не являются AppError.
func TestHTTPStatus_GRPCStatusFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"NotFound", status.Error(codes.NotFound, "not found"), http.StatusNotFound},
		{"PermissionDenied", status.Error(codes.PermissionDenied, "denied"), http.StatusForbidden},
		{"Unauthenticated", status.Error(codes.Unauthenticated, "auth"), http.StatusUnauthorized},
		{"Internal", status.Error(codes.Internal, "internal"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Проверяем fallback: gRPC status-ошибка маппится в HTTP без AppError.
			if got := app_error.HTTPStatus(tt.err); got != tt.want {
				t.Errorf("HTTPStatus(%v) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}
