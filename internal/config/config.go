package config

import (
	"flag"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	authconf "github.com/ar4ie13/loyaltysystem/internal/auth/config"
	serverconf "github.com/ar4ie13/loyaltysystem/internal/handlers/config"
	logconf "github.com/ar4ie13/loyaltysystem/internal/logger/config"
	pgconf "github.com/ar4ie13/loyaltysystem/internal/repository/db/postgresql/config"
	reqconf "github.com/ar4ie13/loyaltysystem/internal/requestor/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config is a main configuration object
type Config struct {
	AuthConf    authconf.Config
	ServerConf  serverconf.ServerConf
	PGConf      pgconf.PGConf
	AccrualConf reqconf.ReqConf
	LogConf     logconf.LogLevel
}

// NewConfig creates new Config configuration object
func NewConfig() *Config {
	c := &Config{
		AuthConf: authconf.Config{
			SecretKey:       "nHhjHgahbioHBGbBHJ",
			TokenExpiration: 24 * time.Hour,
			PasswordLen:     6,
		},
		ServerConf: serverconf.ServerConf{
			ServerAddr: "localhost:8080",
		},
		AccrualConf: reqconf.ReqConf{
			WorkerNum:   runtime.NumCPU(),
			AccrualAddr: "http://localhost:8081",
		},
		LogConf: logconf.LogLevel{
			Level: zerolog.DebugLevel,
		},
	}

	c.BindFlags()
	flag.Parse()
	c.LoadEnv()

	return c
}

// BindFlags parses flags and environment variables for service configuration
func (c *Config) BindFlags() {
	flag.StringVar(&c.ServerConf.ServerAddr, "a", c.ServerConf.ServerAddr, "server startup address (host:port)")
	flag.StringVar(&c.PGConf.DatabaseDSN, "d", c.PGConf.DatabaseDSN, "database connection string")
	flag.StringVar(&c.AccrualConf.AccrualAddr, "r", c.AccrualConf.AccrualAddr, "accrual server address")
	flag.Var(&c.LogConf, "l", "log level (debug, info, warn, error, fatal)")
	flag.StringVar(&c.AuthConf.SecretKey, "k", c.AuthConf.SecretKey, "secret key for authorization")
	flag.DurationVar(&c.AuthConf.TokenExpiration, "e", c.AuthConf.TokenExpiration, "token expiration")
	flag.IntVar(&c.AuthConf.PasswordLen, "p", c.AuthConf.PasswordLen, "password minimal length")
}

func (c *Config) LoadEnv() {
	if serverAddr := os.Getenv("RUN_ADDRESS"); serverAddr != "" {
		if _, err := strconv.Unquote("\"" + serverAddr + "\""); err != nil {
			parts := strings.SplitN(serverAddr, ":", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				log.Fatal().Err(err).Msg("Failed to set server address from RUN_ADDRESS")
			}
		}
		c.ServerConf.ServerAddr = serverAddr
	}

	if databaseDSN := os.Getenv("DATABASE_URI"); databaseDSN != "" {
		c.PGConf.DatabaseDSN = databaseDSN
	}

	if accrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); accrualAddr != "" {
		if _, err := strconv.Unquote("\"" + accrualAddr + "\""); err != nil {
			parts := strings.SplitN(accrualAddr, ":", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				log.Fatal().Err(err).Msg("Failed to set accrual system address from ACCRUAL_SYSTEM_ADDRESS")
			}
		}
		c.AccrualConf.AccrualAddr = accrualAddr
	}

	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		err := c.LogConf.Set(logLevelStr)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to set log level from LOG_LEVEL")
		}
	}

	if secretKey := os.Getenv("SECRET_KEY"); secretKey != "" {
		c.AuthConf.SecretKey = secretKey
	}

	if tokenExpirationStr := os.Getenv("TOKEN_EXPIRATION"); tokenExpirationStr != "" {
		var err error
		c.AuthConf.TokenExpiration, err = time.ParseDuration(tokenExpirationStr)
		if err != nil {
			log.Fatal().Err(err).Msg("cannot parse token expiration environment variable")
		}

	}

	if passwordLen, err := strconv.Atoi(os.Getenv("PASSWORD_MIN_LENGTH")); passwordLen != 0 {
		if err != nil {
			log.Fatal().Err(err).Msg("cannot parse password length environment variable")
		}
		c.AuthConf.PasswordLen = passwordLen
	}
}
