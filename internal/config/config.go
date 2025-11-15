package config

import (
	"flag"
	"os"
	"strconv"
	"strings"

	serverconf "github.com/ar4ie13/loyaltysystem/internal/handlers/config"
	logconf "github.com/ar4ie13/loyaltysystem/internal/logger/config"
	pgconf "github.com/ar4ie13/loyaltysystem/internal/repository/postgresql/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	ServerConf  serverconf.ServerConf
	PGConf      pgconf.PGConf
	AccrualAddr string
	LogConf     logconf.LogLevel
}

func NewConfig() *Config {
	c := &Config{}
	c.GetConfig()
	return c
}

func (c *Config) GetConfig() {
	defaultServerAddr := "localhost:8080"
	defaultDatabaseDSN := ""
	defaultAccrualAddr := ""
	defaultLogLevel := zerolog.InfoLevel
	flag.StringVar(&c.ServerConf.ServerAddr, "a", defaultServerAddr, "server startup address (host:port)")
	flag.StringVar(&c.PGConf.DatabaseDSN, "d", defaultDatabaseDSN, "database connection string")
	flag.StringVar(&c.AccrualAddr, "r", defaultAccrualAddr, "accrual server address")
	flag.Var(&c.LogConf, "l", "log level (debug, info, warn, error, fatal)")

	if err := c.LogConf.Set(defaultLogLevel.String()); err != nil {
		log.Fatal().Err(err).Msg("Failed to set default log level")
	}

	flag.Parse()

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
		c.AccrualAddr = accrualAddr
	}

	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		err := c.LogConf.Set(logLevelStr)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to set log level from LOG_LEVEL")
		}
	}
}
