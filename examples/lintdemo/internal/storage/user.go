// Package storage is the one place allowed to hold a database handle.
package storage

import (
	"context"
	"database/sql"

	"example.com/lintdemo/internal/services"
)

type UserStore struct{ DB *sql.DB }

func (s UserStore) ByID(ctx context.Context, id int64) (services.User, error) {
	var u services.User
	err := s.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE id=$1", id).Scan(&u.ID)
	return u, err
}
