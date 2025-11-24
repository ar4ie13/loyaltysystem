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
	c := &Config{}
	c.GetConfig()

	return c
}

// GetConfig parses flags and environment variables for service configuration
func (c *Config) GetConfig() {

	defaultServerAddr := "localhost:8080"
	defaultDatabaseDSN := ""
	defaultAccrualAddr := "localhost:8081"
	defaultLogLevel := zerolog.DebugLevel
	defaultSecretKey := "nHhjHgahbioHBGbBHJ"
	defaultTokenExpiration := time.Hour * 24
	defaultPasswordLength := 6

	flag.StringVar(&c.ServerConf.ServerAddr, "a", defaultServerAddr, "server startup address (host:port)")
	flag.StringVar(&c.PGConf.DatabaseDSN, "d", defaultDatabaseDSN, "database connection string")
	flag.StringVar(&c.AccrualConf.AccrualAddr, "r", defaultAccrualAddr, "accrual server address")
	flag.Var(&c.LogConf, "l", "log level (debug, info, warn, error, fatal)")
	flag.StringVar(&c.AuthConf.SecretKey, "k", defaultSecretKey, "secret key for authorization")
	flag.DurationVar(&c.AuthConf.TokenExpiration, "e", defaultTokenExpiration, "token expiration")
	flag.IntVar(&c.AuthConf.PasswordLen, "p", defaultPasswordLength, "password minimal length")

	if err := c.LogConf.Set(defaultLogLevel.String()); err != nil {
		log.Fatal().Err(err).Msg("Failed to set default log level")
	}

	flag.Parse()

	c.AccrualConf.WorkerNum = runtime.NumCPU()

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

	if passwordLen, err := strconv.Atoi(os.Getenv("SECRET_KEY")); passwordLen != 0 {
		if err != nil {
			log.Fatal().Err(err).Msg("cannot parse password length environment variable")
		}
		c.AuthConf.PasswordLen = passwordLen
	}
}
