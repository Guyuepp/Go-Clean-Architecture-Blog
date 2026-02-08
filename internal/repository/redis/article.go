package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository/cache"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	KeyArticles               = "article:%d"
	KeyUserLikedArticles      = "article:user:%d:likedArticles"
	KeyHotDailyRaw            = "article:hot:daily:raw:%s"
	KeyHotDailyAggreGatedRank = "article:hot:daily:rank"
	KeyHotHistoryRank         = "article:hot:history:rank"
	KeyLikesBuffer            = "article:likes:%d"
	KeyViewsBuffer            = "article:views:buffer"
	KeyViewsProcessing        = "article:views:processing"
	KeyHome                   = "article:home"
)

type articleCache struct {
	client *redis.Client
}

var _ domain.ArticleCache = (*articleCache)(nil)

func NewArticleCache(client *redis.Client) *articleCache {
	return &articleCache{
		client,
	}
}

// GetHomeWithLogicalExpire 获取首页数据，支持逻辑过期检测
// 返回: 数据、是否逻辑过期、错误
func (c *articleCache) GetHomeWithLogicalExpire(ctx context.Context) ([]domain.Article, bool, error) {
	key := KeyHome
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false, err
	}

	var wrapper cache.DataWithLogicalExpire
	err = json.Unmarshal(data, &wrapper)
	if err != nil {
		return nil, false, err
	}

	// 解析实际数据
	articlesJSON, err := json.Marshal(wrapper.Data)
	if err != nil {
		return nil, false, err
	}

	var articles []domain.Article
	err = json.Unmarshal(articlesJSON, &articles)
	if err != nil {
		return nil, false, err
	}

	// 检查是否逻辑过期
	isExpired := wrapper.IsLogicalExpired()
	return articles, isExpired, nil
}

// SetHomeWithLogicalExpire 设置首页数据，使用逻辑过期
func (c *articleCache) SetHomeWithLogicalExpire(ctx context.Context, ars []domain.Article, ttl time.Duration) error {
	key := KeyHome
	wrapper := cache.NewDataWithLogicalExpire(ars, ttl)
	data, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}
	// 物理永不过期（或设置很长时间），避免缓存击穿
	err = c.client.Set(ctx, key, data, 24*time.Hour).Err()
	return err
}

// GetArticleWithLogicalExpire 获取文章，支持逻辑过期
func (c *articleCache) GetArticleWithLogicalExpire(ctx context.Context, id int64) (domain.Article, bool, error) {
	key := fmt.Sprintf(KeyArticles, id)
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.Article{}, false, redis.Nil
	} else if err != nil {
		return domain.Article{}, false, err
	}

	var wrapper cache.DataWithLogicalExpire
	if err = json.Unmarshal(data, &wrapper); err != nil {
		return domain.Article{}, false, err
	}

	// 解析实际文章数据
	articleJSON, err := json.Marshal(wrapper.Data)
	if err != nil {
		return domain.Article{}, false, err
	}

	var article domain.Article
	if err = json.Unmarshal(articleJSON, &article); err != nil {
		return domain.Article{}, false, err
	}

	isExpired := wrapper.IsLogicalExpired()
	return article, isExpired, nil
}

// GetArticleByIDsWithLogicalExpire 批量获取文章（支持逻辑过期）
func (c *articleCache) GetArticleByIDsWithLogicalExpire(ctx context.Context, ids []int64) ([]domain.Article, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf(KeyArticles, id)
	}

	jsonList, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	articles := make([]domain.Article, 0, len(ids))
	for _, val := range jsonList {
		if val == nil {
			continue
		}

		if str, ok := val.(string); ok {
			var wrapper cache.DataWithLogicalExpire
			if err := json.Unmarshal([]byte(str), &wrapper); err != nil {
				continue
			}

			articleJSON, _ := json.Marshal(wrapper.Data)
			var ar domain.Article
			if err := json.Unmarshal(articleJSON, &ar); err != nil {
				continue
			}

			if !wrapper.IsLogicalExpired() {
				articles = append(articles, ar)
			}
		}
	}

	return articles, nil
}

