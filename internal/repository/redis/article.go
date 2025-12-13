package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/redis/go-redis/v9"
)

const (
	KeyViewsBuffer     = "article:views:buffer"
	KeyViewsProcessing = "article:views:processing"
)

type ArticleCache struct {
	client *redis.Client
}

func NewArticleCache(client *redis.Client) *ArticleCache {
	return &ArticleCache{
		client,
	}
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
	return c.client.HIncrBy(ctx, KeyViewsBuffer, strconv.FormatInt(id, 10), 1).Result()
}

func (c *ArticleCache) FetchAndResetViews(ctx context.Context) (map[int64]int64, error) {
	result := make(map[int64]int64)
	err := c.client.Rename(ctx, KeyViewsBuffer, KeyViewsProcessing).Err()
	if err != nil {
		return result, err
	}

	data, err := c.client.HGetAll(ctx, KeyViewsProcessing).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return result, nil
		}
		return result, err
	}

	for idStr, viewsStr := range data {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		views, _ := strconv.ParseInt(viewsStr, 10, 64)
		result[id] = views
	}

	c.client.Del(ctx, KeyViewsProcessing)

	return result, nil
}

func (c *ArticleCache) Del(ctx context.Context, id int64) (err error) {
	key := fmt.Sprintf("article:%d", id)
	err = c.client.Del(ctx, key).Err()
	if err != nil {
		return
	}
	key = fmt.Sprintf("article:views:%d", id)
	err = c.client.Del(ctx, key).Err()
	return
}
