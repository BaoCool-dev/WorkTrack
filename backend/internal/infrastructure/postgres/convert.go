package postgres

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgUniqueViolation là SQLSTATE của vi phạm ràng buộc UNIQUE.
const pgUniqueViolation = "23505"

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

// pgFKViolation là SQLSTATE của vi phạm khóa ngoại.
const pgFKViolation = "23503"

// fkViolation trả tên ràng buộc khóa ngoại bị vi phạm, hoặc "" nếu err không phải lỗi đó.
func fkViolation(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgFKViolation {
		return pgErr.ConstraintName
	}
	return ""
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func uuidPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	id := v.UUID
	return &id
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

// likeContains thoát ký tự đặc biệt của LIKE (\ % _) rồi bọc %…% để tìm chứa chuỗi.
func likeContains(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func timePtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
