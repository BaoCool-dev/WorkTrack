---
name: go-backend
description: Mẫu code Go cho backend WorkTrack theo Clean Architecture với chi, pgx v5, sqlc, validator v10, slog. Dùng khi viết hoặc sửa handler HTTP, middleware, use case, port, repository, ánh xạ lỗi, wiring trong main.go.
---

# Go backend — mẫu chuẩn của WorkTrack

Quy tắc phụ thuộc và cấu trúc thư mục: xem `backend/CLAUDE.md`. Skill này chỉ đưa mẫu.

## Domain — quy tắc thuần, không I/O

```go
// internal/domain/task.go
package domain

var (
    ErrWIPLimitReached   = errors.New("wip limit reached")
    ErrForbidden         = errors.New("forbidden")
    ErrTaskNotFound      = errors.New("task not found")
    ErrLogTimeNegative   = errors.New("log time must be positive")
    ErrLogTimeTooLate    = errors.New("cannot log time 24h after done")
)

// CanLogTime là hàm thuần — test table-driven, không cần CSDL.
func CanLogTime(minutes int, doneAt *time.Time, now time.Time) error {
    if minutes <= 0 {
        return ErrLogTimeNegative
    }
    if doneAt != nil && now.Sub(*doneAt) > 24*time.Hour {
        return ErrLogTimeTooLate
    }
    return nil
}
```

## Application — use case + port khai báo ở nơi dùng

```go
// internal/application/timesheet/log_time.go
package timesheet

type TaskReader interface {
    GetTask(ctx context.Context, id uuid.UUID) (domain.Task, error)
}
type TimeLogWriter interface {
    InsertTimeLog(ctx context.Context, in domain.TimeLog) (domain.TimeLog, error)
}

type LogTime struct {
    tasks TaskReader
    logs  TimeLogWriter
    authz authz.Checker
    clock func() time.Time
}

func NewLogTime(t TaskReader, l TimeLogWriter, a authz.Checker, clock func() time.Time) *LogTime {
    return &LogTime{tasks: t, logs: l, authz: a, clock: clock}
}

type LogTimeInput struct {
    ActorID uuid.UUID
    TaskID  uuid.UUID
    Minutes int
    Note    string
}

func (uc *LogTime) Execute(ctx context.Context, in LogTimeInput) (domain.TimeLog, error) {
    task, err := uc.tasks.GetTask(ctx, in.TaskID)
    if err != nil {
        return domain.TimeLog{}, fmt.Errorf("get task: %w", err)
    }
    if err := uc.authz.Can(ctx, in.ActorID, authz.ActionLogTime, task.ProjectID); err != nil {
        return domain.TimeLog{}, err
    }
    if err := domain.CanLogTime(in.Minutes, task.DoneAt, uc.clock()); err != nil {
        return domain.TimeLog{}, err
    }
    return uc.logs.InsertTimeLog(ctx, domain.TimeLog{TaskID: task.ID, UserID: in.ActorID, Minutes: in.Minutes, Note: in.Note})
}
```

`clock` được inject để test được mốc 24 giờ.

## Transport — handler chi mỏng

```go
// internal/transport/httpapi/timesheet_handler.go
type logTimeRequest struct {
    Minutes int    `json:"minutes" validate:"required,gt=0,lte=1440"`
    Note    string `json:"note"    validate:"max=500"`
}

func (h *TimesheetHandler) LogTime(w http.ResponseWriter, r *http.Request) {
    taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
    if err != nil {
        writeError(w, r, errBadRequest("TASK_ID_INVALID", "Mã công việc không hợp lệ"))
        return
    }
    var req logTimeRequest
    if err := decodeAndValidate(r, &req, h.validate); err != nil {
        writeError(w, r, err)
        return
    }
    out, err := h.logTime.Execute(r.Context(), timesheet.LogTimeInput{
        ActorID: auth.UserID(r.Context()), TaskID: taskID, Minutes: req.Minutes, Note: req.Note,
    })
    if err != nil {
        writeError(w, r, err)
        return
    }
    writeJSON(w, http.StatusCreated, toTimeLogResponse(out))
}
```

