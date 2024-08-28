package handlers

import (
	"context"
	"yandex_gophermart/pkg/entities"
)

//go:generate mockgen -destination=mocks/mock_interfaces.go yandex_gophermart/internal/app/handlers StorageInt,JWTHelperInt

type StorageInt interface {
	SaveUser(ctx context.Context, login string, passwordHash string, passwordSalt string) (int, error) //int - ID
	GetUserIDWithCheck(ctx context.Context, login string, passwordHash string) (int, error)            //int - ID
	SaveNewOrder(ctx context.Context, orderData entities.OrderData) error
	UpdateOrder(ctx context.Context, orderData entities.OrderData) error
	GetOrdersList(ctx context.Context, userID int) ([]entities.OrderData, error)
	GetBalance(ctx context.Context, userID int) (entities.BalanceData, error)
	//AddToBalance(ctx context.Context, userID int, amount float64) error
	WithdrawFromBalance(ctx context.Context, userID int, orderNum string, amount float64) error
	GetWithdrawals(ctx context.Context, userID int) (withdrawals []entities.WithdrawalData, err error)
}

type JWTHelperInt interface {
	BuildNewJWTString(userID int) (string, error)
	GetUserID(token string) (int, error)
}
