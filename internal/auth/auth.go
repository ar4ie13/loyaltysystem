package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	authconf "github.com/ar4ie13/loyaltysystem/internal/auth/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	Claims Claims
	conf   authconf.Config
}

type Claims struct {
	jwt.RegisteredClaims
	UserUUID uuid.UUID
}

// NewAuth creates Auth object
func NewAuth(conf authconf.Config) *Auth {
	return &Auth{
		conf: conf,
	}
}

// GenerateUserUUID generates new UUID for user
func (a Auth) GenerateUserUUID() uuid.UUID {
	return uuid.New()
}

// BuildJWTString creates new JWT token
func (a Auth) BuildJWTString(userUUID uuid.UUID) (string, error) {
	// creating new token with HS256 algorithm and claims — Auth
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// token expiration date
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.conf.TokenExpiration)),
		},
		// personal claim
		UserUUID: userUUID,
	})

	// creating signed token string
	tokenString, err := token.SignedString([]byte(a.conf.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateUserUUID validates token and return the UUID of user
func (a Auth) ValidateUserUUID(tokenString string) (uuid.UUID, error) {
	claims, token, err := a.parseTokenString(tokenString)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, apperrors.ErrUserIsNotAuthorized
		} else {
			return uuid.Nil, err
		}
	}
	if claims.UserUUID.String() == "" || claims.UserUUID == uuid.Nil {
		return uuid.Nil, apperrors.ErrInvalidUserUUID
	}

	if !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}

	return claims.UserUUID, nil
}

// parseTokenString parses token string and returns claims and token (for validation)
func (a Auth) parseTokenString(tokenString string) (*Claims, *jwt.Token, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.conf.SecretKey), nil
	})
	if err != nil {
		return claims, token, err
	}
	return claims, token, nil
}

func (a Auth) GenerateHashFromPassword(password string) (string, error) {
	if len(password) < a.conf.PasswordLen {
		return "", fmt.Errorf("%w: should be %d", apperrors.ErrPasswordMinSymbols, a.conf.PasswordLen)
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func (a Auth) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}
