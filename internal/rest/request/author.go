package request

import "github.com/bxcodec/go-clean-arch/domain"

type Author struct {
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *Author) ToDomain() domain.Author {
	return domain.Author{
		Name:     a.Name,
		Username: a.Username,
		Password: a.Password,
	}
}
