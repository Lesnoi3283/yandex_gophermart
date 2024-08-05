package databases

import (
	"context"
	"database/sql"
	"errors"
	_ "github.com/jackc/pgx/v5"
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
			order_number VARCHAR(255),
			status VARCHAR(255),
			accural FLOAT,
			uploaded_at TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS balances (
			id SERIAL PRIMARY KEY,
			user_id INTEGER,
			points FLOAT
		);`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			id SERIAL PRIMARY KEY,
			order_id INTEGER,
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

// todo: лучше объединять ошибки с помощью errors.Join() или fmt.Errorf("an error happend during..., err: %v), err.Error()) ?
func (p *Postgresql) SaveUser(login string, passwordHash string, passwordSalt string, ctx context.Context) (int, error) {
	var userID int

	err := p.store.QueryRowContext(ctx, `
		INSERT INTO users (login, password_hash, password_salt)
		VALUES ($1, $2, $3)
		ON CONFLICT (login) DO NOTHING
		RETURNING id;`,
		login, passwordHash, passwordSalt).Scan(&userID)

	if err != nil {
		return 0, err
	}

	return userID, nil
}

// CheckUser finds an id, password and password_salt by login, then checks password (using "security" package).
// Returns an ID if password is correct, "0" + "error" if not
func (p *Postgresql) GetUserIDWithCheck(login string, password string, ctx context.Context) (int, error) {
	var userID int
	var passwordHash, passwordSalt string

	err := p.store.QueryRowContext(ctx, `
		SELECT id, password_hash, password_salt 
		FROM users 
		WHERE login = $1`, login).Scan(&userID, &passwordHash, &passwordSalt)
	if err != nil {
		return 0, err
	}

	if !security.CheckPassword(password, passwordHash, passwordSalt) {
		return 0, gophermart_errors.MakeErrWrongLoginOrPassword()
	}

	return userID, nil
}

func (p *Postgresql) SaveNewOrder(orderData entities.OrderData, ctx context.Context) error {
	_, err := p.store.ExecContext(ctx, `
		INSERT INTO orders (user_id, order_number, status, accural, uploaded_at)
		VALUES ($1, $2, $3, $4, $5)`,
		orderData.UserID, orderData.Number, orderData.Status, orderData.Accural, orderData.UploadedAt)
	return err
}

func (p *Postgresql) UpdateOrder(orderData entities.OrderData, ctx context.Context) error {
	_, err := p.store.ExecContext(ctx, `
		UPDATE orders 
		SET status = $1, accural = $2, uploaded_at = $3
		WHERE id = $4 AND user_id = $5`,
		orderData.Status, orderData.Accural, orderData.UploadedAt, orderData.ID, orderData.UserID)
	return err
}

func (p *Postgresql) GetOrdersList(userID int, ctx context.Context) ([]entities.OrderData, error) {
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
		if err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accural, &order.UploadedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (p *Postgresql) GetBalance(userID int, ctx context.Context) (entities.BalanceData, error) {
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
		WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1)`, userID).Scan(&balance.Withdrawn)
	if err != nil {
		return balance, err
	}

	return balance, nil
}

func (p *Postgresql) AddToBalance(userID int, amount float64, ctx context.Context) error {
	_, err := p.store.ExecContext(ctx, `
		INSERT INTO balances (user_id, points) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id) 
		DO UPDATE SET points = balances.points + $2`,
		userID, amount)
	return err
}

func (p *Postgresql) WithdrawFromBalance(userID int, orderID int, amount float64, ctx context.Context) error {
	tx, err := p.store.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Check balance
	var currentBalance float64
	err = tx.QueryRowContext(ctx, `
		SELECT points 
		FROM balances 
		WHERE user_id = $1`, userID).Scan(&currentBalance)
	if err != nil {
		tx.Rollback()
		return err
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
		return err
	}

	// Add new withdrawal
	_, err = tx.ExecContext(ctx, `
		INSERT INTO withdrawals (order_id, amount, processed_at) 
		VALUES ($1, $2, now())`,
		orderID, amount)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (p *Postgresql) GetWithdrawals(userID int, ctx context.Context) ([]entities.WithdrawalData, error) {
	rows, err := p.store.QueryContext(ctx, `
		SELECT w.order_id, w.amount, w.processed_at 
		FROM withdrawals w
		JOIN orders o ON w.order_id = o.id
		WHERE o.user_id = $1`, userID)
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
