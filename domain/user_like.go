package domain

import "time"

const (
	// 默认每个用户只加载最近发布的300篇文章的点赞
	LikeRecordLimit = 300
)

// UserLike is representing a like record
type UserLike struct {
	ArticleID int64
	UserID    int64
	CreatedAt time.Time
}

type LikeStateChanges struct {
	ToAdd    []UserLike
	ToRemove []UserLike
}
