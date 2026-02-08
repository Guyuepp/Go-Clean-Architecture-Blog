package article

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
)

type service struct {
	articleRepo     domain.ArticleRepository
	articleCache    domain.ArticleCache
	syncLikesWorker domain.SyncLikesWorker
	bloomRepo       domain.BloomRepository
}

var _ domain.ArticleUsecase = (*service)(nil)

// NewService 创建article usecase服务
// 注意：articleCache仅用于点赞等特殊缓存操作，一般的缓存逻辑由repository层处理
func NewService(a domain.ArticleRepository, ac domain.ArticleCache, s domain.SyncLikesWorker, b domain.BloomRepository) *service {
	return &service{
		articleRepo:     a,
		articleCache:    ac,
		syncLikesWorker: s,
		bloomRepo:       b,
	}
}

// Fetch 获取文章列表
func (a *service) Fetch(ctx context.Context, cursor string, num int64) ([]domain.Article, string, error) {
	articles, err := a.articleRepo.Fetch(ctx, cursor, num)
	if err != nil {
		return nil, "", err
	}

	if len(articles) == 0 {
		return articles, "", nil
	}

	// 生成下一个cursor
	nextCursor := encodeCursor(articles[len(articles)-1].CreatedAt)
	return articles, nextCursor, nil
}

// GetByID 根据ID获取文章（所有缓存逻辑由repository层处理）
func (a *service) GetByID(ctx context.Context, id int64) (domain.Article, error) {
	if err := a.mustExists(ctx, id); err != nil {
		return domain.Article{}, err
	}

	return a.articleRepo.GetByID(ctx, id)
}

// Update 更新文章
func (a *service) Update(ctx context.Context, ar *domain.Article) error {
	if err := a.mustExists(ctx, ar.ID); err != nil {
		return err
	}
	ar.UpdatedAt = time.Now()
	return a.articleRepo.Update(ctx, ar)
}

// Store 创建文章
func (a *service) Store(ctx context.Context, m *domain.Article) error {
	// 检查标题是否已存在
	existedArticle, _ := a.articleRepo.GetByTitle(ctx, m.Title)
	if existedArticle.ID != 0 {
		return domain.ErrConflict
	}

	err := a.articleRepo.Store(ctx, m)
	if err != nil {
		return err
	}

	// 添加到布隆过滤器
	a.bloomRepo.Add(ctx, m.ID)

	return nil
}

// Delete 删除文章
func (a *service) Delete(ctx context.Context, id int64) error {
	if err := a.mustExists(ctx, id); err != nil {
		return err
	}

	return a.articleRepo.Delete(ctx, id)
}

// AddLikeRecord 添加点赞记录
func (a *service) AddLikeRecord(ctx context.Context, likeRecord domain.UserLike) (bool, error) {
	if err := a.mustExists(ctx, likeRecord.ArticleID); err != nil {
		return false, err
	}

	// 尝试从缓存添加点赞
	ok, err := a.articleCache.AddLikeRecord(ctx, likeRecord)
	if err != nil {
		if errors.Is(err, domain.ErrCacheMiss) {
			// 缓存未命中，从数据库加载用户点赞列表
			likedArticles, err := a.articleRepo.FetchUserLikedArticles(ctx, likeRecord.UserID, domain.LikeRecordLimit)
			if err != nil {
				logrus.Errorf("failed to FetchUserLikedArticles: %v", err)
				return false, err
			}

			// 更新缓存
			err = a.articleCache.SetUserLikedArticles(ctx, likeRecord.UserID, likedArticles)
			if err != nil {
				logrus.Errorf("failed to SetUserLikedArticles: %v", err)
				return false, err
			}

			// 重试
			ok, err = a.articleCache.AddLikeRecord(ctx, likeRecord)
			if err != nil {
				logrus.Errorf("failed to AddLikeRecord after cache reload: %v", err)
				return false, err
			}
		} else {
			logrus.Errorf("failed to AddLikeRecord: %v", err)
			return false, err
		}
	}

	// 发送到worker异步同步到数据库
	if ok {
		a.syncLikesWorker.Send(likeRecord, domain.Like)
	}

	return ok, nil
}

// RemoveLikeRecord 移除点赞记录
func (a *service) RemoveLikeRecord(ctx context.Context, likeRecord domain.UserLike) (bool, error) {
	if err := a.mustExists(ctx, likeRecord.ArticleID); err != nil {
		return false, err
	}

	// 尝试从缓存移除点赞
	ok, err := a.articleCache.DecrLikeRecord(ctx, likeRecord)
	if err != nil {
		if errors.Is(err, domain.ErrCacheMiss) {
			// 缓存未命中
			likedArticles, err := a.articleRepo.FetchUserLikedArticles(ctx, likeRecord.UserID, domain.LikeRecordLimit)
			if err != nil {
				logrus.Errorf("failed to FetchUserLikedArticles: %v", err)
				return false, err
			}

			// 更新缓存
			err = a.articleCache.SetUserLikedArticles(ctx, likeRecord.UserID, likedArticles)
			if err != nil {
				logrus.Errorf("failed to SetUserLikedArticles: %v", err)
				return false, err
			}

			// 重试
			ok, err = a.articleCache.DecrLikeRecord(ctx, likeRecord)
			if err != nil {
				logrus.Errorf("failed to DecrLikeRecord after cache reload: %v", err)
				return false, err
			}
		} else {
			logrus.Errorf("failed to DecrLikeRecord: %v", err)
			return false, err
		}
	}

	// 发送到worker异步同步到数据库
	if ok {
		a.syncLikesWorker.Send(likeRecord, domain.Unlike)
	}

	return ok, nil
}

// FetchDailyRank 获取每日热榜
func (a *service) FetchDailyRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	return a.articleRepo.GetDailyRank(ctx, limit)
}

// FetchHistoryRank 获取历史热榜
func (a *service) FetchHistoryRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	return a.articleRepo.GetHistoryRank(ctx, limit)
}

// InitBloomFilter 初始化布隆过滤器
func (a *service) InitBloomFilter(ctx context.Context) error {
	const (
		BatchSize   = 2000
		WorkerCount = 5
	)

	idBatchChan := make(chan []int64, WorkerCount*2)
	g, ctx := errgroup.WithContext(ctx)

	// 启动消费者（Redis Writers）
	for range WorkerCount {
		g.Go(func() error {
			for ids := range idBatchChan {
				if err := a.bloomRepo.BulkAdd(ctx, ids); err != nil {
					return err
				}
			}
			return nil
		})
	}

	// 启动生产者
	g.Go(func() error {
		defer close(idBatchChan)
		var cursor int64 = 0
		for {
			ids, err := a.articleRepo.FetchIDs(ctx, cursor, BatchSize)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				break
			}

			select {
			case idBatchChan <- ids:
				cursor = ids[len(ids)-1]
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// 等待完成
	if err := g.Wait(); err != nil {
		logrus.Errorf("bloom filter init failed: %v", err)
		return err
	}
	return nil
}

// mustExists 检查文章是否存在
func (a *service) mustExists(ctx context.Context, id int64) error {
	exists, err := a.bloomRepo.Exists(ctx, id)
	if err == nil && !exists {
		return domain.ErrNotFound
	}
	return nil
}

// encodeCursor 编码cursor
func encodeCursor(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}
