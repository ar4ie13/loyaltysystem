package postgresql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	"github.com/ar4ie13/loyaltysystem/internal/models"
	"github.com/ar4ie13/loyaltysystem/internal/repository/db/postgresql/config"
	"github.com/google/uuid"
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

func (db *DB) PutUserOrder(ctx context.Context, userUUID uuid.UUID, order string) error {
	const (
		queryInsert = `
		INSERT INTO orders (order_num, status, user_uuid) 
		VALUES ($1, $2, $3)`

		querySelect = `
		SELECT user_uuid FROM ORDERS WHERE order_num = $1`
	)

	var checkUserUUID uuid.UUID

	row := db.pool.QueryRow(ctx, querySelect, order)

	err := row.Scan(&checkUserUUID)
	if err != nil {
		switch {
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
	}

	if checkUserUUID != uuid.Nil {
		switch {
		case checkUserUUID != userUUID:
			return apperrors.ErrOrderNumberAlreadyUsed
		case checkUserUUID == userUUID:
			return apperrors.ErrOrderAlreadyExists
		}
	}

	tag, err := db.pool.Exec(ctx, queryInsert, order, "NEW", userUUID)
	if err != nil {
		return fmt.Errorf("failed to insert userUUID: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return apperrors.ErrOrderAlreadyExists
	}

	return nil
}

func (db *DB) GetUserOrders(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error) {
	const queryStmt = `SELECT order_num, status, accrual, user_uuid, created_at FROM orders 
                    	WHERE user_uuid = $1 ORDER BY created_at DESC`

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		db.zlog.Debug().Msgf("request execution duration: %s", elapsed)
	}()

	rows, err := db.pool.Query(ctx, queryStmt, userUUID)
	if err != nil {
		return nil, err
	}

	var orders []models.Order

	for rows.Next() {
		var order models.Order

		err = rows.Scan(&order.OrderNumber, &order.Status, &order.Accrual, &order.UserUUID, &order.CreatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, apperrors.ErrNoOrders
	}

	return orders, nil
}

func (db *DB) GetUnprocessedOrders(ctx context.Context, limit int) ([]string, error) {
	const query = `SELECT order_num FROM orders WHERE status IN ('NEW', 'PROCESSING') ORDER BY created_at ASC LIMIT $1`

	var orderNums []string

	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var order string

		err = rows.Scan(&order)
		if err != nil {
			return nil, err
		}
		orderNums = append(orderNums, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return orderNums, nil
}

func (db *DB) UpdateOrder(ctx context.Context, orderNum string, status string, accrual float64) error {
	var query string
	if accrual == 0 {
		query = `UPDATE orders  SET status = $1 WHERE order_num = $2`
		tag, err := db.pool.Exec(ctx, query, status, orderNum)
		if err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		rowsInserted := tag.RowsAffected()

		if rowsInserted == 0 {
			return fmt.Errorf("no rows were updated")
		}

	} else {
		query = `UPDATE  orders  SET accrual = $1, status = $2 WHERE order_num = $3`
		tag, err := db.pool.Exec(ctx, query, accrual, status, orderNum)
		if err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		rowsInserted := tag.RowsAffected()

		if rowsInserted == 0 {
			return fmt.Errorf("no rows were updated")
		}
	}

	return nil
}
