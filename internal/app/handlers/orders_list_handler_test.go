package handlers

import (
	"context"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	mock_handlers "yandex_gophermart/internal/app/handlers/mocks"
	"yandex_gophermart/pkg/entities"
)

func TestHandler_OrdersListHandler(t *testing.T) {

	//logger set
	logger := zaptest.NewLogger(t)
	sugarLogger := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

	//data set
	correctUserID := 1

	correctOrdersList := []entities.OrderData{
		{
			ID:         1,
			UserID:     correctUserID,
			Number:     12345678903,
			Status:     entities.OrderStatusNew,
			UploadedAt: time.Date(2020, 12, 20, 18, 30, 0, 0, time.FixedZone("GMT+3", 60*60*3)),
		},
		{
			ID:         2,
			UserID:     correctUserID,
			Number:     900,
			Status:     entities.OrderStatusNew,
			UploadedAt: time.Date(2021, 12, 20, 18, 30, 0, 0, time.FixedZone("GMT+3", 60*60*3)),
		},
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
		checkBody  bool
	}{
		{
			name: "normal",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().GetOrdersList(1, gomock.Any()).Return(correctOrdersList, nil)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/api/user/orders", nil).WithContext(context.WithValue(context.Background(), UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusOK,
			checkBody:  true,
		},
		{
			name: "empty orders list",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().GetOrdersList(1, gomock.Any()).Return(make([]entities.OrderData, 0), nil)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/api/user/orders", nil).WithContext(context.WithValue(context.Background(), UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusNoContent,
			checkBody:  false,
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
				r: httptest.NewRequest(http.MethodGet, "/api/user/orders", nil),
			},
			statusWant: http.StatusUnauthorized,
			checkBody:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Logger:  tt.fields.Logger,
				Storage: tt.fields.Storage,
			}
			h.OrdersListHandler(tt.args.w, tt.args.r)

			assert.Equal(t, tt.statusWant, tt.args.w.Code, "wrong status code")

			if tt.checkBody {
				ordersListJSON, err := json.Marshal(correctOrdersList)
				require.NoError(t, err, "cant marshal test data (error occurred in test, not in a testing func)")

				assert.Equal(t, ordersListJSON, tt.args.w.Body.Bytes())
			}
		})
	}
}
