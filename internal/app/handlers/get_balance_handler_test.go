package handlers

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"net/http"
	"net/http/httptest"
	"testing"
	mock_handlers "yandex_gophermart/internal/app/handlers/mocks"
	"yandex_gophermart/internal/app/middlewares"
	"yandex_gophermart/pkg/entities"
)

func TestHandler_GetBalanceHandler(t *testing.T) {

	//logger set
	logger := zaptest.NewLogger(t)
	sugarLogger := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

	//data set
	correctUserID := 1

	correctBalance := entities.BalanceData{
		ID:        1,
		UserID:    correctUserID,
		Current:   500.5,
		Withdrawn: 42,
	}

	//tests set
	type fields struct {
		Logger  zap.SugaredLogger
		Storage StorageInt
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
					storage.EXPECT().GetBalance(gomock.Any(), 1).Return(correctBalance, nil)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/balance", nil).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusOK,
		},
		{
			name: "no user ID",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/balance", nil),
			},
			statusWant: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Logger:  tt.fields.Logger,
				Storage: tt.fields.Storage,
			}
			h.GetBalanceHandler(tt.args.w, tt.args.r)

			assert.Equal(t, tt.statusWant, tt.args.w.Code, "wrong status code")
		})
	}
}
