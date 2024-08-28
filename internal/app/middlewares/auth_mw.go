package middlewares

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"net/http"
	gophermarterrors "yandex_gophermart/pkg/errors"
	"yandex_gophermart/pkg/security"
)

type ContextKeyString string

const UserIDContextKey ContextKeyString = "userID"

func AuthMW(logger zap.SugaredLogger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/user/register":
				{
					logger.Debugf("no auth needed, serving requst: %s", r.URL.Path)
					next.ServeHTTP(w, r)
					return
				}
			case "/api/user/login":
				{
					logger.Debugf("no auth needed, serving requst: %s", r.URL.Path)
					next.ServeHTTP(w, r)
					return
				}
			default:
				{
					//Get JWT cookie
					JWTCookie, err := r.Cookie(security.JWTCookieName)
					if errors.Is(err, http.ErrNoCookie) {
						logger.Debugf("cant find any JWT in cookies, err: %v", err.Error())
						w.WriteHeader(http.StatusUnauthorized)
						return
					} else if err != nil {
						logger.Errorf("error while trying to get JWT from cookies, err: %v", err.Error())
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					//Get userID from JWT
					JWTHelper := security.NewJWTHelper()
					userID, err := JWTHelper.GetUserID(JWTCookie.Value)
					if errors.Is(err, gophermarterrors.MakeErrJWTTokenIsNotValid()) {
						w.WriteHeader(http.StatusUnauthorized)
						return
					} else if errors.Is(err, jwt.ErrTokenExpired) {
						logger.Errorf("JWT token expired, err: %v", err.Error())
						w.WriteHeader(http.StatusUnauthorized)
						return
					} else if err != nil {
						logger.Errorf("cant parse JWT token, err: %v", err.Error())
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					//Put userID in request.ctx
					logger.Debugf("user was authenticated, path - %s", r.URL.Path)
					ctxWithUserID := context.WithValue(r.Context(), UserIDContextKey, userID)
					next.ServeHTTP(w, r.WithContext(ctxWithUserID))
				}
			}
		})
	}
}
