package config

import "time"

type Config struct {
	SecretKey       string
	TokenExpiration time.Duration
	PasswordLen     int
}
