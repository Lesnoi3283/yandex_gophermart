package handlers

import (
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"yandex_gophermart/internal/app/middlewares"
	"yandex_gophermart/pkg/security"
)

func NewRouter(logger zap.SugaredLogger, storage StorageInt, accrualSystemAddress string) chi.Router {
	//configure
	r := chi.NewRouter()
	handler := Handler{
		Logger:               logger,
		Storage:              storage,
		JWTH:                 security.NewJWTHelper(),
		AccrualSystemAddress: accrualSystemAddress,
	}

	//middlewares
	r.Use(middlewares.AuthMW(logger))
	r.Use(middlewares.LoggerMW(logger))

	//handlers
	r.Post("/api/user/register", handler.RegisterUser)
	r.Post("/api/user/login", handler.AuthUser)
	r.Post("/api/user/orders", handler.OrderUploadHandler)
	r.Get("/api/user/orders", handler.OrdersListHandler)
	r.Get("/api/user/balance", handler.GetBalanceHandler)
	r.Post("/api/user/balance/withdraw", handler.WithdrawHandler)
	r.Get("/api/user/withdrawals", handler.GetWithdrawals)

	return r
}
