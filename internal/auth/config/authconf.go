package config

import "time"

// Config object for authentication service
type Config struct {
	SecretKey       string
	TokenExpiration time.Duration
	PasswordLen     int
}
