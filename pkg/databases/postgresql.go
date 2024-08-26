package databases

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	time2 "time"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
	"yandex_gophermart/pkg/security"
)

type Postgresql struct {
	store *sql.DB
}

func NewPostgresql(connStr string) (*Postgresql, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, errors.Join(errors.New("cant create a new postgresql storage"), err)
	}
	return &Postgresql{
		store: db,
	}, nil
}

func (p *Postgresql) Ping() error {
	return p.store.Ping()
}

func (p *Postgresql) SetTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			login VARCHAR(255) UNIQUE,
			password_hash VARCHAR(255),
			password_salt VARCHAR(255)
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER,
			order_number VARCHAR(255) UNIQUE,
			status VARCHAR(255),
			accural FLOAT,
			uploaded_at TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS balances (
			id SERIAL PRIMARY KEY,
			user_id INTEGER UNIQUE,
			points FLOAT
		);`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			id SERIAL PRIMARY KEY,
			order_num VARCHAR(255),
			user_id INTEGER,
			amount FLOAT,
			processed_at TIMESTAMP
		);`,
	}

	for _, query := range queries {
		if _, err := p.store.Exec(query); err != nil {
			return errors.Join(errors.New("error while setting tables in postgres db"), err)
		}
	}

	return nil
}

func (p *Postgresql) SaveUser(ctx context.Context, login string, passwordHash string, passwordSalt string) (int, error) {
	var userID int

	err := p.store.QueryRowContext(ctx, `
		INSERT INTO users (login, password_hash, password_salt)
		VALUES ($1, $2, $3)
		ON CONFLICT (login) DO NOTHING
		RETURNING id;`,
		login, passwordHash, passwordSalt).Scan(&userID)

	if errors.Is(err, sql.ErrNoRows) {
		// "save user err: %v"
		return 0, gophermart_errors.MakeErrUserAlreadyExists()
	} else if err != nil {
		return 0, err
	}

	return userID, nil
}

// CheckUser finds an id, password and password_salt by login, then checks password (using "security" package).
// Returns an ID if password is correct, "0" + "error" if not
func (p *Postgresql) GetUserIDWithCheck(ctx context.Context, login string, password string) (int, error) {
	var userID int
	var passwordHash, passwordSalt string

	err := p.store.QueryRowContext(ctx, `
		SELECT id, password_hash, password_salt 
		FROM users 
		WHERE login = $1`, login).Scan(&userID, &passwordHash, &passwordSalt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, gophermart_errors.MakeErrWrongLoginOrPassword()
	} else if err != nil {
		return 0, err
	}

	if !security.CheckPassword(password, passwordHash, passwordSalt) {
		return 0, gophermart_errors.MakeErrWrongLoginOrPassword()
	}

	return userID, nil
}

func (p *Postgresql) SaveNewOrder(ctx context.Context, orderData entities.OrderData) error {
	var userID int
	time := orderData.UploadedAt.Time

	err := p.store.QueryRowContext(ctx, `
		INSERT INTO orders (user_id, order_number, status, accural, uploaded_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING user_id`,
		orderData.UserID, orderData.Number, orderData.Status, orderData.Accrual, time).Scan(&userID)

	//check who uploaded this order first (conflict)
	if err != nil {
		if pqErr, ok := err.(*pgconn.PgError); ok && pqErr.Code == "23505" {
			err = p.store.QueryRowContext(ctx, `
				SELECT user_id FROM orders WHERE order_number = $1`,
				orderData.Number).Scan(&userID)
			if err != nil {
				return err
			}
			if userID == orderData.UserID {
				return gophermart_errors.MakeErrUserHasAlreadyUploadedThisOrder()
			} else {
				return gophermart_errors.MakeErrThisOrderWasUploadedByDifferentUser()
			}
		}
		return err
	}

	return nil
}

