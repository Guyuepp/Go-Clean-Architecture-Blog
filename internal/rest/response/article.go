package response // 建议包名就叫 response

import (
	"github.com/bxcodec/go-clean-arch/domain"
)

type Article struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	AuthorName string `json:"author_name"`
	UpdatedAt  string `json:"updated_at"`
	CreatedAt  string `json:"created_at"`
	Views      int64  `json:"views"`
}

// FromDomain: Domain -> Response
func NewArticleFromDomain(a *domain.Article) *Article {
	return &Article{
		ID:         a.ID,
		Title:      a.Title,
		Content:    a.Content,
		AuthorName: a.Author.Name,
		UpdatedAt:  a.UpdatedAt.Format("2006-01-02 15:04:05"),
		CreatedAt:  a.CreatedAt.Format("2006-01-02 15:04:05"),
		Views:      a.Views,
	}
}
