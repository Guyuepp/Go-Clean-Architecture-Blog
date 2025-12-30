package response // 建议包名就叫 response

import (
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
)

type Article struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	UserName  string `json:"user_name"`
	UpdatedAt string `json:"updated_at"`
	CreatedAt string `json:"created_at"`
	Views     int64  `json:"views"`
	Likes     int64  `json:"likes"`
}

// FromDomain: Domain -> Response
func NewArticleFromDomain(a *domain.Article) Article {
	return Article{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		UserName:  a.User.Name,
		UpdatedAt: a.UpdatedAt.Format("2006-01-02 15:04:05"),
		CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
		Views:     a.Views,
		Likes:     a.Likes,
	}
}
