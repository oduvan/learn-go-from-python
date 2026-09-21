// Package services declares store contracts. It must not import a
// database driver — depguard enforces that.
package services

import "context"

type User struct{ ID int64 }

type UserStore interface {
	ByID(ctx context.Context, id int64) (User, error)
}
