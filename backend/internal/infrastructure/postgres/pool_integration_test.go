//go:build integration

package postgres

import (
	"testing"
)

// testDSN: TEST_DATABASE_URL nếu có (máy không Docker, B12), ngược lại testcontainers (CI).
func TestNewPool_RealPostgres_Pings(t *testing.T) {
	ctx := t.Context()
	dsn := testDSN(t)

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
