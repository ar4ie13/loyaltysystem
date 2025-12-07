package postgresql

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/ar4ie13/loyaltysystem/internal/repository/db/postgresql/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrationsDir embed.FS

// ApplyMigrations applies all required migrations to the latest version
func ApplyMigrations(pgcfg config.PGConf, zlog zerolog.Logger) error {
	sourceDriver, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return iofs driver: %w", err)
	}

	zlog.Debug().Msgf("connecting to postgresql_url=%s", pgcfg.DatabaseDSN)
	dbConn, err := sql.Open("pgx", pgcfg.DatabaseDSN)
	if err != nil {
		zlog.Fatal().Err(err).Msg("while connecting to postgresql")
	}
	defer func() {
		if err = dbConn.Close(); err != nil {
			zlog.Fatal().Err(err).Msg("while closing postgresql")
		}
	}()

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, pgcfg.DatabaseDSN)
	if err != nil {
		zlog.Fatal().Err(err).Msg("failed to create golang-migrate instance")
	}

	if err = m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			zlog.Fatal().Err(err).Msg("migration up failed")
		}
		zlog.Info().Msg("no data to migrate")
		return nil
	}
	zlog.Info().Msg("migration up applied successfully")

	return nil
}