Route: nhóm theo tài nguyên trong `router.go`, middleware xác thực áp cho cả nhóm.

```go
r.Route("/api/v1", func(r chi.Router) {
    r.Post("/auth/login", h.Auth.Login)            // công khai
    r.Group(func(r chi.Router) {
        r.Use(mw.RequireAuth(jwtVerifier))
        r.Route("/tasks/{taskID}", func(r chi.Router) {
            r.Post("/time-logs", h.Timesheet.LogTime)
        })
    })
})
```

## Ánh xạ lỗi — một chỗ duy nhất

```go
// internal/transport/httpapi/errors.go
func writeError(w http.ResponseWriter, r *http.Request, err error) {
    var he *httpError
    switch {
    case errors.As(err, &he):
    case errors.Is(err, domain.ErrForbidden):
        he = &httpError{http.StatusForbidden, "FORBIDDEN", "Bạn không có quyền thực hiện thao tác này"}
    case errors.Is(err, domain.ErrTaskNotFound):
        he = &httpError{http.StatusNotFound, "TASK_NOT_FOUND", "Không tìm thấy công việc"}
    case errors.Is(err, domain.ErrWIPLimitReached):
        he = &httpError{http.StatusConflict, "TASK_WIP_LIMIT", "Đã đủ số việc đang làm"}
    case errors.Is(err, domain.ErrLogTimeNegative):
        he = &httpError{http.StatusUnprocessableEntity, "LOGTIME_NEGATIVE", "Số giờ phải lớn hơn 0"}
    case errors.Is(err, domain.ErrLogTimeTooLate):
        he = &httpError{http.StatusUnprocessableEntity, "LOGTIME_TOO_LATE", "Không thể log giờ sau 24 giờ kể từ khi hoàn thành"}
    default:
        slog.ErrorContext(r.Context(), "unhandled error", "err", err)
        he = &httpError{http.StatusInternalServerError, "INTERNAL", "Lỗi hệ thống, vui lòng thử lại"}
    }
    writeJSON(w, he.status, map[string]any{"error": map[string]string{"code": he.code, "message": he.msg}})
}
```

Thêm lỗi nghiệp vụ mới → thêm vào `domain` + một case ở đây + test API cho mã lỗi đó.

## Infrastructure — repository bọc sqlc

```go
// internal/infrastructure/postgres/task_repo.go
type TaskRepo struct{ q *sqlcgen.Queries }

func (r *TaskRepo) GetTask(ctx context.Context, id uuid.UUID) (domain.Task, error) {
    row, err := r.q.GetTask(ctx, id)
    if errors.Is(err, pgx.ErrNoRows) {
        return domain.Task{}, domain.ErrTaskNotFound
    }
    if err != nil {
        return domain.Task{}, fmt.Errorf("query task: %w", err)
    }
    return toDomainTask(row), nil
}
```

Chuyển `pgx.ErrNoRows` thành lỗi domain **ở repository**, không để lọt lên use case.

## Test

```go
func TestCanLogTime(t *testing.T) {
    now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
    done := now.Add(-25 * time.Hour)
    tests := []struct {
        name    string
        minutes int
        doneAt  *time.Time
        want    error
    }{
        {"giờ dương, chưa done", 90, nil, nil},
        {"giờ bằng 0", 0, nil, domain.ErrLogTimeNegative},
        {"giờ âm", -30, nil, domain.ErrLogTimeNegative},
        {"done quá 24h", 30, &done, domain.ErrLogTimeTooLate},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := domain.CanLogTime(tt.minutes, tt.doneAt, now); !errors.Is(got, tt.want) {
                t.Fatalf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Checklist trước khi xong
- [ ] `ctx` là tham số đầu mọi hàm I/O.
- [ ] Không import `pgx`/`chi`/`net/http` trong `domain`/`application`.
- [ ] Phân quyền gọi `authz` trong use case, có test 403.
- [ ] Lỗi mới có mã + thông điệp tiếng Việt trong `errors.go`.
- [ ] `golangci-lint run` sạch.
