package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	"github.com/ar4ie13/loyaltysystem/internal/models"
	"github.com/ar4ie13/loyaltysystem/internal/repository/db/postgresql/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// DB is a main postgres repository object
type DB struct {
	pool *pgxpool.Pool
	zlog zerolog.Logger
}

// NewDB construct postgres DB object
func NewDB(ctx context.Context, cfg config.PGConf, zlog zerolog.Logger) (*DB, error) {
	pool, err := initPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}
	return &DB{
		pool: pool,
		zlog: zlog,
	}, nil
}

// initPool initializes pgx connection pool
func initPool(ctx context.Context, cfg config.PGConf) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the DSN: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping the DB: %w", err)
	}
	return pool, nil
}

// Close closes pgx pool
func (db *DB) Close() error {
	db.pool.Close()
	return nil
}

func (db *DB) CreateUser(ctx context.Context, user models.User) error {
	const query = `
		INSERT INTO users (uuid, login, password_hash) 
		VALUES ($1, $2, $3) ON CONFLICT (login) DO NOTHING`

	tag, err := db.pool.Exec(ctx, query, user.UUID, user.Login, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return apperrors.ErrUserAlreadyExists
	}

	return nil
}

func (db *DB) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	const query = `SELECT uuid, login, password_hash, created_at, updated_at from users where login=$1`

	var user models.User

	row := db.pool.QueryRow(ctx, query, login)

	err := row.Scan(&user.UUID, &user.Login, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.User{}, apperrors.ErrUserNotFound
		default:
			return models.User{}, fmt.Errorf("failed to scan a response row: %w", err)
		}
	}

	return user, nil
}
