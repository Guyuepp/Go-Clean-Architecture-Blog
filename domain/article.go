package domain

import (
	"context"
	"time"
)

// Article is representing the Article data struct
type Article struct {
	ID        int64     // Unique iedntifier for the article
	Title     string    // Article title
	Content   string    // Article body content
	User      User      // Author information
	UpdatedAt time.Time // Last update timestamp
	CreatedAt time.Time // Creation timestamp
	Views     int64     // Number of views
	Likes     int64     // Number of likes
}

// ArticleRepository defines the contract for article data persistence
type ArticleRepository interface {
	// Fetch retrieves a paginated list of articles.
	// cursor: for pagination, pass the last article ID or empty string for the first page.
	// num: number of articles to fetch per page.
	// Returns: articles, next cursor for the next page, and error if any.
	Fetch(ctx context.Context, cursor string, num int64) (res []Article, err error)

	// GetByID retrieves a single article by its ID.
	// Returns ErrNotFound if the article doesn't exist.
	GetByID(ctx context.Context, id int64) (Article, error)

	// GetByIDs retrieves articles by given IDs.
	// Returns ErrNotFound if some of the articles do not exist.
	GetByIDs(ctx context.Context, ids []int64) ([]Article, error)

	// GetByTitle retrieves an article by its title.
	GetByTitle(ctx context.Context, title string) (Article, error)

	// UpdateViews increments the view count of an article.
	AddViews(ctx context.Context, id int64, deltaViews int64) error

	// Update modifies an existing article.
	// Returns ErrNotFount if the article doesn't exist.
	Update(ctx context.Context, ar *Article) error

	// Store creates a new article in the repository.
	Store(ctx context.Context, a *Article) error

	// Delete removes an article by its ID.
	// Returns ErrNotFount if not exists
	Delete(ctx context.Context, id int64) error

	// AddLikes add the likes of an article by deltaLikes
	AddLikes(ctx context.Context, id int64, deltaLikes int64) error

	// FetchUserLikedArticles 从 user_likes 表中按 article_id DESC 排序选择 user_id=? 的记录，限制条数
	FetchUserLikedArticles(ctx context.Context, uid int64, limit int64) ([]int64, error)

	ApplyLikeChanges(ctx context.Context, changes LikeStateChanges) error

	FetchArticlesByLikes(ctx context.Context, limit int64) ([]Article, error)

	FetchIDs(ctx context.Context, cursor, limit int64) ([]int64, error)
}

type ArticleCache interface {
	// Article related
	GetHome(context.Context) ([]Article, error)
	SetHome(context.Context, []Article) error
	GetArticle(ctx context.Context, id int64) (res Article, err error)
	GetArticleByIDs(ctx context.Context, ids []int64) (res []Article, err error)
	SetArticle(ctx context.Context, ar *Article) (err error)
	BatchSetArticle(ctx context.Context, ars []Article) error

	// Del delete article, views and likes in cache
	DeleteArticle(ctx context.Context, id int64) (err error)

	// Views related
	IncrViews(ctx context.Context, id int64) (views int64, err error)
	FetchAndResetViews(ctx context.Context) (map[int64]int64, error)

	// Likes related
	GetLikeCount(ctx context.Context, articleID int64) (int64, error)
	MGetLikeCounts(ctx context.Context, articleIDs []int64) (map[int64]int64, error)
	SetLikeCount(ctx context.Context, articleID int64, likes int64) error
	MSetLikeCount(ctx context.Context, articleIDs []int64, likes []int64) error

	AddLikeRecord(ctx context.Context, likeRecord UserLike) (bool, error)
	DecrLikeRecord(ctx context.Context, likeRecord UserLike) (bool, error)
	IsLiked(ctx context.Context, likeRecord UserLike) (bool, error)
	IsLikedBatch(ctx context.Context, userID int64, articleIDs []int64) (map[int64]bool, error)
	SetUserLikedArticles(ctx context.Context, UserID int64, articleIDs []int64) error

	GetDailyRank(ctx context.Context, limit int64) ([]Article, error)
	IncrDailyRankScore(ctx context.Context, aid int64, scoreDelta float64) error
	GetHistoryRank(ctx context.Context, limit int64) ([]Article, error)
	SetHistoryRank(ctx context.Context, articleIDs []int64, scores []float64) error
}

type ArticleUsecase interface {
	Fetch(ctx context.Context, cursor string, num int64) ([]Article, string, error)
	GetByID(ctx context.Context, id int64) (Article, error)
	Store(ctx context.Context, ar *Article) error
	Update(ctx context.Context, ar *Article) error
	Delete(ctx context.Context, id int64) error
	AddLikeRecord(ctx context.Context, likeRecord UserLike) (bool, error)
	RemoveLikeRecord(ctx context.Context, likeRecord UserLike) (bool, error)
	FetchDailyRank(ctx context.Context, limit int64) ([]Article, error)
	FetchHistoryRank(ctx context.Context, limit int64) ([]Article, error)
	InitBloomFilter(ctx context.Context) error
}
