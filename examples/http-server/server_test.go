package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harshakonda/heapcheck/guard"
)

func TestHandleUserBad(t *testing.T) {
	defer guard.VerifyNone(t)

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	w := httptest.NewRecorder()

	HandleUserBad(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleUserGood(t *testing.T) {
	defer guard.VerifyNone(t)

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	w := httptest.NewRecorder()

	HandleUserGood(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleError(t *testing.T) {
	defer guard.VerifyNone(t)

	req := httptest.NewRequest(http.MethodGet, "/error", nil)

	w1 := httptest.NewRecorder()
	HandleErrorBad(w1, req, 404)

	w2 := httptest.NewRecorder()
	HandleErrorGood(w2, req, 404)
}

func TestLoggingMiddleware(t *testing.T) {
	defer guard.VerifyNone(t)

	logger := &Logger{prefix: "[TEST] "}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := NewLoggingMiddleware(logger, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCreateUserPooled(t *testing.T) {
	defer guard.VerifyNone(t)

	body := strings.NewReader(`{"id":1,"name":"test","email":"test@test.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/user", body)
	w := httptest.NewRecorder()

	CreateUserPooled(w, req)
}
