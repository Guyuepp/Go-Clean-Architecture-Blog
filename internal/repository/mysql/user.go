package mysql

import (
	"context"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository/mysql/model"
	"gorm.io/gorm"
)

type userRepository struct {
	DB *gorm.DB
}

var _ domain.UserRepository = (*userRepository)(nil)

// NewMysqlUserRepository will create an implementation of user.Repository
func NewUserRepository(db *gorm.DB) *userRepository {
	return &userRepository{
		DB: db,
	}
}

func (m *userRepository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	var user model.User
	if err := m.DB.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return domain.User{}, err
	}

	return user.ToDomain(), nil
}

func (m *userRepository) Insert(ctx context.Context, a *domain.User) error {
	userModel := model.NewUserFromDomain(a)

	result := m.DB.WithContext(ctx).Create(&userModel)
	if result.Error != nil {
		return result.Error
	}

	a.ID = userModel.ID

	return nil
}

func (m *userRepository) Update(ctx context.Context, a *domain.User) error {
	userModel := model.NewUserFromDomain(a)

	err := m.DB.WithContext(ctx).Model(&userModel).Updates(&userModel).Error
	return err
}

func (m *userRepository) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	var user model.User
	if err := m.DB.WithContext(ctx).First(&user, "username = ?", username).Error; err != nil {
		return domain.User{}, err
	}

	return user.ToDomain(), nil
}

func (m *userRepository) GetByIDs(ctx context.Context, uids []int64) ([]domain.User, error) {
	var users []model.User
	err := m.DB.WithContext(ctx).Model(&model.User{}).Where("id in ?", uids).Find(&users).Error
	res := make([]domain.User, len(users))
	for i := range users {
		res[i] = users[i].ToDomain()
	}
	return res, err
}
