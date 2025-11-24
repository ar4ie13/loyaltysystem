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

// CreateUser stores user information to the db
func (db *DB) CreateUser(ctx context.Context, user models.User) error {
	const query = `
		INSERT INTO users (uuid, login, password_hash, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (login) DO NOTHING`

	tag, err := db.pool.Exec(ctx, query, user.UUID, user.Login, user.PasswordHash, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return apperrors.ErrUserAlreadyExists
	}

	return nil
}

// GetUserByLogin retrieves user information from db
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

// PutUserOrder stores user's order without withdrawn to the db
func (db *DB) PutUserOrder(ctx context.Context, userUUID uuid.UUID, order string) error {
	const (
		queryInsert = `
		INSERT INTO orders (order_num, status, user_uuid, created_at)
		VALUES ($1, $2, $3, $4)`

		querySelect = `
		SELECT user_uuid FROM ORDERS WHERE order_num = $1`
	)

	var checkUserUUID uuid.UUID
	// Begin transaction
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to start a transaction: %w", err)
	}

	row := db.pool.QueryRow(ctx, querySelect, order)

	err = row.Scan(&checkUserUUID)
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

	tag, err := db.pool.Exec(ctx, queryInsert, order, "NEW", userUUID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert userUUID: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if rowsInserted == 0 {
		return apperrors.ErrOrderAlreadyExists
	}

	return nil
}

// GetUserOrders retrieves all user's orders from db
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

// GetUnprocessedOrders retrieves orders withoud final status from db, used by requestor service
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

// UpdateOrderWithoutAccrual updates status for orders without accrual, used by requestor service
func (db *DB) UpdateOrderWithoutAccrual(ctx context.Context, orderNum string, status string) error {

	queryUpdOrders := `UPDATE orders  SET status = $1 WHERE order_num = $2`
	tag, err := db.pool.Exec(ctx, queryUpdOrders, status, orderNum)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("no rows were updated")
	}

	return nil
}

// UpdateOrderWithAccrual updates status for orders with accrual, used by requestor service
func (db *DB) UpdateOrderWithAccrual(ctx context.Context, orderNum string, status string, accrual float64) error {
	// Begin transaction
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to start a transaction: %w", err)
	}

	queryUpdOrders := `UPDATE  orders  SET accrual = $1, status = $2 WHERE order_num = $3`
	tag, err := db.pool.Exec(ctx, queryUpdOrders, accrual, status, orderNum)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("no rows were updated during order update")
	}

	queryUpdUsers := `UPDATE  users  SET balance = balance + $1, updated_at = $2  WHERE uuid = (SELECT user_uuid from orders where order_num = $3)`
	tag, err = db.pool.Exec(ctx, queryUpdUsers, accrual, time.Now(), orderNum)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	rowsInserted = tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("no rows were updated during user balance update")
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetBalance retrieves user's balance from db
func (db *DB) GetBalance(ctx context.Context, user uuid.UUID) (models.User, error) {
	const queryStmt = `SELECT balance, withdrawn FROM users 
                    	WHERE uuid = $1`
	var balance models.User

	row := db.pool.QueryRow(ctx, queryStmt, user)
	err := row.Scan(&balance.Balance, &balance.Withdrawn)
	if err != nil {
		db.zlog.Error().Msgf("failed to query user balance: %v", err)
		return balance, err
	}
	return balance, nil
}

// PutUserWithdrawnOrder stores user's order with withdrawn to the db
func (db *DB) PutUserWithdrawnOrder(ctx context.Context, user uuid.UUID, orderNum string, withdrawn float64) error {
	const (
		querySelect = `SELECT balance FROM users WHERE uuid = $1 FOR UPDATE`
		queryInsert = `INSERT INTO orders (order_num, status, user_uuid, withdrawn, created_at) 
						VALUES ($1, $2, $3, $4, $5)`
		queryUpdate = `UPDATE users  SET withdrawn = withdrawn + $1, balance = balance - $1, updated_at = $2
              WHERE uuid = $3`
	)

	// Begin transaction
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Checking user balance
	var balance models.User
	row := db.pool.QueryRow(ctx, querySelect, user)
	err = row.Scan(&balance.Balance)
	if err != nil {
		db.zlog.Error().Msgf("failed to query user balance: %v", err)
		return err
	}

	if balance.Balance < withdrawn {
		return apperrors.ErrBalanceNotEnough
	}

	// Inserting order
	tag, err := db.pool.Exec(ctx, queryInsert, orderNum, "PROCESSED", user, withdrawn, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert order balance: %w", err)
	}

	rowsInserted := tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("no rows were updated during order insert")
	}

	// Updating user balance
	tag, err = db.pool.Exec(ctx, queryUpdate, withdrawn, time.Now(), user)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	rowsInserted = tag.RowsAffected()

	if rowsInserted == 0 {
		return fmt.Errorf("no rows were updated during user balance update")
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserWithdrawals retrieves all users withdrawals from the db
func (db *DB) GetUserWithdrawals(ctx context.Context, userUUID uuid.UUID) ([]models.Order, error) {
	const queryStmt = `SELECT order_num, withdrawn, created_at FROM orders 
                    	WHERE user_uuid = $1 AND withdrawn IS NOT NULL ORDER BY created_at DESC`

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

		err = rows.Scan(&order.OrderNumber, &order.Withdrawn, &order.CreatedAt)
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
