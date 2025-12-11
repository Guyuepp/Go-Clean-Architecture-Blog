package model

import "github.com/bxcodec/go-clean-arch/domain"

type User struct {
	ID        int64
	Name      string
	Username  string
	Password  string
	CreatedAt string
	UpdatedAt string
}

func (m *User) TableName() string {
	return "user"
}

// ToDomain turns the DTO into the domain Author struct
func (m *User) ToDomain() domain.Author {
	return domain.Author{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// FromDomain turns the domain Author struct into the DTO
func (m *User) FromDomain(a *domain.Author) *User {
	return &User{
		ID:        a.ID,
		Name:      a.Name,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
