package domain

import (
	"context"
	"time"
)

// User representing the User data struct
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
