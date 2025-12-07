package main

import (
	"context"
	"log"

	"github.com/ar4ie13/loyaltysystem/internal/auth"
	"github.com/ar4ie13/loyaltysystem/internal/config"
	"github.com/ar4ie13/loyaltysystem/internal/handlers"
	"github.com/ar4ie13/loyaltysystem/internal/logger"
	"github.com/ar4ie13/loyaltysystem/internal/repository"
	"github.com/ar4ie13/loyaltysystem/internal/requestor"
	"github.com/ar4ie13/loyaltysystem/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.NewConfig()
	zlog := logger.NewLogger(cfg.LogConf.Level)
	authorize := auth.NewAuth(cfg.AuthConf)
	repo, err := repository.NewRepository(context.Background(), cfg.PGConf, zlog.Logger)
	if err != nil {
		return err
	}
	srv := service.NewService(repo, zlog.Logger)
	hndlr := handlers.NewHandlers(cfg.ServerConf, authorize, srv, zlog.Logger)
	requestor.NewRequestor(cfg.AccrualConf, zlog.Logger, repo)
	if err = hndlr.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
