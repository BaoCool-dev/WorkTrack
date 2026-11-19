package domain_test

import (
	"testing"
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestNewPageRequest_Normalize_B9(t *testing.T) {
	cases := []struct{ page, size, wantPage, wantSize, wantOffset int }{
		{1, 20, 1, 20, 0},
		{0, 0, 1, 20, 0},
		{-3, -1, 1, 20, 0},
		{3, 10, 3, 10, 20},
		{2, 101, 2, 100, 100},
		{1, 100, 1, 100, 0},
		{5_000_000, 100, 1_000_000, 100, 99_999_900},
	}
	for _, c := range cases {
		p := domain.NewPageRequest(c.page, c.size)
		if p.Page != c.wantPage || p.PageSize != c.wantSize || p.Offset() != c.wantOffset {
			t.Errorf("NewPageRequest(%d,%d) = %+v offset %d", c.page, c.size, p, p.Offset())
		}
	}
}

func TestRefreshToken_ExpiredRevoked_CF01_B4(t *testing.T) {
	now := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	tok := domain.RefreshToken{ExpiresAt: now}
	if !tok.IsExpired(now) || tok.IsExpired(now.Add(-time.Second)) {
		t.Fatal("IsExpired sai ở biên")
	}
	if tok.IsRevoked() {
		t.Fatal("chưa thu hồi")
	}
	tok.RevokedAt = &now
	if !tok.IsRevoked() {
		t.Fatal("đã thu hồi")
	}
}
