package handlers

import (
	"go.uber.org/zap"
)

type Handler struct {
	Logger  zap.SugaredLogger
	Storage StorageInt
	JWTH    JWTHelperInt
}
