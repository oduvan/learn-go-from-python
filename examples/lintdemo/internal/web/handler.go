// Package web is an upper layer. The *sql.DB field below is a
// deliberate depguard violation, used to prove the rule fires.
package web

import (
	"context"
	"database/sql"

	"example.com/lintdemo/internal/services"
)

type Handler struct {
	Users services.UserStore
	DB    *sql.DB
}

func (h Handler) Get(ctx context.Context, id int64) (services.User, error) {
	return h.Users.ByID(ctx, id)
}
