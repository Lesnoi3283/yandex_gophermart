package handlers

import (
	"context"
	"yandex_gophermart/pkg/entities"
)

//go:generate mockgen -destination=mocks/mock_interfaces.go yandex_gophermart/internal/app/handlers StorageInt,JWTHelperInt

type StorageInt interface {
	SaveUser(login string, passwordHash string, passwordSalt string, ctx context.Context) (int, error) //int - ID
	GetUserIDWithCheck(login string, passwordHash string, ctx context.Context) (int, error)            //int - ID
	SaveNewOrder(orderData entities.OrderData, ctx context.Context) error
	UpdateOrder(orderData entities.OrderData, ctx context.Context) error
	GetOrdersList(userID int, ctx context.Context) ([]entities.OrderData, error)
	GetBalance(userID int, ctx context.Context) (entities.BalanceData, error)
	AddToBalance(userID int, amount float64, ctx context.Context) error
	WithdrawFromBalance(userID int, orderNum string, amount float64, ctx context.Context) error
	GetWithdrawals(userID int, ctx context.Context) (withdrawals []entities.WithdrawalData, err error)
}

type JWTHelperInt interface {
	BuildNewJWTString(userID int) (string, error)
	GetUserID(token string) (int, error)
}
