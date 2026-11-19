package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Health check là hạ tầng (chore/scaffold), không gắn UC/AC.

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func doGet(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthz_ReturnsOK(t *testing.T) {
	// /healthz không phụ thuộc CSDL: DB lỗi vẫn trả 200.
	r := NewRouter(Deps{DB: fakePinger{err: errors.New("down")}})

	rec := doGet(t, r, "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body statusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status field = %q, want ok", body.Status)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("missing X-Request-Id header")
	}
}

func TestReadyz_DBUp_ReturnsReady(t *testing.T) {
	r := NewRouter(Deps{DB: fakePinger{}})

	rec := doGet(t, r, "/readyz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body statusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ready" {
		t.Errorf("status field = %q, want ready", body.Status)
	}
}

func TestReadyz_DBDown_Returns503(t *testing.T) {
	r := NewRouter(Deps{DB: fakePinger{err: errors.New("connection refused")}})

	rec := doGet(t, r, "/readyz")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body errorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "DB_UNAVAILABLE" {
		t.Errorf("code = %q, want DB_UNAVAILABLE", body.Error.Code)
	}
	if body.Error.Message == "" {
		t.Error("message must not be empty")
	}
}

func TestCORS_AllowedOrigin_SetsHeaders(t *testing.T) {
	const origin = "http://localhost:5173"
	r := NewRouter(Deps{DB: fakePinger{}, CORSOrigin: origin})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Errorf("allow-origin = %q, want %q", got, origin)
	}
}

func TestCORS_OtherOrigin_NoHeaders(t *testing.T) {
	r := NewRouter(Deps{DB: fakePinger{}, CORSOrigin: "http://localhost:5173"})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow-origin = %q, want empty", got)
	}
}
