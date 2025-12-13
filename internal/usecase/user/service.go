package user

import (
	"context"
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	userRepo  domain.UserRepository
	jwtSecret []byte
	ttl       time.Duration
}

func NewService(r domain.UserRepository, jwtSecret []byte, ttl time.Duration) *Service {
	return &Service{
		userRepo:  r,
		jwtSecret: jwtSecret,
		ttl:       ttl,
	}
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *Service) Register(ctx context.Context, name, username, password string) error {
	existingUser, err := s.userRepo.GetByUsername(ctx, username)
	if err == nil && existingUser.ID != 0 {
		return domain.ErrUserAlreadyExists
	}

	if password == "" {
		password = "123456"
	}
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return err
	}

	user := &domain.User{
		Name:     name,
		Username: username,
		Password: hashedPassword,
	}
	return s.userRepo.Insert(ctx, user)
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", domain.ErrUserNotFound
	}
	if !checkPasswordHash(password, user.Password) {
		return "", domain.ErrBadParamInput
	}

	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) generateJWT(userID int64, username string) (string, error) {
	// 定义 Claims (载荷)
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(s.ttl).Unix(),
		"iat":      time.Now().Unix(),
	}

	// 创建 Token 对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名并生成字符串
	return token.SignedString(s.jwtSecret)
}

func (s *Service) EditPassword(ctx context.Context, id int64, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrUserNotFound
	}
	if !checkPasswordHash(oldPassword, user.Password) {
		return domain.ErrInvalidCredentials
	}

	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.userRepo.Update(ctx, &user)
}
