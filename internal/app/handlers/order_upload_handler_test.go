package handlers

import (
	"bytes"
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	mock_handlers "yandex_gophermart/internal/app/handlers/mocks"
	"yandex_gophermart/internal/app/middlewares"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

func TestHandler_OrderUploadHandler(t *testing.T) {

	//logger set
	logger := zaptest.NewLogger(t)
	sugarLogger := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

	//data set
	correctUserID := 2
	correctOrderNumBytes := []byte("1234567890")

	//wait group set
	wg := sync.WaitGroup{}

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
		wgAmout    int
	}{
		{
			name: "normal",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveNewOrder(gomock.Any(), gomock.Any()).Return(nil)
					storage.EXPECT().UpdateOrder(gomock.Any(), gomock.Any()).DoAndReturn(func(order entities.OrderData, ctx context.Context) error {
						wg.Done()
						return errors.New("all good")
					})
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderNumBytes)).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusAccepted,
			wgAmout:    1,
		},
		{
			name: "was already uploaded",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveNewOrder(gomock.Any(), gomock.Any()).Return(gophermart_errors.MakeErrUserHasAlreadyUploadedThisOrder())
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderNumBytes)).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusOK,
			wgAmout:    0,
		},
		{
			name: "conflict",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveNewOrder(gomock.Any(), gomock.Any()).Return(gophermart_errors.MakeErrThisOrderWasUploadedByDifferentUser())
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderNumBytes)).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusConflict,
			wgAmout:    0,
		},
		{
			name: "broken order id",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte("123sometext123"))).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusUnprocessableEntity,
			wgAmout:    0,
		},
		{
			name: "no order id",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", nil).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusBadRequest,
			wgAmout:    0,
		},
		{
			name: "no user id",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderNumBytes)),
			},
			statusWant: http.StatusUnauthorized,
			wgAmout:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Logger:  tt.fields.Logger,
				Storage: tt.fields.Storage,
			}
			h.OrderUploadHandler(tt.args.w, tt.args.r)
			wg.Add(tt.wgAmout)
			wg.Wait()

			assert.Equal(t, tt.statusWant, tt.args.w.Code, "wrong status code")
		})
	}
}
