package request

import "github.com/bxcodec/go-clean-arch/domain"

type User struct {
	Name     string `json:"name"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *User) ToDomain() domain.User {
	return domain.User{
		Name:     a.Name,
		Username: a.Username,
		Password: a.Password,
	}
}
