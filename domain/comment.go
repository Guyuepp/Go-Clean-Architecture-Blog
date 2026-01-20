package domain

import (
	"context"
	"time"
)

// Comment domain model
type Comment struct {
	ID        int64     `json:"id"`
	ArticleID int64     `json:"article_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	ParentID  int64     `json:"parent_id"`
	RootID    int64     `json:"root_id"`
	CreatedAt time.Time `json:"created_at"`

	// User 评论作者信息
	User *User `json:"user,omitempty"`
	// Replies 子评论列表
	Replies []*Comment `json:"replies,omitempty"`
}

// CommentUsecase 业务逻辑接口
type CommentUsecase interface {
	Create(ctx context.Context, c *Comment) error
	Delete(ctx context.Context, articleID int64, userID int64) error
	FetchByArticle(ctx context.Context, articleID int64, cursor string, limit int64) ([]*Comment, string, error)
}

// CommentRepository 数据存取接口
type CommentRepository interface {
	Store(ctx context.Context, c *Comment) error
	Delete(ctx context.Context, articleID int64, userID int64) error
	GetByID(ctx context.Context, id int64) (*Comment, error)
	// FetchRoots 获取一级评论
	FetchRoots(ctx context.Context, articleID int64, cursor string, limit int64) ([]*Comment, error)
	// FetchReplies 获取指定根评论ID列表的所有子回复
	FetchReplies(ctx context.Context, rootIDs []int64) ([]*Comment, error)
}
