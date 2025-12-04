package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	"github.com/ar4ie13/loyaltysystem/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service is a main object of service layer
type Service struct {
	repo Repository
	zlog zerolog.Logger
}

// NewService constructs new service object
func NewService(repo Repository, zlog zerolog.Logger) *Service {
	return &Service{
		repo: repo,
		zlog: zlog,
	}
}

// Repository interface used to communicate with repository from service
type Repository interface {
	CreateUser(ctx context.Context, user models.User) error
	GetUserByLogin(ctx context.Context, login string) (models.User, error)
	PutUserOrder(ctx context.Context, user uuid.UUID, order string) error
	GetUserOrders(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error)
	GetBalance(ctx context.Context, user uuid.UUID) (models.User, error)
	PutUserWithdrawnOrder(ctx context.Context, user uuid.UUID, orderNum string, withdrawn float64) error
	GetUserWithdrawals(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error)
}

// checkLoginString is a helper to validate login string
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

// CreateUser used to create user by provided login and password hashed at handler's layer
func (s *Service) CreateUser(ctx context.Context, user models.User) error {

	if !s.checkLoginString(user.Login) {
		return apperrors.ErrInvalidLoginString
	}

	user.UUID = uuid.New()

	err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

// LoginUser used for logging users in
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

// PutUserOrder used to register user's order without withdrawn
func (s *Service) PutUserOrder(ctx context.Context, user uuid.UUID, order string) error {
	if !s.checkOrderNumber(order) {
		return apperrors.ErrIncorrectOrderNumber
	}

	err := s.repo.PutUserOrder(ctx, user, order)
	if err != nil {
		return err
	}
	return nil
}

// checkOrderNumber checks order number for Luhn algorithm compliance
func (s *Service) checkOrderNumber(order string) bool {
	if len(order) < 2 {
		return false
	}

	t := time.Now()
	digits := make([]int, len(order))
	for i, char := range order {
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			return false
		}
		digits[i] = digit
	}

	sum := 0
	isSecond := false

	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]

		if isSecond {
			digit = digit * 2
			if digit > 9 {
				digit = digit - 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}
	fmt.Println(time.Since(t))

	return sum%10 == 0
}

// GetUserOrders return all registered user's orders
func (s *Service) GetUserOrders(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error) {
	if userUUID == uuid.Nil {
		return nil, apperrors.ErrInvalidUserUUID
	}

	orders, err := s.repo.GetUserOrders(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetBalance return user's balance
func (s *Service) GetBalance(ctx context.Context, user uuid.UUID) (models.User, error) {
	var balance models.User
	balance, err := s.repo.GetBalance(ctx, user)
	if err != nil {
		return balance, err
	}
	return balance, nil
}

// PutUserWithdrawnOrder used for registering user's order with withdrawn
func (s *Service) PutUserWithdrawnOrder(ctx context.Context, user uuid.UUID, orderNum string, withdrawn float64) error {
	if withdrawn <= 0 {
		return fmt.Errorf("withdrawn must be greater than zero")
	}

	if user == uuid.Nil {
		return apperrors.ErrInvalidUserUUID
	}

	if !s.checkOrderNumber(orderNum) {
		return apperrors.ErrIncorrectOrderNumber
	}

	if err := s.repo.PutUserWithdrawnOrder(ctx, user, orderNum, withdrawn); err != nil {
		return err
	}

	return nil
}

// GetUserWithdrawals returns all user's orders with withdrawn
func (s *Service) GetUserWithdrawals(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error) {
	if userUUID == uuid.Nil {
		return nil, apperrors.ErrInvalidUserUUID
	}
	orders, err := s.repo.GetUserWithdrawals(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}
