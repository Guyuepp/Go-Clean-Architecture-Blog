package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/redis/go-redis/v9"
)

type ArticleCache struct {
	client *redis.Client
}

func NewArticleCache(client *redis.Client) *ArticleCache {
	return &ArticleCache{client}
}

func (c *ArticleCache) Get(ctx context.Context, id int64) (res domain.Article, err error) {
	key := fmt.Sprintf("article:%d", id)
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.Article{}, redis.Nil
	} else if err != nil {
		return domain.Article{}, err
	}
	json.Unmarshal(data, &res)
	return
}

func (c *ArticleCache) Set(ctx context.Context, ar *domain.Article) (err error) {
	key := fmt.Sprintf("article:%d", ar.ID)
	data, err := json.Marshal(ar)
	if err != nil {
		return
	}
	err = c.client.Set(ctx, key, data, 10*time.Minute).Err()
	return
}

func (c *ArticleCache) Incr(ctx context.Context, id int64) (int64, error) {
	key := fmt.Sprintf("article:views:%d", id)
	return c.client.Incr(ctx, key).Result()
}
