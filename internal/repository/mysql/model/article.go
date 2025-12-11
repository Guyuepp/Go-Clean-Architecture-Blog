package model

import (
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
)

type Article struct {
	ID        int64
	Title     string
	Content   string
	UserID    int64
	UpdatedAt time.Time
	CreatedAt time.Time
	Views     int64
}

func (m *Article) TableName() string {
	return "article"
}

// ToDomain turns the DTO into the domain Article struct
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

// FromDomain turns the domain Article struct into the DTO
func FromDomain(a *domain.Article) *Article {
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
