package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"net/http"
	"net/http/httptest"
	"testing"
	mock_handlers "yandex_gophermart/internal/app/handlers/mocks"
	"yandex_gophermart/pkg/entities"
	gophermarterrors "yandex_gophermart/pkg/errors"
	"yandex_gophermart/pkg/security"
)

func TestHandler_RegisterUser(t *testing.T) {
	//data set
	testUser := entities.UserData{
		ID:       1,
		Login:    "login",
		Password: "123",
	}
	testUserRequestData := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{
		Login:    testUser.Login,
		Password: testUser.Password,
	}
	JSONTestUserData, err := json.Marshal(testUserRequestData)
	require.NoError(t, err, "cant marshal test data")

	correctJWTString := "someTestJWT"

	//logger set
	logger := zaptest.NewLogger(t)
	sugarLogger := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

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
	}{
		{
			name: "normal",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveUser(testUser.Login, gomock.Any(), gomock.Any(), gomock.Any()).Return(testUser.ID, nil)
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
				r: httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(JSONTestUserData)),
			},
			statusWant: http.StatusOK,
		},
		{
			name: "User already exists",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveUser(testUser.Login, gomock.Any(), gomock.Any(), gomock.Any()).Return(0, gophermarterrors.MakeErrUserAlreadyExists())
					return storage
				}(),
				JWTH: func() JWTHelperInt {
					JWTH := mock_handlers.NewMockJWTHelperInt(controller)
					return JWTH
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(JSONTestUserData)),
			},
			statusWant: http.StatusConflict,
		},
		{
			name: "bad request",
			fields: fields{
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
				r: httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(JSONTestUserData[0:(len(JSONTestUserData)/2)])),
			},
			statusWant: http.StatusBadRequest,
		},
		{
			name: "db error (internal server error)",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveUser(testUser.Login, gomock.Any(), gomock.Any(), gomock.Any()).Return(0, errors.New("some test error"))
					return storage
				}(),
				JWTH: func() JWTHelperInt {
					JWTH := mock_handlers.NewMockJWTHelperInt(controller)
					return JWTH
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(JSONTestUserData)),
			},
			statusWant: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Logger:  tt.fields.Logger,
				Storage: tt.fields.Storage,
				JWTH:    tt.fields.JWTH,
			}

			h.RegisterUser(tt.args.w, tt.args.r)

			if assert.Equal(t, tt.statusWant, tt.args.w.Code, "wrong status code") {
				if tt.args.w.Code == http.StatusOK {
					wasJWTFound := false
					res := tt.args.w.Result()
					cookies := res.Cookies()
					for _, cookie := range cookies {
						if cookie.Name == security.JWTCookieName {
							wasJWTFound = true
							assert.Equal(t, correctJWTString, cookie.Value)
						}
					}
					res.Body.Close()
					assert.Equal(t, true, wasJWTFound, "JWT cookie wasn`t found")
				}

			}

		})
	}
}
