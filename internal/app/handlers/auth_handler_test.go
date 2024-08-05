package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	mock_handlers "yandex_gophermart/internal/app/handlers/mocks"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
	"yandex_gophermart/pkg/security"
)

func TestHandler_AuthUser(t *testing.T) {

	//data set
	testUser := entities.UserData{
		ID:       1,
		Login:    "login",
		Password: "123",
	}

	correctJWTString := "someTestJWT"

	//logger set
	logger := zaptest.NewLogger(t)
	sugarLogger := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

	//tests set
	type fields struct {
		Logger  zap.SugaredLogger
		Storage StorageInt
		JWTH    JWTHelperInt
	}
	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		statusWant int
		checkJWT   bool
	}{
		// TODO: Add test cases.
		{
			name: "normal",
			fields: struct {
				Logger  zap.SugaredLogger
				Storage StorageInt
				JWTH    JWTHelperInt
			}{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().GetUserIDWithCheck(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(login string, password string, ctx context.Context) (int, error) {
						if (login == testUser.Login) && (password == testUser.Password) {
							return testUser.ID, nil
						} else {
							return -1, gophermart_errors.MakeErrUserNotFound()
						}
					})
					return storage
				}(),
				JWTH: func() JWTHelperInt {
					JWTH := mock_handlers.NewMockJWTHelperInt(controller)
					JWTH.EXPECT().BuildNewJWTString(testUser.ID).Return(correctJWTString, nil)
					return JWTH
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/login", func() io.Reader {
					data := struct {
						Login    string `json:"login"`
						Password string `json:"password"`
					}{
						Login:    testUser.Login,
						Password: testUser.Password,
					}
					jsonData, err := json.Marshal(data)
					if err != nil {
						logger.Error("auth handler test, err while building json request", zap.Error(err))
					}
					return bytes.NewReader(jsonData)
				}()),
			},
			statusWant: http.StatusOK,
			checkJWT:   true,
		},
		{
			name: "wrong password",
			fields: struct {
				Logger  zap.SugaredLogger
				Storage StorageInt
				JWTH    JWTHelperInt
			}{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().GetUserIDWithCheck(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(login string, password string, ctx context.Context) (int, error) {
						return 0, gophermart_errors.MakeErrUserNotFound()
					})
					return storage
				}(),
				JWTH: func() JWTHelperInt {
					JWTH := mock_handlers.NewMockJWTHelperInt(controller)
					return JWTH
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/login", func() io.Reader {
					data := struct {
						Login    string `json:"login"`
						Password string `json:"password"`
					}{
						Login:    testUser.Login,
						Password: "wrong password",
					}
					json, err := json.Marshal(data)
					if err != nil {
						logger.Error("auth handler test", zap.Error(err))
					}
					return bytes.NewReader(json)
				}()),
			},
			statusWant: http.StatusUnauthorized,
			checkJWT:   false,
		},
		{
			name: "bad request",
			fields: struct {
				Logger  zap.SugaredLogger
				Storage StorageInt
				JWTH    JWTHelperInt
			}{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					return storage
				}(),
				JWTH: func() JWTHelperInt {
					JWTH := mock_handlers.NewMockJWTHelperInt(controller)
					return JWTH
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/login", func() io.Reader {
					return bytes.NewReader([]byte("bad request"))
				}()),
			},
			statusWant: http.StatusBadRequest,
			checkJWT:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Logger:  tt.fields.Logger,
				Storage: tt.fields.Storage,
				JWTH:    tt.fields.JWTH,
			}
			h.AuthUser(tt.args.w, tt.args.r)

			assert.Equal(t, tt.statusWant, tt.args.w.Code, "HTTP status is wrong")

			if tt.checkJWT {
				wasJWTFound := false
				cookies := tt.args.w.Result().Cookies()
				tt.args.w.Result().Body.Close()
				//todo: vet check ругается на незакрытое тело ответа
				for _, cookie := range cookies {
					if cookie.Name == security.JWTCookieName {
						wasJWTFound = true
						assert.Equal(t, correctJWTString, cookie.Value)
					}
				}
				assert.Equal(t, true, wasJWTFound, "JWT cookie wasn`t found")
			}
		})
	}
}
