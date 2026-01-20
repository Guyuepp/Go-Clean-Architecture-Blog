package model

import (
	"time"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
)

type Comment struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	ArticleID int64     `gorm:"column:article_id;not null"`
	UserID    int64     `gorm:"column:user_id;not null"`
	Content   string    `gorm:"type:text;not null"`
	ParentID  int64     `gorm:"column:parent_id;default:0"`
	RootID    int64     `gorm:"column:root_id;default:0"`
	CreatedAt time.Time `gorm:"type:datetime"`
}

func (Comment) TableName() string {
	return "comment"
}

func NewCommentFromDomain(c *domain.Comment) *Comment {
	return &Comment{
		ID:        c.ID,
		ArticleID: c.ArticleID,
		UserID:    c.UserID,
		Content:   c.Content,
		ParentID:  c.ParentID,
		RootID:    c.RootID,
		CreatedAt: c.CreatedAt,
	}
}

func (m *Comment) ToDomain() domain.Comment {
	return domain.Comment{
		ID:        m.ID,
		ArticleID: m.ArticleID,
		UserID:    m.UserID,
		Content:   m.Content,
		ParentID:  m.ParentID,
		RootID:    m.RootID,
		CreatedAt: m.CreatedAt,
	}
}