// SetArticleWithLogicalExpire 设置文章缓存，使用逻辑过期
func (c *articleCache) SetArticleWithLogicalExpire(ctx context.Context, ar *domain.Article, ttl time.Duration) error {
	key := fmt.Sprintf(KeyArticles, ar.ID)
	wrapper := cache.NewDataWithLogicalExpire(ar, ttl)
	data, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, 24*time.Hour).Err()
}

// BatchSetArticleWithLogicalExpire 批量设置文章缓存
func (c *articleCache) BatchSetArticleWithLogicalExpire(ctx context.Context, ars []domain.Article, ttl time.Duration) error {
	if len(ars) == 0 {
		return nil
	}

	iar := make([]any, 0, 2*len(ars))
	var errMarshal error = nil
	for i := range ars {
		wrapper := cache.NewDataWithLogicalExpire(ars[i], ttl)
		data, err := json.Marshal(wrapper)
		if err != nil {
			logrus.Warnf("failed to marshal article for cache, ID: %d, err: %v", ars[i].ID, err)
			errMarshal = err
			continue
		}
		key := fmt.Sprintf(KeyArticles, ars[i].ID)
		iar = append(iar, key, data)
	}
	if len(iar) == 0 {
		return errMarshal
	}
	return c.client.MSet(ctx, iar...).Err()
}

func (c *articleCache) IncrViews(ctx context.Context, id int64) (int64, error) {
	return c.client.HIncrBy(ctx, KeyViewsBuffer, strconv.FormatInt(id, 10), 1).Result()
}

func (c *articleCache) FetchAndResetViews(ctx context.Context) (map[int64]int64, error) {
	var script = redis.NewScript(`
		-- 1. 检查 Buffer 是否存在
		if redis.call("EXISTS", KEYS[1]) == 0 then
			return nil
		end

		-- 2. 将 Buffer 重命名为 Processing (直接覆盖或先检查)
		-- 注意：这里用 RENAME，如果 KEYS[2] 已存在会被覆盖
		redis.call("RENAME", KEYS[1], KEYS[2])

		-- 3. 获取所有数据
		local data = redis.call("HGETALL", KEYS[2])

		-- 4. 删除 Processing 键（因为数据已经读到 Lua 内存中了）
		redis.call("DEL", KEYS[2])

		-- 5. 返回数据给 Go
		return data
	`)
	result := make(map[int64]int64)

	// KEYS[1] = KeyViewsBuffer, KEYS[2] = KeyViewsProcessing
	val, err := script.Run(ctx, c.client, []string{KeyViewsBuffer, KeyViewsProcessing}).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return result, nil
		}
		return nil, err
	}

	data, ok := val.([]any)
	if !ok {
		return result, nil
	}

	for i := 0; i < len(data); i += 2 {
		idStr, _ := data[i].(string)
		viewsStr, _ := data[i+1].(string)

		id, _ := strconv.ParseInt(idStr, 10, 64)
		views, _ := strconv.ParseInt(viewsStr, 10, 64)
		result[id] = views
	}

	return result, nil
}

// TODO 应该删除缓存中的相关数据
func (c *articleCache) DeleteArticle(ctx context.Context, id int64) error {
	key := fmt.Sprintf(KeyArticles, id)
	err := c.client.Del(ctx, key).Err()
	return err
}

