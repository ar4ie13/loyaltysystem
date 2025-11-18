package apperrors

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidUserUUID     = errors.New("invalid user uuid")
	ErrUserIsNotAuthorized = errors.New("user is not authorized")
	ErrInvalidLoginString  = errors.New("invalid login string, use letters and digits only")
	ErrPasswordMinSymbols  = errors.New("password minimum symbols")
	ErrInvalidPassword     = errors.New("invalid password")
)
