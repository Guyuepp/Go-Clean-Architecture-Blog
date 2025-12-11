package request

import (
	"github.com/bxcodec/go-clean-arch/domain"
)

// Article 用户创建/修改文章时填写的字段
type Article struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// ToDomain: Request -> Domain
func (r *Article) ToDomain() *domain.Article {
	return &domain.Article{
		Title:   r.Title,
		Content: r.Content,
	}
}
