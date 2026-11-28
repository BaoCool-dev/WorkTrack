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
	Projects ProjectService
	Tasks    TaskService
	Labels   LabelService
	Settings SettingService
	Kanban   KanbanService
	// CF-07
	Subtasks      SubtaskService
	Comments      CommentService
	Notifications NotificationService
	Hub           BoardHub
	Tokens        TokenVerifier
	// CF-08
	TimeLogs TimeLogService
	// CF-09
	Reviews ReviewService

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

	// WebSocket sống lâu (hàng giờ) nên KHÔNG nằm dưới middleware.Timeout — mọi route còn lại có Timeout 30s.
	// Tự xác thực bằng access_token trên query (trình duyệt không gửi được header khi mở WebSocket).
	wsh := &wsHandler{hub: d.Hub, projects: d.Projects, tokens: d.Tokens, now: d.Now, origins: wsOriginPatterns(d.CORSOrigin)}
	r.Get("/api/v1/projects/{projectID}/ws", wsh.serve)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		mountHTTP(r, d)
	})
	return r
}

// mountHTTP gắn các route HTTP thường (có Timeout).
func mountHTTP(r chi.Router, d Deps) {
	h := &healthHandler{db: d.DB, pingTimeout: 2 * time.Second}
	r.Get("/healthz", h.healthz)
	r.Get("/readyz", h.readyz)

	v := newValidator()
	ah := &authHandler{svc: d.Auth, validate: v, now: d.Now}
	uh := &userHandler{svc: d.Accounts, validate: v}
	oh := &orgHandler{svc: d.Org, validate: v}
	ph := &projectHandler{svc: d.Projects, validate: v}
	th := &taskHandler{svc: d.Tasks, labels: d.Labels, settings: d.Settings, validate: v}
	kh := &kanbanHandler{svc: d.Kanban, projects: d.Projects, validate: v}
	ch := &collabHandler{subtasks: d.Subtasks, comments: d.Comments, notifications: d.Notifications, validate: v}
	tlh := &timeLogHandler{svc: d.TimeLogs, validate: v}
	rh := &reviewHandler{svc: d.Reviews, validate: v}
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

			// CF-03: dự án. Quyền theo vai trò + Manager phụ trách kiểm trong project.Service.
			r.Route("/projects", func(r chi.Router) {
				r.Get("/", ph.list)
				r.Post("/", ph.create)
				r.Route("/{projectID}", func(r chi.Router) {
					r.Get("/", ph.get)
					r.Put("/", ph.update)
					r.Get("/members", ph.listMembers)
					r.Post("/members", ph.addMember)
					r.Delete("/members/{userID}", ph.removeMember)
					r.Get("/member-candidates", ph.listCandidates)
					r.Get("/board", ph.getBoard)
					r.Post("/columns", ph.addColumn)
					r.Put("/columns/{columnID}", ph.updateColumn)
					r.Delete("/columns/{columnID}", ph.deleteColumn)
					r.Put("/column-order", ph.reorderColumns)
					r.Put("/transitions", ph.replaceTransitions)
					// CF-04, CF-05: bảng Kanban (toàn bộ việc cha) và giới hạn WIP của dự án.
					r.Get("/board/tasks", kh.boardTasks)
					r.Put("/wip-limit", kh.setWIPLimit)
					r.Get("/tasks", th.listProjectTasks)
					r.Post("/tasks", th.createTask)
				})
			})

			// CF-06: công việc cha, nhãn, tham số B1. Quyền kiểm trong service (IDOR → 404).
			r.Route("/tasks/{taskID}", func(r chi.Router) {
				r.Get("/", th.getTask)
				r.Put("/", th.updateTask)
				r.Post("/claim", th.claimTask)
				r.Post("/move", kh.move) // CF-04: kéo thả (luật chuyển cột, WIP, version kiểm trong kanban.Service)
				// CF-07: việc con 1 cấp và bình luận của một công việc.
				r.Get("/subtasks", ch.listSubtasks)
				r.Post("/subtasks", ch.createSubtask)
				r.Get("/comments", ch.listComments)
				r.Post("/comments", ch.createComment)
				// CF-08: Log time (cha và con); GET của việc cha gồm cả log của việc con.
				r.Get("/time-logs", tlh.list)
				r.Post("/time-logs", tlh.create)
				// CF-09: nghiệm thu — chỉ Manager phụ trách, chỉ khi thẻ ở cột "Chờ nghiệm thu".
				r.Post("/review/approve", rh.approve)
				r.Post("/review/reject", rh.reject)
				r.Get("/reviews", rh.list)
			})
			r.Delete("/time-logs/{logID}", tlh.delete)
			r.Post("/subtasks/{subtaskID}/move", ch.moveSubtask)
			r.Delete("/subtasks/{subtaskID}", ch.deleteSubtask)
			// CF-07: chuông thông báo — luôn là thông báo của chính người gọi.
			r.Get("/me/notifications", ch.listNotifications)
			r.Get("/me/notifications/unread-count", ch.unreadCount)
			r.Post("/me/notifications/read-all", ch.markAllRead)
			r.Post("/me/notifications/{notificationID}/read", ch.markRead)
			r.Get("/me/tasks", th.listMyTasks)
			r.Get("/me/wip", th.myWIP)
			r.Get("/labels", th.listLabels)
			r.Post("/labels", th.createLabel)
			r.Put("/labels/{labelID}", th.updateLabel)
			r.Delete("/labels/{labelID}", th.deleteLabel)
			r.Get("/settings", th.getSettings)
			r.Put("/settings", th.updateSettings)
		})
	})
}
