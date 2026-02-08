package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
)

// articleRepository 协调层，协调缓存和数据库
type articleRepository struct {
	db            domain.ArticleDBRepository
	cache         domain.ArticleCache
	userRepo      domain.UserRepository
	rebuildGroup  singleflight.Group
	rankGroup     singleflight.Group
	mu            sync.Mutex
	rebuildingMap map[int64]bool // 正在重建的文章ID
}

var _ domain.ArticleRepository = (*articleRepository)(nil)

// NewArticleRepository 创建协调层repository
func NewArticleRepository(db domain.ArticleDBRepository, cache domain.ArticleCache, userRepo domain.UserRepository) *articleRepository {
	return &articleRepository{
		db:            db,
		cache:         cache,
		userRepo:      userRepo,
		rebuildingMap: make(map[int64]bool),
	}
}

// Fetch 获取文章列表
func (r *articleRepository) Fetch(ctx context.Context, cursor string, num int64) ([]domain.Article, error) {
	if cursor == "" {
		articles, expired, err := r.cache.GetHomeWithLogicalExpire(ctx)
		if err == nil {
			if expired {
				go r.rebuildHomeCache(context.Background(), num)
			}
			return articles, nil
		}
	}

	// 从数据库获取
	articles, err := r.db.Fetch(ctx, cursor, num)
	if err != nil {
		return nil, err
	}

	// 填充用户信息
	articles, err = r.fillUserDetails(ctx, articles)
	if err != nil {
		return nil, err
	}

	// 如果是首页，异步更新缓存
	if cursor == "" {
		go func(data []domain.Article) {
			_ = r.cache.SetHomeWithLogicalExpire(context.Background(), data, 30*time.Second)
		}(articles)
	}

	return articles, nil
}

// GetByID 根据ID获取文章，使用逻辑过期策略避免缓存击穿
func (r *articleRepository) GetByID(ctx context.Context, id int64) (domain.Article, error) {
	// 1. 先从缓存获取
	article, expired, err := r.cache.GetArticleWithLogicalExpire(ctx, id)
	if err == nil {
		// 缓存命中
		if expired {
			go r.rebuildArticleCache(context.Background(), id)
		}

		// 更新浏览量（先增加缓存中的浏览量）
		deltaViews, _ := r.cache.IncrViews(ctx, id)
		article.Views += deltaViews

		// 获取最新的点赞数
		newLikes, err := r.cache.GetLikeCount(ctx, id)
		if err == nil {
			article.Likes = newLikes
		}

		return article, nil
	}

	// 2. 缓存未命中，使用singleflight避免缓存击穿
	key := "article:" + string(rune(id))
	result, err, _ := r.rebuildGroup.Do(key, func() (interface{}, error) {
		// 从数据库加载
		art, err := r.db.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}

		// 填充用户信息
		user, err := r.userRepo.GetByID(ctx, art.User.ID)
		if err != nil {
			return nil, err
		}
		art.User = user

		// 更新缓存（使用逻辑过期）
		_ = r.cache.SetArticleWithLogicalExpire(context.Background(), &art, 10*time.Minute)

		// 初始化点赞数缓存
		_ = r.cache.SetLikeCount(ctx, art.ID, art.Likes)

		return art, nil
	})

	if err != nil {
		return domain.Article{}, err
	}

	article = result.(domain.Article)

	// 更新浏览量
	deltaViews, _ := r.cache.IncrViews(ctx, id)
	article.Views += deltaViews

	return article, nil
}

// GetByIDs 批量获取文章
func (r *articleRepository) GetByIDs(ctx context.Context, ids []int64) ([]domain.Article, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 先从缓存批量获取
	cachedArticles, err := r.cache.GetArticleByIDsWithLogicalExpire(ctx, ids)
	if err == nil && len(cachedArticles) == len(ids) {
		// 全部命中
		return cachedArticles, nil
	}

	// 部分未命中，从数据库获取
	articles, err := r.db.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// 填充用户信息
	articles, err = r.fillUserDetails(ctx, articles)
	if err != nil {
		return nil, err
	}

	// 异步更新缓存
	go func(arts []domain.Article) {
		_ = r.cache.BatchSetArticleWithLogicalExpire(context.Background(), arts, 10*time.Minute)
	}(articles)

	return articles, nil
}

// GetByTitle 根据标题获取文章
func (r *articleRepository) GetByTitle(ctx context.Context, title string) (domain.Article, error) {
	// 直接从数据库查询（标题查询不常用，不走缓存）
	article, err := r.db.GetByTitle(ctx, title)
	if err != nil {
		return domain.Article{}, err
	}

	// 填充用户信息
	user, err := r.userRepo.GetByID(ctx, article.User.ID)
	if err != nil {
		return domain.Article{}, err
	}
	article.User = user

	return article, nil
}

