package mysql

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/repository"
	"github.com/bxcodec/go-clean-arch/internal/repository/mysql/model"
)

type articleRepository struct {
	DB *gorm.DB
}

var _ domain.ArticleRepository = (*articleRepository)(nil)

// NewArticleRepository will create an object that represent the article.Repository interface
func NewArticleRepository(db *gorm.DB) *articleRepository {
	return &articleRepository{db}
}

// TODO 从数据库中拿文章时应该使用连表查询把user信息也查出来

func (m *articleRepository) Fetch(ctx context.Context, cursor string, num int64) (res []domain.Article, nextCursor string, err error) {
	var articles []model.Article
	decodedCursor, err := repository.DecodeCursor(cursor)
	if err != nil && cursor != "" {
		return nil, "", domain.ErrBadParamInput
	}

	repository.PageVerify(&num)
	err = m.DB.WithContext(ctx).Where("created_at > ?", decodedCursor).
		Order("created_at").
		Limit(int(num)).
		Find(&articles).
		Error

	if err != nil {
		return
	}

	for _, article := range articles {
		res = append(res, article.ToDomain())
	}
	if len(res) == int(num) {
		nextCursor = repository.EncodeCursor(res[len(res)-1].CreatedAt)
	}
	return
}

func (m *articleRepository) GetByID(ctx context.Context, id int64) (res domain.Article, err error) {
	var article model.Article
	err = m.DB.WithContext(ctx).First(&article, "id = ?", id).Error
	if err != nil {
		return res, domain.ErrNotFound
	}
	res = article.ToDomain()
	return
}

func (m *articleRepository) GetByTitle(ctx context.Context, title string) (res domain.Article, err error) {
	var article model.Article
	err = m.DB.WithContext(ctx).First(&article, "title = ?", title).Error
	if err != nil {
		return res, domain.ErrNotFound
	}
	res = article.ToDomain()
	return
}

func (m *articleRepository) Store(ctx context.Context, a *domain.Article) (err error) {
	articleModel := model.NewArticleFromDomain(a)
	result := m.DB.WithContext(ctx).Create(&articleModel)
	if result.Error != nil {
		return result.Error
	}
	a.ID = articleModel.ID
	a.CreatedAt = articleModel.CreatedAt
	a.UpdatedAt = articleModel.UpdatedAt
	return
}

func (m *articleRepository) Delete(ctx context.Context, id int64) error {
	result := m.DB.WithContext(ctx).Delete(&model.Article{}, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (m *articleRepository) Update(ctx context.Context, ar *domain.Article) (err error) {
	articleModel := model.NewArticleFromDomain(ar)
	result := m.DB.WithContext(ctx).Model(&articleModel).Updates(&articleModel)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return
}

func (m *articleRepository) AddViews(ctx context.Context, id int64, deltaViews int64) (err error) {
	result := m.DB.WithContext(ctx).Model(&model.Article{}).Where("id = ?", id).Update("views", gorm.Expr("views + ?", deltaViews))
	if result.Error != nil {
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (m *articleRepository) AddLikes(ctx context.Context, id int64, deltaLikes int64) error {
	result := m.DB.WithContext(ctx).Model(&model.Article{}).Where("id = ?", id).Update("likes", gorm.Expr("likes + ?", deltaLikes))
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (m *articleRepository) AddLikeRecord(ctx context.Context, articleID int64, userID int64) error {
	userLike := &model.UserLike{
		UserID:    userID,
		ArticleID: articleID,
	}
	result := m.DB.WithContext(ctx).Create(userLike)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrConflict
	}
	return nil
}

func (m *articleRepository) RemoveLikeRecord(ctx context.Context, articleID int64, userID int64) error {
	result := m.DB.WithContext(ctx).
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Delete(&model.UserLike{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (m *articleRepository) FetchByLikes(ctx context.Context, limit int) ([]domain.Article, error) {
	var articles []model.Article
	err := m.DB.WithContext(ctx).Model(&model.Article{}).Limit(limit).Find(&articles).Error
	if err != nil {
		return nil, err
	}
	res := make([]domain.Article, len(articles))
	for i, model := range articles {
		res[i] = model.ToDomain()
	}
	return res, nil
}

func (m *articleRepository) GetLikedUsers(ctx context.Context, id int64) ([]int64, error) {
	var res []int64
	err := m.DB.WithContext(ctx).
		Model(&model.UserLike{}).
		Where("article_id = ?", id).
		Pluck("user_id", &res).
		Error

	return res, err
}

func (m *articleRepository) GetByIDs(ctx context.Context, ids []int64) ([]domain.Article, error) {
	var articles []model.Article
	err := m.DB.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&articles).Error
	if err != nil {
		return nil, err
	}

	res := make([]domain.Article, len(articles))
	for i, model := range articles {
		res[i] = model.ToDomain()
	}

	if len(res) < len(ids) {
		err = domain.ErrNotFound
	}
	return res, err
}

func (m *articleRepository) ApplyLikeChanges(ctx context.Context, changes domain.LikeStateChanges) error {
	return m.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(changes.ToRemove) > 0 {
			aid := make([]int64, len(changes.ToRemove))
			uid := make([]int64, len(changes.ToRemove))
			for i := range changes.ToRemove {
				aid[i] = changes.ToRemove[i].ArticleID
				uid[i] = changes.ToRemove[i].UserID
			}
			if err := tx.
				Where("article_id = ? AND user_id IN ?", aid, uid).
				Delete(&model.UserLike{}).Error; err != nil {
				return err
			}
		}

		if len(changes.ToAdd) > 0 {
			recs := make([]model.UserLike, 0, len(changes.ToAdd))
			for _, d := range changes.ToAdd {
				recs = append(recs, model.NewUserLikeFromDomain(d))
			}

			err := tx.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&recs).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (m *articleRepository) FetchUserLikedArticles(ctx context.Context, uid int64, limit int64) ([]int64, error) {
	var res []int64
	err := m.DB.WithContext(ctx).
		Model(&model.UserLike{}).
		Select("article_id").
		Where("user_id = ?", uid).
		Order("article_id desc").
		Limit(int(limit)).
		Find(&res).Error

	return res, err
}

func (m *articleRepository) FetchArticlesByLikes(ctx context.Context, limit int64) ([]domain.Article, error) {
	var res []model.Article
	err := m.DB.WithContext(ctx).Model(&model.Article{}).Order("likes desc").Limit(int(limit)).Find(&res).Error
	ars := make([]domain.Article, len(res))
	for i := range res {
		ars[i] = res[i].ToDomain()
	}
	return ars, err
}