func (c *articleCache) AddLikeRecord(ctx context.Context, likeRecord domain.UserLike) (bool, error) {
	// KEYS = {该用户喜欢的文章列表, 今日热榜, 点赞数}
	// ARGV = {本次文章ID, 点赞加分}
	keys := []string{
		fmt.Sprintf(KeyUserLikedArticles, likeRecord.UserID),
		fmt.Sprintf(KeyHotDailyRaw, time.Now().Format("2006010215")),
		fmt.Sprintf(KeyLikesBuffer, likeRecord.ArticleID),
	}
	args := []any{likeRecord.ArticleID, 1}
	var script = redis.NewScript(`
		if redis.call('EXISTS', KEYS[1]) == 0 then
			return -1 -- 未缓存, 需要加载缓存
		end

		if redis.call('SISMEMBER', KEYS[1], ARGV[1]) == 1 then
			return 0 -- 最近已点赞
		else 
			redis.call('SADD', KEYS[1], ARGV[1])
			redis.call('EXPIRE', KEYS[1], 1800)

			redis.call('ZINCRBY', KEYS[2], ARGV[2], ARGV[1])
			redis.call('EXPIRE', KEYS[2], 60*60*26) -- 26 hours
			
			if redis.call('EXISTS', KEYS[3]) == 1 then
				redis.call('INCR', KEYS[3])
				redis.call('EXPIRE', KEYS[3], 7*24*60*60)
			end

			return 1 -- 点赞成功
		end
	`)

	res, err := script.Run(ctx, c.client, keys, args).Int()
	if err != nil {
		return false, err
	}
	switch res {
	case -1:
		return false, domain.ErrCacheMiss
	case 0:
		return false, nil
	default:
		return true, nil
	}
}

func (c *articleCache) DecrLikeRecord(ctx context.Context, likeRecord domain.UserLike) (bool, error) {
	// KEYS = {该用户喜欢的文章列表, 今日热榜, 点赞数}
	// ARGV = {本次文章ID, 点赞加分}
	keys := []string{
		fmt.Sprintf(KeyUserLikedArticles, likeRecord.UserID),
		fmt.Sprintf(KeyHotDailyRaw, time.Now().Format("2006010215")),
		fmt.Sprintf(KeyLikesBuffer, likeRecord.ArticleID),
	}
	args := []any{likeRecord.ArticleID, -1}
	var script = redis.NewScript(`
		if redis.call('EXISTS', KEYS[1]) == 0 then
			return -1 -- 未缓存, 需要加载缓存
		end

		if redis.call('SISMEMBER', KEYS[1], ARGV[1]) == 0 then
			return 0 -- 最近未点赞
		else 
			redis.call('SREM', KEYS[1], ARGV[1])
			redis.call('EXPIRE', KEYS[1], 1800)

			redis.call('ZINCRBY', KEYS[2], ARGV[2], ARGV[1])
			redis.call('EXPIRE', KEYS[2], 60*60*26) -- 26 hours

			if redis.call('EXISTS', KEYS[3]) == 1 then
				redis.call('DECR', KEYS[3])
				redis.call('EXPIRE', KEYS[3], 7*24*60*60)
			end

			return 1 -- 取消赞成功
		end
	`)

	res, err := script.Run(ctx, c.client, keys, args).Int()
	if err != nil {
		return false, err
	}
	switch res {
	case -1:
		return false, domain.ErrCacheMiss
	case 0:
		return false, nil
	default:
		return true, nil
	}
}

func (c *articleCache) IsLiked(ctx context.Context, likeRecord domain.UserLike) (bool, error) {
	return c.client.SIsMember(ctx, fmt.Sprintf(KeyUserLikedArticles, likeRecord.UserID), any(likeRecord.ArticleID)).Result()
}

func (c *articleCache) IsLikedBatch(ctx context.Context, uid int64, aids []int64) (map[int64]bool, error) {
	if len(aids) == 0 {
		return nil, nil
	}
	args := make([]any, len(aids))
	for i, aid := range aids {
		args[i] = any(aid)
	}

	script := redis.NewScript(`
        if redis.call('EXISTS', KEYS[1]) == 0 then
            return nil
        end
        
        redis.call('EXPIRE', KEYS[1], 60*30) 

        local results = {}
        for i, id in ipairs(ARGV) do
            results[i] = redis.call('SISMEMBER', KEYS[1], id)
        end
        return results
    `)
	result, err := script.Run(ctx, c.client, []string{fmt.Sprintf(KeyUserLikedArticles, uid)}, args).Slice()

	if err == redis.Nil {
		return nil, domain.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}

	resMap := make(map[int64]bool)
	for i, val := range result {
		resMap[aids[i]] = (val.(int64) == 1)
	}

	return resMap, nil
}

