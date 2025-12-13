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
	articleRepo  domain.ArticleRepository
	userRepo     domain.UserRepository
	articleCache domain.ArticleCache
}

// NewService will create a new article service object
func NewService(a domain.ArticleRepository, u domain.UserRepository, ac domain.ArticleCache) *Service {
	return &Service{
		articleRepo:  a,
		userRepo:     u,
		articleCache: ac,
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
	res, err = a.articleCache.Get(ctx, id)
	shouldSetCache := false

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
		shouldSetCache = true
	}

	newViews, errInrc := a.articleCache.Incr(context.Background(), id)
	if errInrc != nil {
		logrus.Warnf("redis incr error: %v", errInrc)
	} else {
		res.Views = newViews
	}

	go func() {
		bgCtx := context.Background()

		if errInrc == nil && newViews%10 == 0 {
			_ = a.articleRepo.UpdateViews(bgCtx, id, newViews)
		}

		if shouldSetCache {
			_ = a.articleCache.Set(bgCtx, &res)
		}
	}()

	return res, nil
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
	err = a.articleCache.Del(ctx, id)
	if err != nil {
		return
	}
	return
}

func (a *Service) UpdateViews(ctx context.Context, id int64, newViews int64) error {
	return a.articleRepo.UpdateViews(ctx, id, newViews)
}