// Store 创建文章
func (r *articleRepository) Store(ctx context.Context, a *domain.Article) error {
	return r.db.Store(ctx, a)
}

// Update 更新文章
func (r *articleRepository) Update(ctx context.Context, ar *domain.Article) error {
	err := r.db.Update(ctx, ar)
	if err != nil {
		return err
	}

	// 异步删除缓存
	go func(id int64) {
		_ = r.cache.DeleteArticle(context.Background(), id)
	}(ar.ID)

	return nil
}

// Delete 删除文章
func (r *articleRepository) Delete(ctx context.Context, id int64) error {
	err := r.db.Delete(ctx, id)
	if err != nil {
		return err
	}

	// 异步删除缓存
	go func(id int64) {
		_ = r.cache.DeleteArticle(context.Background(), id)
	}(id)

	return nil
}

// AddViews 增加浏览量（这个方法在新架构下由worker处理）
func (r *articleRepository) AddViews(ctx context.Context, id int64, deltaViews int64) error {
	return r.db.AddViews(ctx, id, deltaViews)
}

// AddLikes 增加点赞数
func (r *articleRepository) AddLikes(ctx context.Context, id int64, deltaLikes int64) error {
	return r.db.AddLikes(ctx, id, deltaLikes)
}

// FetchUserLikedArticles 获取用户点赞的文章列表
func (r *articleRepository) FetchUserLikedArticles(ctx context.Context, uid int64, limit int64) ([]int64, error) {
	return r.db.FetchUserLikedArticles(ctx, uid, limit)
}

// ApplyLikeChanges 应用点赞变更
func (r *articleRepository) ApplyLikeChanges(ctx context.Context, changes domain.LikeStateChanges) error {
	return r.db.ApplyLikeChanges(ctx, changes)
}

// FetchArticlesByLikes 按点赞数获取文章
func (r *articleRepository) FetchArticlesByLikes(ctx context.Context, limit int64) ([]domain.Article, error) {
	return r.db.FetchArticlesByLikes(ctx, limit)
}

// FetchIDs 获取文章ID列表
func (r *articleRepository) FetchIDs(ctx context.Context, cursor, limit int64) ([]int64, error) {
	return r.db.FetchIDs(ctx, cursor, limit)
}

// fillUserDetails 批量填充用户详细信息
func (r *articleRepository) fillUserDetails(ctx context.Context, articles []domain.Article) ([]domain.Article, error) {
	if len(articles) == 0 {
		return articles, nil
	}

	// 收集所有不重复的UserID
	userIDs := make([]int64, 0, len(articles))
	existMap := make(map[int64]bool)
	for _, item := range articles {
		if !existMap[item.User.ID] {
			userIDs = append(userIDs, item.User.ID)
			existMap[item.User.ID] = true
		}
	}

	// 批量查询用户
	users, err := r.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// 转成Map方便查找
	userMap := make(map[int64]domain.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 填充回Article
	for i := range articles {
		if u, ok := userMap[articles[i].User.ID]; ok {
			articles[i].User = u
		}
	}

	return articles, nil
}

// rebuildHomeCache 异步重建首页缓存
func (r *articleRepository) rebuildHomeCache(ctx context.Context, num int64) {
	_, err, _ := r.rebuildGroup.Do("home", func() (any, error) {
		articles, err := r.db.Fetch(ctx, "", num)
		if err != nil {
			logrus.Errorf("failed to rebuild home cache from db: %v", err)
			return nil, err
		}

		articles, err = r.fillUserDetails(ctx, articles)
		if err != nil {
			logrus.Errorf("failed to fill user details: %v", err)
			return nil, err
		}

		err = r.cache.SetHomeWithLogicalExpire(ctx, articles, 30*time.Second)
		if err != nil {
			logrus.Errorf("failed to set home cache: %v", err)
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		logrus.Errorf("rebuildHomeCache failed: %v", err)
	}
}

// rebuildArticleCache 异步重建文章缓存
func (r *articleRepository) rebuildArticleCache(ctx context.Context, id int64) {
	// 检查是否已经在重建中
	r.mu.Lock()
	if r.rebuildingMap[id] {
		r.mu.Unlock()
		return
	}
	r.rebuildingMap[id] = true
	r.mu.Unlock()

	// 完成后清除标记
	defer func() {
		r.mu.Lock()
		delete(r.rebuildingMap, id)
		r.mu.Unlock()
	}()

	// 使用singleflight避免并发重建
	key := "rebuild:" + string(rune(id))
	_, err, _ := r.rebuildGroup.Do(key, func() (any, error) {
		article, err := r.db.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				// 文章不存在，删除缓存
				_ = r.cache.DeleteArticle(ctx, id)
			}
			return nil, err
		}

		// 填充用户信息
		user, err := r.userRepo.GetByID(ctx, article.User.ID)
		if err != nil {
			logrus.Errorf("failed to get user: %v", err)
			return nil, err
		}
		article.User = user

		// 更新缓存
		err = r.cache.SetArticleWithLogicalExpire(ctx, &article, 10*time.Minute)
		if err != nil {
			logrus.Errorf("failed to set article cache: %v", err)
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		logrus.Errorf("rebuildArticleCache failed for id %d: %v", id, err)
	}
}

// GetDailyRank 获取每日热榜
func (r *articleRepository) GetDailyRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	// 先尝试从缓存获取
	articles, err := r.cache.GetDailyRank(ctx, limit)
	if err == nil {
		return r.fillRankArticles(ctx, articles)
	}

	// 缓存未命中
	result, err, _ := r.rankGroup.Do("daily", func() (any, error) {
		return r.buildDailyRank(ctx, limit)
	})

	if err != nil {
		return nil, err
	}

	return result.([]domain.Article), nil
}

// GetHistoryRank 获取历史热榜
func (r *articleRepository) GetHistoryRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	articles, err := r.cache.GetHistoryRank(ctx, limit)
	if err == nil {
		// 填充完整文章信息
		return r.fillRankArticles(ctx, articles)
	}

	// 缓存未命中
	result, err, _ := r.rankGroup.Do("history", func() (any, error) {
		return r.buildHistoryRank(ctx, limit)
	})

	if err != nil {
		return nil, err
	}

	return result.([]domain.Article), nil
}

// buildDailyRank 构建每日热榜
func (r *articleRepository) buildDailyRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	// // 从数据库按点赞数获取
	// articles, err := r.db.FetchArticlesByLikes(ctx, limit)
	// if err != nil {
	// 	return nil, err
	// }

	// // 填充用户信息
	// articles, err = r.fillUserDetails(ctx, articles)
	// if err != nil {
	// 	return nil, err
	// }

	// // 更新缓存（逻辑过期，5分钟TTL）
	// go func(arts []domain.Article) {
	// 	_ = r.cache.SetDailyRankWithLogicalExpire(context.Background(), arts, 5*time.Minute)
	// }(articles)

	// return articles, nil

	panic("Unreachable: unimplement")
}