func (c *articleCache) SetUserLikedArticles(ctx context.Context, uid int64, aids []int64) error {
	if len(aids) == 0 {
		aids = append(aids, -1)
	}
	iaids := make([]any, len(aids))
	for i, aid := range aids {
		iaids[i] = any(aid)
	}
	key := fmt.Sprintf(KeyUserLikedArticles, uid)
	return c.client.SAdd(ctx, key, iaids...).Err()
}

func (c *articleCache) GetDailyRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	if c.client.Exists(ctx, KeyHotDailyAggreGatedRank).Val() > 0 {
		return c.fetchRankFromKey(ctx, KeyHotDailyAggreGatedRank, limit)
	}

	keys := make([]string, 24)
	now := time.Now()
	for i := range 24 {
		keys[i] = fmt.Sprintf(KeyHotDailyRaw, now.Add(time.Duration(-i)*time.Hour).Format("2006010215"))
	}

	err := c.client.ZUnionStore(ctx, KeyHotDailyAggreGatedRank, &redis.ZStore{
		Keys:      keys,
		Aggregate: "SUM",
	}).Err()

	if err != nil {
		return nil, err
	}

	c.client.Expire(ctx, KeyHotDailyAggreGatedRank, 5*time.Minute)

	return c.fetchRankFromKey(ctx, KeyHotDailyAggreGatedRank, limit)
}

// GetDailyRankWithLogicalExpire 获取每日热榜，支持逻辑过期
func (c *articleCache) GetDailyRankWithLogicalExpire(ctx context.Context, limit int64) ([]domain.Article, bool, error) {
	data, err := c.client.Get(ctx, KeyHotDailyAggreGatedRank+"_logical").Bytes()
	if err == nil {
		var wrapper cache.DataWithLogicalExpire
		if err := json.Unmarshal(data, &wrapper); err == nil {
			articlesJSON, _ := json.Marshal(wrapper.Data)
			var articles []domain.Article
			if err := json.Unmarshal(articlesJSON, &articles); err == nil {
				return articles, wrapper.IsLogicalExpired(), nil
			}
		}
	}

	return nil, false, redis.Nil
}

// SetDailyRankWithLogicalExpire 设置每日热榜，使用逻辑过期
func (c *articleCache) SetDailyRankWithLogicalExpire(ctx context.Context, articles []domain.Article, ttl time.Duration) error {
	wrapper := cache.NewDataWithLogicalExpire(articles, ttl)
	data, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, KeyHotDailyAggreGatedRank+"_logical", data, 24*time.Hour).Err()
}

func (c *articleCache) fetchRankFromKey(ctx context.Context, key string, limit int64) ([]domain.Article, error) {
	zRes, err := c.client.ZRevRangeWithScores(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	res := make([]domain.Article, 0, len(zRes))
	for _, z := range zRes {
		aid, _ := strconv.ParseInt(z.Member.(string), 10, 64)
		res = append(res, domain.Article{
			ID:    aid,
			Likes: int64(z.Score),
		})
	}
	return res, nil
}

func (c *articleCache) IncrDailyRankScore(ctx context.Context, aid int64, scoreDelta float64) error {
	key := fmt.Sprintf(KeyHotDailyRaw, time.Now().Format("2006010215"))
	return c.client.ZIncrBy(ctx, key, scoreDelta, fmt.Sprintf("%d", aid)).Err()
}

func (c *articleCache) GetHistoryRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	if c.client.Exists(ctx, KeyHotHistoryRank).Val() > 0 {
		return c.fetchRankFromKey(ctx, KeyHotHistoryRank, limit)
	}
	return nil, domain.ErrCacheMiss
}

