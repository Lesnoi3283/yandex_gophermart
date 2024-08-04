package handlers

import (
	"context"
	"yandex_gophermart/pkg/entities"
)

//go:generate mockgen -destination=mocks/mock_interfaces.go yandex_gophermart/internal/app/handlers StorageInt,JWTHelperInt

// todo: ? лучше объявлять разные интерфейсы (userStorage, ordersStorage и т.д.) или один большой StorageInt?

type StorageInt interface {
	SaveUser(login string, password string, ctx context.Context) (int, error)  //int - ID
	CheckUser(login string, password string, ctx context.Context) (int, error) //int - ID
	SaveNewOrder(orderData entities.OrderData, ctx context.Context) error
	UpdateOrder(orderData entities.OrderData, ctx context.Context) error
	GetOrdersList(userID int, ctx context.Context) ([]entities.OrderData, error)
	GetBalance(userID int, ctx context.Context) (entities.BalanceData, error)
	AddRoBalance(userID int, amount float64, ctx context.Context) error
	WithdrawFromBalance(userID int, orderID int, amount float64, ctx context.Context) error
}

type JWTHelperInt interface {
	BuildNewJWTString(userID int) (string, error)
	GetUserID(token string) (int, error)
}
