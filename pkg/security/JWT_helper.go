package security

import (
	"github.com/golang-jwt/jwt/v4"
	"time"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

const JWTCookieName = "auth_JWT_token"
const secretKey = "SuperSecretKeyPart"

type JWTHelper struct {
}

func NewJWTHelper() *JWTHelper {
	return &JWTHelper{}
}

type claims struct {
	UserID int
	jwt.Claims
}

// todo: улучшить секурность, почитать про разные методы подписи
func (j *JWTHelper) BuildNewJWTString(userID int) (string, error) {
	claims := claims{
		UserID: userID,
		Claims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72))},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	stringToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return stringToken, nil
}

// todo: улучшить секурность, почитать про работу функции возврата секрета
func (j *JWTHelper) GetUserID(token string) (int, error) {
	claims := claims{}

	tokenGot, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return -1, err
	}
	if !tokenGot.Valid {
		return -1, gophermart_errors.MakeJWTTokenIsNotValid()
	}

	return claims.UserID, nil
}
