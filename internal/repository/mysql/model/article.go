package model

import (
	"time"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
)

type Article struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Title     string    `gorm:"type:varchar(45);not null"`
	Content   string    `gorm:"type:longtext;not null"`
	UserID    int64     `gorm:"column:user_id;not null"`
	Views     int64     `gorm:"default:0"`
	Likes     int64     `gorm:"default:0"`
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
		User: domain.User{
			ID: m.UserID,
		},
		Views: m.Views,
		Likes: m.Likes,
	}
}

func NewArticleFromDomain(a *domain.Article) *Article {
	return &Article{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		UserID:    a.User.ID,
		UpdatedAt: a.UpdatedAt,
		CreatedAt: a.CreatedAt,
		Views:     a.Views,
		Likes:     a.Likes,
	}
}
