package model

import (
	"time"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
)

type UserLike struct {
	ArticleID int64     `gorm:"column:article_id;not null"`
	UserID    int64     `gorm:"column:user_id;not null"`
	CreatedAt time.Time `gorm:"type:datatime"`
}

func (UserLike) TableName() string {
	return "user_likes"
}

func NewUserLikeFromDomain(ul domain.UserLike) UserLike {
	return UserLike{
		ArticleID: ul.ArticleID,
		UserID:    ul.UserID,
		CreatedAt: ul.CreatedAt,
	}
}
