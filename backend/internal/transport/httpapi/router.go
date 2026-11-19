// Package httpapi là tầng vận chuyển HTTP: router chi, middleware, handler, ánh xạ lỗi.
package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Deps là các phụ thuộc router cần, inject từ main.
type Deps struct {
	DB         Pinger
	CORSOrigin string
}

// NewRouter dựng toàn bộ route của API.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestIDHeader)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors(d.CORSOrigin))
	r.Use(middleware.Timeout(30 * time.Second))

	h := &healthHandler{db: d.DB, pingTimeout: 2 * time.Second}
	r.Get("/healthz", h.healthz)
	r.Get("/readyz", h.readyz)

	r.Route("/api/v1", func(_ chi.Router) {
		// Các route nghiệp vụ được thêm ở các nhánh tính năng.
	})

	return r
}
