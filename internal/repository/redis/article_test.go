package redis_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
	redisRepo "github.com/bxcodec/go-clean-arch/internal/repository/redis"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	db, mock := redismock.NewClientMock()
	cache := redisRepo.NewArticleCache(db)

	t.Run("success", func(t *testing.T) {
		article := &domain.Article{ID: 1, Title: "Test Article"}
		data := `{"id":1,"title":"Test Article"}`
		mock.ExpectGet("article:1").SetVal(data)

		result, err := cache.Get(context.Background(), 1)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, article.ID, result.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectGet("article:2").RedisNil()

		result, err := cache.Get(context.Background(), 2)

		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("redis error", func(t *testing.T) {
		mock.ExpectGet("article:3").SetErr(assert.AnError)

		result, err := cache.Get(context.Background(), 3)

		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSet(t *testing.T) {
	db, mock := redismock.NewClientMock()
	cache := redisRepo.NewArticleCache(db)

	t.Run("success", func(t *testing.T) {
		article := &domain.Article{ID: 1, Title: "Test Article"}
		data, _ := json.Marshal(article)
		mock.ExpectSet("article:1", data, 10*time.Minute).SetVal("OK")

		err := cache.Set(context.Background(), article)
		assert.NoError(t, err)
	})

	t.Run("redis error", func(t *testing.T) {
		article := &domain.Article{ID: 2, Title: "Test Article"}
		data, _ := json.Marshal(article)
		mock.ExpectSet("article:2", data, 10*time.Minute).SetErr(assert.AnError)

		err := cache.Set(context.Background(), article)
		assert.Error(t, err)
	})
}
