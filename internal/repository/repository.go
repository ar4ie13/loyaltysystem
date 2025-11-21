package repository

import (
	"context"
	"log"

	"github.com/ar4ie13/loyaltysystem/internal/repository/db/postgresql"
	"github.com/ar4ie13/loyaltysystem/internal/repository/db/postgresql/config"
	"github.com/rs/zerolog"
)

func NewRepository(ctx context.Context, conf config.PGConf, zlog zerolog.Logger) (*postgresql.DB, error) {
	repo, err := postgresql.NewDB(ctx, conf, zlog)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	zlog.Info().Msg("using PostgreSQL repository")
	zlog.Info().Msg("applying migrations")
	err = postgresql.ApplyMigrations(conf, zlog)
	if err != nil {
		return nil, err
	}
	return repo, nil
}
