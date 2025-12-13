package domain

import (
	"context"
	"time"
)

// Article is representing the Article data struct
type Article struct {
	ID        int64
	Title     string
	Content   string
	User      User
	UpdatedAt time.Time
	CreatedAt time.Time
	Views     int64
}

type ArticleRepository interface {
	Fetch(ctx context.Context, cursor string, num int64) (res []Article, nextCursor string, err error)
	GetByID(ctx context.Context, id int64) (Article, error)
	GetByTitle(ctx context.Context, title string) (Article, error)
	AddViews(ctx context.Context, id int64, newViews int64) error
	Update(ctx context.Context, ar *Article) error
	Store(ctx context.Context, a *Article) error
	Delete(ctx context.Context, id int64) error
}

type ArticleCache interface {
	Get(ctx context.Context, id int64) (res Article, err error)
	Set(ctx context.Context, ar *Article) (err error)
	Del(ctx context.Context, id int64) (err error)
	Incr(ctx context.Context, id int64) (views int64, err error)
	FetchAndResetViews(ctx context.Context) (map[int64]int64, error)
}

type ArticleUsecase interface {
	Fetch(ctx context.Context, cursor string, num int64) ([]Article, string, error)
	GetByID(ctx context.Context, id int64) (Article, error)
	Store(ctx context.Context, ar *Article) error
	Update(ctx context.Context, ar *Article) error
	Delete(ctx context.Context, id int64) error
}
