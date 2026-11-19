package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Pinger là thứ /readyz cần để kiểm tra CSDL (*pgxpool.Pool thỏa mãn).
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthHandler struct {
	db          Pinger
	pingTimeout time.Duration
}

type statusResponse struct {
	Status string `json:"status"`
}

// healthz: tiến trình còn sống, không chạm CSDL.
func (h *healthHandler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

// readyz: sẵn sàng nhận request — CSDL phải ping được.
func (h *healthHandler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.pingTimeout)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		slog.WarnContext(r.Context(), "readiness check failed", "err", err)
		writeError(w, r, fmt.Errorf("ping database: %w: %w", errDBUnavailable, err))
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
}
