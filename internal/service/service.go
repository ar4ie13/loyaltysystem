package service

import (
	"context"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	"github.com/ar4ie13/loyaltysystem/internal/models"
	"github.com/rs/zerolog"
)

type Service struct {
	repo Repository
	zlog zerolog.Logger
}

func NewService(repo Repository, zlog zerolog.Logger) *Service {
	return &Service{
		repo: repo,
		zlog: zlog,
	}
}

type Repository interface {
	CreateUser(ctx context.Context, user models.User) error
	GetUserByLogin(ctx context.Context, login string) (models.User, error)
}

func (s *Service) checkLoginString(login string) bool {
	// Check that only letters and digits are used for login
	for _, char := range login {
		if !(char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

func (s *Service) CreateUser(ctx context.Context, user models.User) error {

	if !s.checkLoginString(user.Login) {
		return apperrors.ErrInvalidLoginString
	}

	err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) LoginUser(ctx context.Context, login string) (models.User, error) {
	if !s.checkLoginString(login) {
		return models.User{}, apperrors.ErrInvalidLoginString
	}

	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}
