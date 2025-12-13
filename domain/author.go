package domain

import (
	"context"
	"time"
)

// User representing the User data struct
type User struct {
	ID        int64
	Name      string
	Username  string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (User, error)
	Insert(ctx context.Context, a *User) error
	Update(ctx context.Context, a *User) error
	GetByUsername(ctx context.Context, username string) (User, error)
}

type UserUsecase interface {
	Register(ctx context.Context, a *User) error
	Login(ctx context.Context, username, password string) (string, error)
	EditPassword(ctx context.Context, id int64, oldPassword, newPassword string) error
}
