package handlers

import (
	"context"
	"go.uber.org/zap"
)

type Handler struct {
	Logger  zap.SugaredLogger
	Storage Storage
	JWTH    JWTHelper
}

// todo: ? лучше объявлять разные интерфейсы (userStorage, ordersStorage и т.д.) или один большой Storage?
type Storage interface {
	saveUser(login string, password string, ctx context.Context) (int, error) //int - id
}

type JWTHelper interface {
	BuildNewJWTString(userID int) (string, error)
}
