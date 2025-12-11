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