// buildHistoryRank 构建历史热榜
func (r *articleRepository) buildHistoryRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	// 从数据库按点赞数获取
	articles, err := r.db.FetchArticlesByLikes(ctx, limit)
	if err != nil {
		return nil, err
	}

	// 填充用户信息
	articles, err = r.fillUserDetails(ctx, articles)
	if err != nil {
		return nil, err
	}

	// 准备缓存数据
	aids := make([]int64, len(articles))
	scores := make([]float64, len(articles))
	for i, art := range articles {
		aids[i] = art.ID
		scores[i] = float64(art.Likes)
	}

	// 更新缓存（使用逻辑过期，1小时TTL）
	go func() {
		_ = r.cache.SetHistoryRankWithLogicalExpire(context.Background(), aids, scores, 1*time.Hour)
	}()

	return articles, nil
}

// rebuildDailyRank 异步重建每日热榜
func (r *articleRepository) rebuildDailyRank(ctx context.Context, limit int64) {
	_, err, _ := r.rebuildGroup.Do("rebuild_daily", func() (any, error) {
		return r.buildDailyRank(ctx, limit)
	})

	if err != nil {
		logrus.Errorf("rebuildDailyRank failed: %v", err)
	}
}

// fillRankArticles 填充热榜文章的完整信息
func (r *articleRepository) fillRankArticles(ctx context.Context, rankArticles []domain.Article) ([]domain.Article, error) {
	if len(rankArticles) == 0 {
		return rankArticles, nil
	}

	// 提取文章ID
	ids := make([]int64, len(rankArticles))
	for i, art := range rankArticles {
		ids[i] = art.ID
	}

	// 批量从缓存/数据库获取完整文章信息
	articles, err := r.GetByIDs(ctx, ids)
	if err != nil {
		// 如果获取失败，返回基本的排名信息
		logrus.Warnf("failed to fill rank articles: %v", err)
		return rankArticles, nil
	}

	// 保持排名顺序，并合并点赞数
	articleMap := make(map[int64]domain.Article)
	for _, art := range articles {
		articleMap[art.ID] = art
	}

	result := make([]domain.Article, 0, len(rankArticles))
	for _, rankArt := range rankArticles {
		if fullArt, ok := articleMap[rankArt.ID]; ok {
			// 使用热榜中的点赞数（可能更新）
			fullArt.Likes = rankArt.Likes
			result = append(result, fullArt)
		} else {
			// 如果找不到完整信息，使用基本信息
			result = append(result, rankArt)
		}
	}

	return result, nil
}
