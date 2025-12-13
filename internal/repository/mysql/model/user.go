package model

import (
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
)

type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"type:varchar(32);not null"`
	Username  string    `gorm:"type:varchar(32);not null"`
	Password  string    `gorm:"type:varchar(64);not null"`
	CreatedAt time.Time `gorm:"type:datetime"`
	UpdatedAt time.Time `gorm:"type:datetime"`
}

func (User) TableName() string {
	return "user"
}

func (m *User) ToDomain() domain.User {
	return domain.User{
		ID:        m.ID,
		Name:      m.Name,
		Username:  m.Username,
		Password:  m.Password,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func NewUserFromDomain(a *domain.User) User {
	return User{
		ID:        a.ID,
		Name:      a.Name,
		Username:  a.Username,
		Password:  a.Password,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
