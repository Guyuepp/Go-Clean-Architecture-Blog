package model

import (
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
)

type Article struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Title     string    `gorm:"type:varchar(45);not null"`
	Content   string    `gorm:"type:longtext;not null"`
	UserID    int64     `gorm:"column:user_id;default:0"`
	Views     int64     `gorm:"default:0"`
	UpdatedAt time.Time `gorm:"type:datetime"`
	CreatedAt time.Time `gorm:"type:datetime"`
}

func (Article) TableName() string {
	return "article"
}

func (m *Article) ToDomain() domain.Article {
	return domain.Article{
		ID:        m.ID,
		Title:     m.Title,
		Content:   m.Content,
		UpdatedAt: m.UpdatedAt,
		CreatedAt: m.CreatedAt,
		Author: domain.Author{
			ID: m.UserID,
		},
		Views: m.Views,
	}
}

func NewArticleFromDomain(a *domain.Article) *Article {
	return &Article{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		UserID:    a.Author.ID,
		UpdatedAt: a.UpdatedAt,
		CreatedAt: a.CreatedAt,
		Views:     a.Views,
	}
}
