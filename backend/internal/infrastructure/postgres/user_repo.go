package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/auth"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// UserRepo cài đặt auth.UserStore và account.UserStore.
type UserRepo struct{ q sqlcgen.Querier }

var (
	_ auth.UserStore    = (*UserRepo)(nil)
	_ account.UserStore = (*UserRepo)(nil)
)

// NewUserRepo tạo UserRepo từ một DBTX (pool hoặc transaction).
func NewUserRepo(db sqlcgen.DBTX) *UserRepo { return &UserRepo{q: sqlcgen.New(db)} }

// Ba truy vấn Get* trả cùng tập cột; sqlc sinh ba kiểu có cấu trúc giống hệt nên đổi kiểu được.
func toUser(r sqlcgen.GetUserByIDRow) domain.User {
	return domain.User{
		ID: r.ID, Email: r.Email, FullName: r.FullName,
		Role: domain.Role(r.RoleCode), Status: domain.UserStatus(r.Status),
		PasswordHash:        r.PasswordHash.String,
		FailedLoginCount:    int(r.FailedLoginCount),
		LockedUntil:         timePtr(r.LockedUntil),
		ActivationExpiresAt: timePtr(r.ActivationExpiresAt),
		LastLoginAt:         timePtr(r.LastLoginAt),
		CreatedAt:           r.CreatedAt.Time,
	}
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("postgres.UserRepo: %w", err)
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, notFound(err)
	}
	return toUser(row), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, notFound(err)
	}
	return toUser(sqlcgen.GetUserByIDRow(row)), nil
}

func (r *UserRepo) GetByActivationTokenHash(ctx context.Context, h []byte) (domain.User, error) {
	row, err := r.q.GetUserByActivationTokenHash(ctx, h)
	if err != nil {
		return domain.User{}, notFound(err)
	}
	return toUser(sqlcgen.GetUserByIDRow(row)), nil
}

func (r *UserRepo) Activate(ctx context.Context, id uuid.UUID, h []byte, passwordHash string, now time.Time) error {
	n, err := r.q.ActivateUser(ctx, sqlcgen.ActivateUserParams{
		ID: id, TokenHash: h, PasswordHash: pgtype.Text{String: passwordHash, Valid: true}, Now: ts(now),
	})
	if err != nil {
		return fmt.Errorf("postgres.UserRepo.Activate: %w", err)
	}
	if n == 0 {
		return domain.ErrActivationTokenInvalid
	}
	return nil
}

// RegisterLoginFailure tăng bộ đếm trong một câu UPDATE nên các request song song không ghi đè nhau (B8).
func (r *UserRepo) RegisterLoginFailure(ctx context.Context, id uuid.UUID, maxFailures int, lockUntil time.Time) (*time.Time, error) {
	locked, err := r.q.RegisterLoginFailure(ctx, sqlcgen.RegisterLoginFailureParams{
		ID: id, MaxFailures: int32(min(maxFailures, domain.MaxFailedLogins)), //nolint:gosec // chặn trên bởi MaxFailedLogins
		LockUntil: ts(lockUntil),
	})
	if err != nil {
		return nil, fmt.Errorf("postgres.UserRepo.RegisterLoginFailure: %w", notFound(err))
	}
	return timePtr(locked), nil
}

// RecordLoginSuccess không cập nhật nếu một request song song vừa khóa tài khoản.
func (r *UserRepo) RecordLoginSuccess(ctx context.Context, id uuid.UUID, at time.Time) error {
	n, err := r.q.RecordLoginSuccess(ctx, sqlcgen.RecordLoginSuccessParams{ID: id, LoggedInAt: ts(at)})
	if err != nil {
		return fmt.Errorf("postgres.UserRepo.RecordLoginSuccess: %w", err)
	}
	if n == 0 {
		return domain.ErrAccountLocked
	}
	return nil
}

func (r *UserRepo) Create(ctx context.Context, u account.NewUser) (uuid.UUID, error) {
	id, err := r.q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email: u.Email, FullName: u.FullName, RoleCode: string(u.Role),
		ActivationTokenHash: u.ActivationTokenHash, ActivationExpiresAt: ts(u.ActivationExpiresAt),
		CreatedBy: uuid.NullUUID{UUID: u.CreatedBy, Valid: u.CreatedBy != uuid.Nil},
	})
	if isUniqueViolation(err) {
		return uuid.Nil, domain.ErrEmailTaken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres.UserRepo.Create: %w", err)
	}
	return id, nil
}

func (r *UserRepo) List(ctx context.Context, p domain.PageRequest) ([]domain.User, int, error) {
	total, err := r.q.CountUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.UserRepo.List: count: %w", err)
	}
	rows, err := r.q.ListUsers(ctx, sqlcgen.ListUsersParams{
		PageLimit: int32(p.PageSize), PageOffset: int32(p.Offset()), //nolint:gosec // PageSize ≤ MaxPageSize; offset giới hạn bởi NewPageRequest
	})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.UserRepo.List: %w", err)
	}
	out := make([]domain.User, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.User{
			ID: x.ID, Email: x.Email, FullName: x.FullName,
			Role: domain.Role(x.RoleCode), Status: domain.UserStatus(x.Status),
			LastLoginAt: timePtr(x.LastLoginAt), CreatedAt: x.CreatedAt.Time,
		})
	}
	return out, int(total), nil
}

func (r *UserRepo) SetActivationToken(ctx context.Context, id uuid.UUID, h []byte, exp time.Time) error {
	n, err := r.q.SetActivationToken(ctx, sqlcgen.SetActivationTokenParams{ID: id, TokenHash: h, ExpiresAt: ts(exp)})
	if err != nil {
		return fmt.Errorf("postgres.UserRepo.SetActivationToken: %w", err)
	}
	if n == 0 {
		return domain.ErrUserAlreadyActive
	}
	return nil
}
