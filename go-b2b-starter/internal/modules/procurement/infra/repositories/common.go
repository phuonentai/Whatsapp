package repositories

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// isNoRows reports whether err is pgx.ErrNoRows (single-row query miss).
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// pgInt8Ptr converts a nullable BIGINT to a *int64.
func pgInt8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

// toPgFloat8 converts a *float64 to a nullable DOUBLE PRECISION.
func toPgFloat8(v *float64) pgtype.Float8 {
	if v == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *v, Valid: true}
}

// pgFloat8Ptr converts a nullable DOUBLE PRECISION to a *float64.
func pgFloat8Ptr(v pgtype.Float8) *float64 {
	if !v.Valid {
		return nil
	}
	out := v.Float64
	return &out
}
