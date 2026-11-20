// Package httpapi là tầng vận chuyển HTTP: router chi, middleware, handler, ánh xạ lỗi.
package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
)

// Deps là các phụ thuộc router cần, inject từ main.
type Deps struct {
	DB         Pinger
	CORSOrigin string

	Auth     AuthService
	Accounts AccountService
	Org      OrgService
	Tokens   TokenVerifier

	// Now mặc định time.Now; test truyền đồng hồ giả.
	Now func() time.Time
	// LoginLimit / RefreshLimit: giới hạn tần suất theo IP. Bỏ trống → mặc định.
	LoginLimit   RateLimit
	RefreshLimit RateLimit
}

var (
	defaultLoginLimit   = RateLimit{Burst: 10, Every: 6 * time.Second}
	defaultRefreshLimit = RateLimit{Burst: 30, Every: 2 * time.Second}
)

// NewRouter dựng toàn bộ route của API.
func NewRouter(d Deps) http.Handler {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.LoginLimit.Burst == 0 {
		d.LoginLimit = defaultLoginLimit
	}
	if d.RefreshLimit.Burst == 0 {
		d.RefreshLimit = defaultRefreshLimit
	}

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

	v := newValidator()
	ah := &authHandler{svc: d.Auth, validate: v, now: d.Now}
	uh := &userHandler{svc: d.Accounts, validate: v}
	oh := &orgHandler{svc: d.Org, validate: v}
	loginRL := newRateLimiter(d.LoginLimit, d.Now)
	refreshRL := newRateLimiter(d.RefreshLimit, d.Now)

	r.Route("/api/v1", func(r chi.Router) {
		// Công khai (có giới hạn tần suất).
		r.Route("/auth", func(r chi.Router) {
			r.With(loginRL.middleware).Post("/login", ah.login)
			r.With(loginRL.middleware).Post("/activation/verify", ah.verifyActivation)
			r.With(loginRL.middleware).Post("/activation", ah.activate)
			r.With(refreshRL.middleware).Post("/refresh", ah.refresh)
			r.Post("/logout", ah.logout)
		})

		// Cần đăng nhập.
		r.Group(func(r chi.Router) {
			r.Use(requireAuth(d.Tokens, d.Now))
			r.Get("/auth/me", ah.me)

			r.Route("/users", func(r chi.Router) {
				r.Use(requirePermission(authz.ManageUsers))
				r.Get("/", uh.list)
				r.Post("/", uh.create)
				r.Get("/{userID}", uh.get)
				r.Put("/{userID}", uh.update)
				r.Put("/{userID}/skills", uh.setSkills)
				r.Post("/{userID}/activation-email", uh.resendActivation)
			})

			// CF-02: cơ cấu tổ chức (Admin, CEO).
			r.Group(func(r chi.Router) {
				r.Use(requirePermission(authz.ManageOrg))
				r.Get("/departments", oh.listDepartments)
				r.Post("/departments", oh.createDepartment)
				r.Put("/departments/{id}", oh.updateDepartment)
				r.Delete("/departments/{id}", oh.deleteDepartment)
				r.Get("/job-titles", oh.listJobTitles)
				r.Post("/job-titles", oh.createJobTitle)
				r.Put("/job-titles/{id}", oh.updateJobTitle)
				r.Delete("/job-titles/{id}", oh.deleteJobTitle)
				r.Get("/skills", oh.listSkills)
				r.Post("/skills", oh.createSkill)
				r.Put("/skills/{id}", oh.updateSkill)
				r.Delete("/skills/{id}", oh.deleteSkill)
			})
		})
	})

	return r
}