func (c *articleCache) SetHistoryRank(ctx context.Context, aids []int64, scores []float64) error {
	if len(aids) != len(scores) || len(aids) == 0 {
		return domain.ErrBadParamInput
	}

	zMem := make([]redis.Z, len(aids))
	for i := range zMem {
		zMem[i] = redis.Z{
			Score:  scores[i],
			Member: any(aids[i]),
		}
	}

	return c.client.ZAdd(ctx, KeyHotHistoryRank, zMem...).Err()
}

// SetHistoryRankWithLogicalExpire 设置历史热榜，使用逻辑过期
func (c *articleCache) SetHistoryRankWithLogicalExpire(ctx context.Context, aids []int64, scores []float64, ttl time.Duration) error {
	if len(aids) != len(scores) || len(aids) == 0 {
		return domain.ErrBadParamInput
	}

	// 构建Article列表
	articles := make([]domain.Article, len(aids))
	for i := range aids {
		articles[i] = domain.Article{
			ID:    aids[i],
			Likes: int64(scores[i]),
		}
	}

	wrapper := cache.NewDataWithLogicalExpire(articles, ttl)
	data, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, KeyHotHistoryRank+"_logical", data, 24*time.Hour).Err()
}

func (c *articleCache) GetLikeCount(ctx context.Context, aid int64) (int64, error) {
	var res int64 = 0
	resStr, err := c.client.Get(ctx, fmt.Sprintf(KeyLikesBuffer, aid)).Result()
	if errors.Is(err, redis.Nil) {
		return res, domain.ErrCacheMiss
	}
	if err != nil {
		logrus.Errorf("failed to get like counts in redis, aid: %d, err: %v", aid, err)
	} else {
		likes, err := strconv.ParseInt(resStr, 10, 64)
		if err != nil {
			logrus.Errorf("strconv.ParseInt failed: %v", err)
		} else {
			res = max(res, likes)
		}
	}
	return res, nil
}

func (c *articleCache) MGetLikeCounts(ctx context.Context, aids []int64) (map[int64]int64, error) {
	if len(aids) == 0 {
		return nil, nil
	}
	keys := make([]string, len(aids))
	for i, aid := range aids {
		keys[i] = fmt.Sprintf(KeyLikesBuffer, aid)
	}

	result, err := c.client.MGet(ctx, keys...).Result()

	if err != nil {
		return nil, err
	}
	res := make(map[int64]int64)
	for i, val := range result {
		if val == nil {
			res[aids[i]] = 0
			continue
		}

		valStr, ok := val.(string)
		if !ok {
			logrus.Errorf("invalid type in redis for like count, id: %d, val: %v", aids[i], val)
			res[aids[i]] = 0
			continue
		}

		likes, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			logrus.Errorf("failed to strconv.ParseInt in redis, id: %d, err: %v", aids[i], err)
			res[aids[i]] = 0
			continue
		}
		res[aids[i]] = likes
	}
	return res, nil
}

func (c *articleCache) IncrLikeCount(ctx context.Context, aid int64) (int64, error) {
	key := fmt.Sprintf(KeyLikesBuffer, aid)
	return c.client.Incr(ctx, key).Result()
}

func (c *articleCache) SetLikeCount(ctx context.Context, aid, likes int64) error {
	key := fmt.Sprintf(KeyLikesBuffer, aid)
	return c.client.Set(ctx, key, likes, 7*24*time.Hour).Err()
}

func (c *articleCache) MSetLikeCount(ctx context.Context, aids, likes []int64) error {
	if len(aids) != len(likes) {
		return domain.ErrBadParamInput
	}
	if len(aids) == 0 {
		return nil
	}

	val := make([]any, 0, 2*len(aids))

	for i, aid := range aids {
		key := fmt.Sprintf(KeyLikesBuffer, aid)
		val = append(val, key, likes[i])
	}
	return c.client.MSet(ctx, val...).Err()
}
