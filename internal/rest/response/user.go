package response

import "github.com/bxcodec/go-clean-arch/domain"

type User struct {
	Name       string `json:"name"`
	Username   string `json:"username"`
	Created_at string `json:"created_at"`
}

func NewUserFromDomain(a *domain.User) User {
	return User{
		Name:       a.Name,
		Username:   a.Username,
		Created_at: a.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
