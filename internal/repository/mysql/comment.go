package mysql

import (
	"context"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository/mysql/model"
	"gorm.io/gorm"
)

type commentRepository struct {
	DB *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *commentRepository {
	return &commentRepository{
		DB: db,
	}
}

func (c *commentRepository) Delete(ctx context.Context, aid int64, uid int64) error {
	result := c.DB.WithContext(ctx).Model(&model.Comment{}).Where("article_id = ? AND user_id = ?", aid, uid).Delete(&model.Comment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrForbidden
	}
	return nil
}

func (c *commentRepository) FetchReplies(ctx context.Context, rootIDs []int64) ([]*domain.Comment, error) {
	var comments []model.Comment
	err := c.DB.WithContext(ctx).
		Where("root_id IN ?", rootIDs).
		Find(&comments).Error
	if err != nil {
		return nil, err
	}

	var res []*domain.Comment
	for _, comment := range comments {
		domainComment := comment.ToDomain()
		res = append(res, &domainComment)
	}
	return res, nil
}

func (c *commentRepository) FetchRoots(ctx context.Context, articleID int64, cursor string, limit int64) ([]*domain.Comment, error) {
	var comments []model.Comment
	decodedCursor, err := repository.DecodeCursor(cursor)
	if err != nil && cursor != "" {
		return nil, domain.ErrBadParamInput
	}
	err = c.DB.WithContext(ctx).
		Where("article_id = ? AND parent_id = 0 AND created_at > ?", articleID, decodedCursor).
		Limit(int(limit)).
		Order("created_at DESC").
		Find(&comments).Error
	if err != nil {
		return nil, err
	}

	var res []*domain.Comment
	for _, comment := range comments {
		domainComment := comment.ToDomain()
		res = append(res, &domainComment)
	}
	return res, nil
}

func (c *commentRepository) GetByID(ctx context.Context, id int64) (*domain.Comment, error) {
	var comment model.Comment
	err := c.DB.WithContext(ctx).First(&comment, "id = ?", id).Error
	if err != nil {
		return nil, domain.ErrNotFound
	}
	domainComment := comment.ToDomain()
	return &domainComment, nil
}

func (c *commentRepository) Store(ctx context.Context, comment *domain.Comment) error {
	err := c.DB.WithContext(ctx).Create(model.NewCommentFromDomain(comment)).Error
	if err != nil {
		return err
	}
	return nil
}

var _ domain.CommentRepository = (*commentRepository)(nil)
