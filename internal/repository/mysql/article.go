package mysql

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/repository"
	"github.com/bxcodec/go-clean-arch/internal/repository/mysql/model"
)

type ArticleRepository struct {
	DB *gorm.DB
}

// NewArticleRepository will create an object that represent the article.Repository interface
func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db}
}

func (m *ArticleRepository) Fetch(ctx context.Context, cursor string, num int64) (res []domain.Article, nextCursor string, err error) {
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

func (m *ArticleRepository) GetByID(ctx context.Context, id int64) (res domain.Article, err error) {
	var article model.Article
	err = m.DB.WithContext(ctx).First(&article, "id = ?", id).Error
	if err != nil {
		return res, domain.ErrNotFound
	}
	res = article.ToDomain()
	return
}

func (m *ArticleRepository) GetByTitle(ctx context.Context, title string) (res domain.Article, err error) {
	var article model.Article
	err = m.DB.WithContext(ctx).First(&article, "title = ?", title).Error
	if err != nil {
		return res, domain.ErrNotFound
	}
	res = article.ToDomain()
	return
}

func (m *ArticleRepository) Store(ctx context.Context, a *domain.Article) (err error) {
	articleModel := model.NewArticleFromDomain(a)
	result := m.DB.WithContext(ctx).Create(&articleModel)
	if result.Error != nil {
		return result.Error
	}
	a.ID = articleModel.ID
	return
}

func (m *ArticleRepository) Delete(ctx context.Context, id int64) error {
	result := m.DB.WithContext(ctx).Delete(&model.Article{}, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (m *ArticleRepository) Update(ctx context.Context, ar *domain.Article) (err error) {
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

func (m *ArticleRepository) UpdateViews(ctx context.Context, id int64, newViews int64) (err error) {
	result := m.DB.WithContext(ctx).Model(&model.Article{}).Where("id = ?", id).Update("views", newViews)
	if result.Error != nil {
		return fmt.Errorf("failed to update views: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return
}