// UpdateOrder updates an order and increases users`s balance if order status is "PROCESSED"
func (p *Postgresql) UpdateOrder(ctx context.Context, orderData entities.OrderData) error {
	tx, err := p.store.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cant begin a transaction, err: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE orders 
		SET status = $1, accural = $2, uploaded_at = $3
		WHERE id = $4 AND user_id = $5`,
		orderData.Status, orderData.Accrual, orderData.UploadedAt.Time, orderData.ID, orderData.UserID)

	if err != nil {
		return fmt.Errorf("error while updating an order: %w", err)
	}

	if orderData.Status == entities.OrderStatusProcessed {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO balances (user_id, points) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id) 
		DO UPDATE SET points = balances.points + $2`,
			orderData.UserID, orderData.Accrual)
		if err != nil {
			return fmt.Errorf("cant increase users balance, err: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error while committing transaction, %w", err)
	}

	return nil
}

func (p *Postgresql) GetOrdersList(ctx context.Context, userID int) ([]entities.OrderData, error) {
	rows, err := p.store.QueryContext(ctx, `
		SELECT id, user_id, order_number, status, accural, uploaded_at 
		FROM orders 
		WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []entities.OrderData
	for rows.Next() {
		var order entities.OrderData
		var time time2.Time
		if err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &time); err != nil {
			return nil, err
		}
		order.UploadedAt.Time = time
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (p *Postgresql) GetUnfinishedOrdersList(ctx context.Context) ([]entities.OrderData, error) {
	rows, err := p.store.QueryContext(ctx, `
		SELECT id, user_id, order_number, status, accural, uploaded_at 
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []entities.OrderData
	for rows.Next() {
		var order entities.OrderData
		err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt.Time)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (p *Postgresql) GetBalance(ctx context.Context, userID int) (entities.BalanceData, error) {
	var balance entities.BalanceData

	// Current balance
	err := p.store.QueryRowContext(ctx, `
		SELECT id, user_id, points 
		FROM balances 
		WHERE user_id = $1`, userID).Scan(&balance.ID, &balance.UserID, &balance.Current)
	if err != nil {
		return balance, err
	}

	//Sum of withdrawals
	err = p.store.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) 
		FROM withdrawals 
		WHERE user_id = $1`, userID).Scan(&balance.Withdrawn)
	if err != nil {
		return balance, err
	}

	return balance, nil
}

func (p *Postgresql) AddToBalance(ctx context.Context, userID int, amount float64) error {
	_, err := p.store.ExecContext(ctx, `
		INSERT INTO balances (user_id, points) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id) 
		DO UPDATE SET points = balances.points + $2`,
		userID, amount)
	return err
}

func (p *Postgresql) WithdrawFromBalance(ctx context.Context, userID int, orderNum string, amount float64) error {
	tx, err := p.store.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Check balance
	var currentBalance float64
	err = tx.QueryRowContext(ctx, `
		SELECT points 
		FROM balances 
		WHERE user_id = $1 FOR UPDATE`, userID).Scan(&currentBalance)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("cant get balance to check, err: %w", err)
	}

	if currentBalance < amount {
		tx.Rollback()
		return gophermart_errors.MakeErrNotEnoughPoints()
	}

	// Withdraw
	_, err = tx.ExecContext(ctx, `
		UPDATE balances 
		SET points = points - $1 
		WHERE user_id = $2`, amount, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("cant withdraw from balance in db, err: %w", err)
	}

	// Add new withdrawal
	_, err = tx.ExecContext(ctx, `
		INSERT INTO withdrawals (order_num, user_id, amount, processed_at) 
		VALUES ($1, $2, $3, now())`,
		orderNum, userID, amount)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("cant add new withdrawal, err: %w", err)
	}

	return tx.Commit()
}

func (p *Postgresql) GetWithdrawals(ctx context.Context, userID int) ([]entities.WithdrawalData, error) {
	rows, err := p.store.QueryContext(ctx, `
		SELECT order_num, amount, processed_at 
		FROM withdrawals WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []entities.WithdrawalData
	for rows.Next() {
		var withdrawal entities.WithdrawalData
		if err := rows.Scan(&withdrawal.OrderNum, &withdrawal.Sum, &withdrawal.ProcessedAt.Time); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, rows.Err()
}
