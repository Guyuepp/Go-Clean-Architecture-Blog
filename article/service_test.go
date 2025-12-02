package article_test

import (
	"context"
	"testing"
	"time"

	"github.com/bxcodec/go-clean-arch/article"
	"github.com/bxcodec/go-clean-arch/article/mocks"
	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetByID(t *testing.T) {
	mockArticle := domain.Article{
		ID:      1,
		Title:   "Hello",
		Content: "Content",
		Author:  domain.Author{ID: 1},
		Views:   0,
	}
	mockAuthor := domain.Author{
		ID:   1,
		Name: "Iman",
	}

	t.Run("success - cache hit, incr ok", func(t *testing.T) {
		mockArticleRepo := new(mocks.ArticleRepository)
		mockAuthorRepo := new(mocks.AuthorRepository)
		mockArticleCache := new(mocks.ArticleCache)
		s := article.NewService(mockArticleRepo, mockAuthorRepo, mockArticleCache)

		// 1. 命中缓存
		mockArticleCache.
			On("Get", context.Background(), int64(1)).
			Return(mockArticle, nil)

		// 2. Incr 成功，返回 10
		mockArticleCache.
			On("Incr", mock.Anything, int64(1)).
			Return(int64(10), nil)

		// 3. newViews % 10 == 0，会异步调用 UpdateViews
		mockArticleRepo.
			On("UpdateViews", mock.Anything, int64(1), int64(10)).
			Return(nil)

		res, err := s.GetByID(context.Background(), 1)

		assert.NoError(t, err)
		// Views 应该是 Incr 返回的 10
		assert.Equal(t, int64(10), res.Views)
		// 其他字段保持与缓存一致
		assert.Equal(t, mockArticle.ID, res.ID)
		assert.Equal(t, mockArticle.Title, res.Title)

		// 等待 goroutine 执行 UpdateViews
		time.Sleep(20 * time.Millisecond)

		// 命中缓存时不应查 DB，也不应 Set 缓存
		mockArticleRepo.AssertNotCalled(t, "GetByID", mock.Anything, int64(1))
		mockAuthorRepo.AssertNotCalled(t, "GetByID", mock.Anything, int64(1))
		mockArticleCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)

		mockArticleRepo.AssertExpectations(t)
		mockArticleCache.AssertExpectations(t)
	})

	t.Run("success - cache miss, incr ok, set cache", func(t *testing.T) {
		mockArticleRepo := new(mocks.ArticleRepository)
		mockAuthorRepo := new(mocks.AuthorRepository)
		mockArticleCache := new(mocks.ArticleCache)
		s := article.NewService(mockArticleRepo, mockAuthorRepo, mockArticleCache)

		// 1. 未命中缓存，返回 redis.Nil
		mockArticleCache.
			On("Get", context.Background(), int64(1)).
			Return(domain.Article{}, redis.Nil)

		// 2. 从 DB 获取文章 + 作者
		mockArticleRepo.
			On("GetByID", context.Background(), int64(1)).
			Return(mockArticle, nil)
		mockAuthorRepo.
			On("GetByID", context.Background(), int64(1)).
			Return(mockAuthor, nil)

		// 3. Incr 成功，返回 5（不触发 %10==0 的 UpdateViews）
		mockArticleCache.
			On("Incr", mock.Anything, int64(1)).
			Return(int64(5), nil)

		// 4. shouldSetCache == true，会在 goroutine 里 Set 缓存
		mockArticleCache.
			On("Set", mock.Anything, mock.MatchedBy(func(a *domain.Article) bool {
				return a.ID == 1 && a.Author.Name == "Iman" && a.Views == 5
			})).
			Return(nil)

		res, err := s.GetByID(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), res.Views)
		assert.Equal(t, "Iman", res.Author.Name)

		time.Sleep(20 * time.Millisecond)

		// views=5，不是 10 的倍数，不会调用 UpdateViews
		mockArticleRepo.AssertNotCalled(t, "UpdateViews", mock.Anything, int64(1), mock.Anything)

		mockArticleRepo.AssertExpectations(t)
		mockAuthorRepo.AssertExpectations(t)
		mockArticleCache.AssertExpectations(t)
	})

	t.Run("error - cache get error (not nil) falls back to db", func(t *testing.T) {
		mockArticleRepo := new(mocks.ArticleRepository)
		mockAuthorRepo := new(mocks.AuthorRepository)
		mockArticleCache := new(mocks.ArticleCache)
		s := article.NewService(mockArticleRepo, mockAuthorRepo, mockArticleCache)

		// Get 出错，但不是 redis.Nil
		mockArticleCache.
			On("Get", context.Background(), int64(1)).
			Return(domain.Article{}, assert.AnError)

		// 退回到 DB
		mockArticleRepo.
			On("GetByID", context.Background(), int64(1)).
			Return(mockArticle, nil)
		mockAuthorRepo.
			On("GetByID", context.Background(), int64(1)).
			Return(mockAuthor, nil)

		// Incr 正常
		mockArticleCache.
			On("Incr", mock.Anything, int64(1)).
			Return(int64(3), nil)

		// shouldSetCache == true，会 Set
		mockArticleCache.
			On("Set", mock.Anything, mock.Anything).
			Return(nil)

		res, err := s.GetByID(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), res.Views)
		assert.Equal(t, mockArticle.ID, res.ID)

		time.Sleep(20 * time.Millisecond)

		mockArticleRepo.AssertExpectations(t)
		mockAuthorRepo.AssertExpectations(t)
		mockArticleCache.AssertExpectations(t)
	})

	t.Run("success - cache hit, incr error, views not overridden", func(t *testing.T) {
		mockArticleRepo := new(mocks.ArticleRepository)
		mockAuthorRepo := new(mocks.AuthorRepository)
		mockArticleCache := new(mocks.ArticleCache)
		s := article.NewService(mockArticleRepo, mockAuthorRepo, mockArticleCache)

		articleWithViews := mockArticle
		articleWithViews.Views = 42

		// 命中缓存
		mockArticleCache.
			On("Get", context.Background(), int64(1)).
			Return(articleWithViews, nil)

		// Incr 出错
		mockArticleCache.
			On("Incr", mock.Anything, int64(1)).
			Return(int64(0), assert.AnError)

		res, err := s.GetByID(context.Background(), 1)

		assert.NoError(t, err)
		// Incr 失败时，当前实现不会修改 Views（只打日志）
		assert.Equal(t, int64(42), res.Views)

		time.Sleep(10 * time.Millisecond)

		// 不会调用 UpdateViews / Set
		mockArticleRepo.AssertNotCalled(t, "UpdateViews", mock.Anything, mock.Anything, mock.Anything)
		mockArticleCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)

		mockArticleRepo.AssertExpectations(t)
		mockArticleCache.AssertExpectations(t)
	})
}
