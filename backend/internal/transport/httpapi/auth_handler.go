package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/auth"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// AuthService là các use case xác thực mà handler cần (cài đặt: auth.Service).
type AuthService interface {
	Login(ctx context.Context, email, password string) (auth.Session, error)
	Refresh(ctx context.Context, raw string) (auth.Session, error)
	Logout(ctx context.Context, raw string) error
	Me(ctx context.Context, p authz.Principal) (domain.User, error)
	InspectActivation(ctx context.Context, raw string) (domain.User, error)
	Activate(ctx context.Context, raw, password string) error
}

const (
	refreshCookieName = "wt_refresh"
	refreshCookiePath = "/api/v1/auth"
)

type authHandler struct {
	svc      AuthService
	validate *validator.Validate
	now      func() time.Time
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,max=72"`
}

type activationTokenRequest struct {
	Token string `json:"token" validate:"required,max=256"`
}

type activateRequest struct {
	Token    string `json:"token" validate:"required,max=256"`
	Password string `json:"password" validate:"required,max=72"`
}

type sessionResponse struct {
	AccessToken string       `json:"accessToken"`
	ExpiresAt   time.Time    `json:"expiresAt"`
	User        userResponse `json:"user"`
}

type activationInfoResponse struct {
	Email    string `json:"email"`
	FullName string `json:"fullName"`
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	s, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.writeSession(w, s)
}

func (h *authHandler) refresh(w http.ResponseWriter, r *http.Request) {
	raw := ""
	if c, err := r.Cookie(refreshCookieName); err == nil {
		raw = c.Value
	}
	s, err := h.svc.Refresh(r.Context(), raw)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthenticated) {
			h.clearRefreshCookie(w)
		}
		writeError(w, r, err)
		return
	}
	h.writeSession(w, s)
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookieName); err == nil {
		if err := h.svc.Logout(r.Context(), c.Value); err != nil {
			writeError(w, r, err)
			return
		}
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r.Context())
	if !ok {
		writeError(w, r, domain.ErrUnauthenticated)
		return
	}
	u, err := h.svc.Me(r.Context(), p)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

// verifyActivation dùng POST để token không nằm trên URL/log của API.
func (h *authHandler) verifyActivation(w http.ResponseWriter, r *http.Request) {
	var req activationTokenRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	u, err := h.svc.InspectActivation(r.Context(), req.Token)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, activationInfoResponse{Email: u.Email, FullName: u.FullName})
}

func (h *authHandler) activate(w http.ResponseWriter, r *http.Request) {
	var req activateRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.svc.Activate(r.Context(), req.Token, req.Password); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) writeSession(w http.ResponseWriter, s auth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    s.RefreshToken,
		Path:     refreshCookiePath,
		Expires:  s.RefreshExpiresAt,
		MaxAge:   int(s.RefreshExpiresAt.Sub(h.now()).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, sessionResponse{
		AccessToken: s.AccessToken,
		ExpiresAt:   s.AccessExpiresAt,
		User:        toUserResponse(s.User),
	})
}

func (h *authHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
