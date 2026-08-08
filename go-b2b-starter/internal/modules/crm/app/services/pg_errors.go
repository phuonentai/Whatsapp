package services

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolationOn(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.SQLState() == "23505" && pgErr.ConstraintName == constraint
}
