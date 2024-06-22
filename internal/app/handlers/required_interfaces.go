package handlers

import "context"

//go:generate mockgen -destination=mocks/mock_interfaces.go yandex_gophermart/internal/app/handlers StorageInt,JWTHelperInt

// todo: ? лучше объявлять разные интерфейсы (userStorage, ordersStorage и т.д.) или один большой StorageInt?

type StorageInt interface {
	SaveUser(login string, password string, ctx context.Context) (int, error)  //int - id
	CheckUser(login string, password string, ctx context.Context) (int, error) //int - id
}

type JWTHelperInt interface {
	BuildNewJWTString(userID int) (string, error)
	GetUserID(token string) (int, error)
}
