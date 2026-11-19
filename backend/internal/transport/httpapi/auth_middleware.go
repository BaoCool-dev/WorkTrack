package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// TokenVerifier kiểm tra access token và trả Principal (cài đặt: security.JWT).
type TokenVerifier interface {
	Verify(token string, now time.Time) (authz.Principal, error)
}

type principalKey struct{}

// principalFrom lấy người dùng đã xác thực từ context (chỉ có sau RequireAuth).
func principalFrom(ctx context.Context) (authz.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(authz.Principal)
	return p, ok
}

// requireAuth đọc "Authorization: Bearer <token>"; thiếu/sai/hết hạn → 401.
func requireAuth(v TokenVerifier, now func() time.Time) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
				writeError(w, r, domain.ErrUnauthenticated)
				return
			}
			p, err := v.Verify(token, now())
			if err != nil {
				writeError(w, r, domain.ErrUnauthenticated)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
		})
	}
}

// requirePermission chặn sớm (403) theo ma trận authz; use case vẫn tự kiểm tra lại.
func requirePermission(perm authz.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := principalFrom(r.Context())
			if !ok {
				writeError(w, r, domain.ErrUnauthenticated)
				return
			}
			if err := authz.Require(p, perm); err != nil {
				writeError(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
