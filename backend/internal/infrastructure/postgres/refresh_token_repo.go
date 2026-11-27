package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/auth"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// RefreshTokenRepo cài đặt auth.RefreshTokenStore.
type RefreshTokenRepo struct{ q sqlcgen.Querier }

var _ auth.RefreshTokenStore = (*RefreshTokenRepo)(nil)

// NewRefreshTokenRepo tạo RefreshTokenRepo.
func NewRefreshTokenRepo(db sqlcgen.DBTX) *RefreshTokenRepo {
	return &RefreshTokenRepo{q: sqlcgen.New(db)}
}

func (r *RefreshTokenRepo) Insert(ctx context.Context, t domain.RefreshToken) error {
	if err := r.q.InsertRefreshToken(ctx, sqlcgen.InsertRefreshTokenParams{
		ID: t.ID, UserID: t.UserID, FamilyID: t.FamilyID, TokenHash: t.TokenHash,
		ExpiresAt: ts(t.ExpiresAt), CreatedAt: ts(t.CreatedAt),
	}); err != nil {
		return fmt.Errorf("postgres.RefreshTokenRepo.Insert: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByHash(ctx context.Context, h []byte) (domain.RefreshToken, error) {
	row, err := r.q.GetRefreshTokenByHash(ctx, h)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, domain.ErrRefreshTokenNotFound
	}
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("postgres.RefreshTokenRepo.GetByHash: %w", err)
	}
	return domain.RefreshToken{
		ID: row.ID, UserID: row.UserID, FamilyID: row.FamilyID, TokenHash: h,
		ExpiresAt: row.ExpiresAt.Time, RevokedAt: timePtr(row.RevokedAt),
	}, nil
}

func (r *RefreshTokenRepo) RevokeIfActive(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	n, err := r.q.RevokeRefreshTokenIfActive(ctx, sqlcgen.RevokeRefreshTokenIfActiveParams{ID: id, RevokedAt: ts(at)})
	if err != nil {
		return false, fmt.Errorf("postgres.RefreshTokenRepo.RevokeIfActive: %w", err)
	}
	return n == 1, nil
}

func (r *RefreshTokenRepo) RevokeFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error {
	if err := r.q.RevokeRefreshTokenFamily(ctx, sqlcgen.RevokeRefreshTokenFamilyParams{FamilyID: familyID, RevokedAt: ts(at)}); err != nil {
		return fmt.Errorf("postgres.RefreshTokenRepo.RevokeFamily: %w", err)
	}
	return nil
}
