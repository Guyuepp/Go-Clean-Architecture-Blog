package response

import "github.com/bxcodec/go-clean-arch/domain"

type Author struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthorFromDomain(a *domain.Author) *Author {
	return &Author{
		Name:     a.Name,
		Username: a.Username,
		Password: a.Password,
	}
}
