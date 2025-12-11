package mysql

import (
	"context"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/repository/mysql/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

// NewMysqlUserRepository will create an implementation of user.Repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (m *UserRepository) GetByID(ctx context.Context, id int64) (domain.Author, error) {
	var user model.User
	if err := m.DB.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return domain.Author{}, err
	}

	author := user.ToDomain()
	return author, nil
}

func (m *UserRepository) Insert(ctx context.Context, a *domain.Author) error {
	userModel := model.NewUserFromDomain(a)

	result := m.DB.WithContext(ctx).Create(&userModel)
	if result.Error != nil {
		return result.Error
	}

	a.ID = userModel.ID

	return nil
}

func (m *UserRepository) Update(ctx context.Context, a *domain.Author) error {
	userModel := model.NewUserFromDomain(a)

	err := m.DB.WithContext(ctx).Model(&userModel).Updates(&userModel).Error
	return err
}

func (m *UserRepository) GetByUsername(ctx context.Context, username string) (domain.Author, error) {
	var user model.User
	if err := m.DB.WithContext(ctx).First(&user, "username = ?", username).Error; err != nil {
		return domain.Author{}, err
	}

	author := user.ToDomain()
	return author, nil
}
