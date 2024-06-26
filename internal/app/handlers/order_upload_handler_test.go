package handlers

import (
	"bytes"
	"context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"net/http"
	"net/http/httptest"
	"testing"
	mock_handlers "yandex_gophermart/internal/app/handlers/mocks"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

func TestHandler_OrderUploadHandler(t *testing.T) {

	//logger set
	logger := zaptest.NewLogger(t)
	sugarLogger := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

	//data set
	correctOrderID := []byte("1234567890")
	correctOrderIDInt := 1234567890
	correctUserID := 1

	//tests set
	type fields struct {
		Logger  zap.SugaredLogger
		Storage StorageInt
		//JWTH    JWTHelperInt
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
					storage.EXPECT().SaveNewOrder(correctUserID, correctOrderIDInt, gomock.Any()).Return(nil)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderID)).WithContext(context.WithValue(context.Background(), "userID", correctUserID)),
			},
			statusWant: http.StatusAccepted,
		},
		{
			name: "was already uploaded",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveNewOrder(correctUserID, correctOrderIDInt, gomock.Any()).Return(gophermart_errors.MakeErrUserHasAlreadyUploadedThisOrder())
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderID)).WithContext(context.WithValue(context.Background(), "userID", correctUserID)),
			},
			statusWant: http.StatusOK,
		},
		{
			name: "conflict",
			fields: fields{
				Logger: *sugarLogger,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().SaveNewOrder(correctUserID, correctOrderIDInt, gomock.Any()).Return(gophermart_errors.MakeErrThisOrderWasUploadedByDifferentUser())
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderID)).WithContext(context.WithValue(context.Background(), "userID", correctUserID)),
			},
			statusWant: http.StatusConflict,
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
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte("123sometext123"))).WithContext(context.WithValue(context.Background(), "userID", correctUserID)),
			},
			statusWant: http.StatusUnprocessableEntity,
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
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(""))).WithContext(context.WithValue(context.Background(), "userID", correctUserID)),
			},
			statusWant: http.StatusBadRequest,
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
				r: httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader(correctOrderID)),
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
			h.OrderUploadHandler(tt.args.w, tt.args.r)

			assert.Equal(t, tt.statusWant, tt.args.w.Code, "wrong status code")
		})
	}
}
