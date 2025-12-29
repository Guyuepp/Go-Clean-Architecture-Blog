package article

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/bxcodec/go-clean-arch/domain"
)

type Service struct {
	articleRepo     domain.ArticleRepository
	userRepo        domain.UserRepository
	articleCache    domain.ArticleCache
	syncLikesWorker domain.SyncLikesWorker
}

var _ domain.ArticleUsecase = (*Service)(nil)

// NewService will create a new article service object
func NewService(a domain.ArticleRepository, u domain.UserRepository, ac domain.ArticleCache, s domain.SyncLikesWorker) *Service {
	return &Service{
		articleRepo:     a,
		userRepo:        u,
		articleCache:    ac,
		syncLikesWorker: s,
	}
}

/*
* In this function below, I'm using errgroup with the pipeline pattern
* Look how this works in this package explanation
* in godoc: https://godoc.org/golang.org/x/sync/errgroup#ex-Group--Pipeline
 */
func (a *Service) fillUserDetails(ctx context.Context, data []domain.Article) ([]domain.Article, error) {
	g, ctx := errgroup.WithContext(ctx)
	// Get the User's id
	mapUsers := map[int64]domain.User{}

	for _, article := range data { //nolint
		mapUsers[article.User.ID] = domain.User{}
	}
	// Using goroutine to fetch the User's detail
	chanUser := make(chan domain.User)
	for UserID := range mapUsers {
		UserID := UserID
		g.Go(func() error {
			res, err := a.userRepo.GetByID(ctx, UserID)
			if err != nil {
				return err
			}
			chanUser <- res
			return nil
		})
	}

	go func() {
		defer close(chanUser)
		err := g.Wait()
		if err != nil {
			logrus.Error(err)
			return
		}

	}()

	for User := range chanUser {
		if User != (domain.User{}) {
			mapUsers[User.ID] = User
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// merge the User's data
	for index, item := range data { //nolint
		if a, ok := mapUsers[item.User.ID]; ok {
			data[index].User = a
		}
	}
	return data, nil
}

func (a *Service) Fetch(ctx context.Context, cursor string, num int64) (res []domain.Article, nextCursor string, err error) {
	res, nextCursor, err = a.articleRepo.Fetch(ctx, cursor, num)
	if err != nil {
		return nil, "", err
	}

	res, err = a.fillUserDetails(ctx, res)
	if err != nil {
		nextCursor = ""
	}
	return
}

func (a *Service) GetByID(ctx context.Context, id int64) (res domain.Article, err error) {
	res, err = a.articleCache.GetArticle(ctx, id)

	if err != nil {
		if !errors.Is(err, redis.Nil) {
			logrus.Warnf("cache get error: %v", err)
		}

		res, err = a.articleRepo.GetByID(ctx, id)
		if err != nil {
			return domain.Article{}, err
		}

		resUser, err := a.userRepo.GetByID(ctx, res.User.ID)
		if err != nil {
			return domain.Article{}, err
		}
		res.User = resUser

		go func(art domain.Article) {
			if err := a.articleCache.SetArticle(context.Background(), &art); err != nil {
				logrus.Warnf("failed to set cache: %v", err)
			}
		}(res)
	}

	newLikes, err := a.articleCache.GetLikeCount(ctx, id)
	if errors.Is(err, domain.ErrCacheMiss) {
		_ = a.articleCache.SetLikeCount(ctx, res.ID, res.Likes)
	} else if err != nil {
		logrus.Errorf("failed to GetLikeCount from redis: %v", err)
	} else {
		res.Likes = newLikes
	}

	deltaViews, err := a.articleCache.IncrViews(ctx, id)
	if err != nil {
		logrus.Errorf("failed to IncrViews from redis: %v", err)
		return res, err
	} else {
		res.Views += deltaViews
		return res, err
	}
}

func (a *Service) Update(ctx context.Context, ar *domain.Article) (err error) {
	ar.UpdatedAt = time.Now()
	return a.articleRepo.Update(ctx, ar)
}

func (a *Service) GetByTitle(ctx context.Context, title string) (res domain.Article, err error) {
	res, err = a.articleRepo.GetByTitle(ctx, title)
	if err != nil {
		return
	}

	resUser, err := a.userRepo.GetByID(ctx, res.User.ID)
	if err != nil {
		return domain.Article{}, err
	}

	res.User = resUser
	return
}

func (a *Service) Store(ctx context.Context, m *domain.Article) (err error) {
	existedArticle, _ := a.GetByTitle(ctx, m.Title) // ignore if any error
	if existedArticle != (domain.Article{}) {
		return domain.ErrConflict
	}

	err = a.articleRepo.Store(ctx, m)
	if err != nil {
		return
	}
	userDetail, err := a.userRepo.GetByID(ctx, m.User.ID)
	if err != nil {
		return
	}
	m.User.Name = userDetail.Name
	m.User.Username = userDetail.Username
	return
}

func (a *Service) Delete(ctx context.Context, id int64) (err error) {
	existedArticle, err := a.articleRepo.GetByID(ctx, id)
	if err != nil {
		return
	}
	if existedArticle == (domain.Article{}) {
		return domain.ErrNotFound
	}
	err = a.articleRepo.Delete(ctx, id)
	if err != nil {
		return
	}
	err = a.articleCache.DeleteArticle(ctx, id)
	if err != nil {
		return
	}
	return
}

func (a *Service) AddViews(ctx context.Context, id int64, deltaViews int64) error {
	return a.articleRepo.AddViews(ctx, id, deltaViews)
}

func (a *Service) AddLikeRecord(ctx context.Context, likeRecord domain.UserLike) (bool, error) {
	ok := false
	ok1, err := a.articleCache.AddLikeRecord(ctx, likeRecord)
	if err != nil {
		// 未命中缓存
		if errors.Is(err, domain.ErrCacheMiss) {
			// 去数据库加载这个用户喜欢哪些文章
			likedArticles, err := a.articleRepo.FetchUserLikedArticles(ctx, likeRecord.UserID, domain.LikeRecordLimit)
			if err != nil {
				logrus.Errorf("failed to FetchUserLikedArticles from repo: %v", err)
				return false, err
			}

			// 存入redis
			err = a.articleCache.SetUserLikedArticles(ctx, likeRecord.UserID, likedArticles)
			if err != nil {
				logrus.Errorf("failed to AddUserLikedArticles to redis: %v", err)
				return false, err
			}

			// 重新读
			ok2, err := a.articleCache.AddLikeRecord(ctx, likeRecord)

			if err != nil {
				logrus.Errorf("failed to AddLikeRecord to redis: %v", err)
				return false, err
			}

			if ok2 {
				ok = true
			}
		} else {
			// 未知错误
			logrus.Errorf("failed to AddLikeRecord to redis: %v", err)
			return false, err
		}
	}

	if ok1 {
		ok = true
	}

	if ok {
		a.syncLikesWorker.Send(likeRecord, domain.Like)
	}
	return ok, nil
}

func (a *Service) RemoveLikeRecord(ctx context.Context, likeRecord domain.UserLike) (bool, error) {
	ok := false
	ok1, err := a.articleCache.DecrLikeRecord(ctx, likeRecord)
	if err != nil {
		// 未命中缓存
		if errors.Is(err, domain.ErrCacheMiss) {
			// 去数据库加载这个用户喜欢哪些文章
			likedArticles, err := a.articleRepo.FetchUserLikedArticles(ctx, likeRecord.UserID, domain.LikeRecordLimit)
			if err != nil {
				logrus.Errorf("failed to FetchUserLikedArticles from repo: %v", err)
				return false, err
			}

			// 存入redis
			err = a.articleCache.SetUserLikedArticles(ctx, likeRecord.UserID, likedArticles)
			if err != nil {
				logrus.Errorf("failed to DecrUserLikedArticles to redis: %v", err)
				return false, err
			}

			// 重新读
			ok2, err := a.articleCache.DecrLikeRecord(ctx, likeRecord)

			if err != nil {
				logrus.Errorf("failed to DecrLikeRecord to redis: %v", err)
				return false, err
			}

			if ok2 {
				ok = true
			}
		} else {
			// 未知错误
			logrus.Errorf("failed to DecrLikeRecord to redis: %v", err)
			return false, err
		}
	}

	if ok1 {
		ok = true
	}

	if ok {
		a.syncLikesWorker.Send(likeRecord, domain.Unlike)
	}
	return ok, nil
}

func (a *Service) FetchDailyRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	res, err := a.articleCache.GetDailyRank(ctx, limit)
	if err != nil {
		logrus.Errorf("failed to GetDailyRank from redis: %v", err)
		return nil, err
	}

	mp := make(map[int64]domain.Article)
	ids := make([]int64, 0, len(res))
	for i := range res {
		mp[res[i].ID] = res[i]
		ids = append(ids, res[i].ID)
	}

	cacheRes, err := a.articleCache.GetArticleByIDs(ctx, ids)
	if err != nil {
		logrus.Warnf("failed to GetArticleByIDs from redis: %v", err)
	} else {
		for i := range cacheRes {
			mp[cacheRes[i].ID] = cacheRes[i]
		}
	}

	idsMissd := make([]int64, 0, len(res))
	for _, ar := range mp {
		idsMissd = append(idsMissd, ar.ID)
	}

	resRepo, err := a.articleRepo.GetByIDs(ctx, idsMissd)
	if err != nil {
		logrus.Warnf("failed to GetByIDs from repo: %v", err)
	} else {
		a.articleCache.BatchSetArticle(ctx, resRepo)
		for i := range resRepo {
			mp[resRepo[i].ID] = resRepo[i]
		}
	}
	for i := range res {
		ar := mp[res[i].ID]
		if ar.Title == "" {
			res[i].Title = "Not Found"
		} else {
			ar.Likes = res[i].Likes
			res[i] = ar
		}
	}
	return res, nil
}

func (a *Service) FetchHistoryRank(ctx context.Context, limit int64) ([]domain.Article, error) {
	res, err := a.articleCache.GetHistoryRank(ctx, limit)
	if errors.Is(err, domain.ErrCacheMiss) {
		res, err := a.articleRepo.FetchArticlesByLikes(ctx, 100) // NOTE 这里定义了默认取最多100篇
		if err != nil {
			logrus.Errorf("failed to FetchArticlesByLikes from repo: %v", err)
			return nil, err
		}
		ids := make([]int64, len(res))
		scores := make([]float64, len(res))
		for i := range res {
			ids[i] = res[i].ID
			scores[i] = float64(res[i].Likes)
		}

		err = a.articleCache.SetHistoryRank(ctx, ids, scores)
		if err != nil {
			logrus.Warnf("fail to SetHistoryRank to redis: %v", err)
		}

		return res[:min(int64(len(res)), limit)], nil
	} else if err != nil {
		logrus.Errorf("failed to GetHotRank from redis: %v", err)
		return nil, err
	}

	mp := make(map[int64]domain.Article)
	ids := make([]int64, len(res))
	for i := range res {
		mp[res[i].ID] = res[i]
		ids[i] = res[i].ID
	}

	cacheRes, err := a.articleCache.GetArticleByIDs(ctx, ids)
	if err != nil {
		logrus.Warnf("failed to GetArticleByIDs from redis: %v", err)
	} else {
		for i := range cacheRes {
			mp[cacheRes[i].ID] = cacheRes[i]
		}
	}

	idsMissd := make([]int64, 0, len(res))
	for _, ar := range mp {
		if ar.Title == "" {
			idsMissd = append(idsMissd, ar.ID)
		}
	}
	resRepo, err := a.articleRepo.GetByIDs(ctx, idsMissd)
	if err != nil {
		logrus.Warnf("failed to GetByIDs from repo: %v", err)
	} else {
		go a.articleCache.BatchSetArticle(context.Background(), resRepo)
		for i := range resRepo {
			mp[resRepo[i].ID] = resRepo[i]
		}
	}
	for i := range res {
		ar := mp[res[i].ID]
		if ar.Title == "" {
			res[i].Title = "Cannot find this article"
		} else {
			ar.Likes = res[i].Likes
			res[i] = ar
		}
	}
	return res, nil
}
