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
	gophermarterrors "yandex_gophermart/pkg/errors"
)

func TestHandler_WithdrawHandler(t *testing.T) {

	//logger set
	logger := zaptest.NewLogger(t)
	sugar := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

	//data set
	correctUserID := 1
	correctOrderId := 2377225624
	correctSum := 750.0
	makeRequestBody := func() io.Reader {
		data := struct {
			Order int
			Sum   float64
		}{
			Order: correctOrderId,
			Sum:   correctSum,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			sugar.Errorf("withdraw handler test, err while building json request: %v", err.Error())
		}
		return bytes.NewReader(jsonData)
	}

	type fields struct {
		Logger  *zap.SugaredLogger
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
				Logger: sugar,
				Storage: func() StorageInt {
					store := mock_handlers.NewMockStorageInt(controller)
					store.EXPECT().WithdrawFromBalance(correctUserID, correctOrderId, correctSum, gomock.Any()).Return(nil)
					return store
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", makeRequestBody()).WithContext(context.WithValue(context.Background(), UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusOK,
		},
		{
			name: "not auth",
			fields: fields{
				Logger: sugar,
				Storage: func() StorageInt {
					store := mock_handlers.NewMockStorageInt(controller)
					return store
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", makeRequestBody()),
			},
			statusWant: http.StatusUnauthorized,
		},
		{
			name: "Not enough points",
			fields: fields{
				Logger: sugar,
				Storage: func() StorageInt {
					store := mock_handlers.NewMockStorageInt(controller)
					store.EXPECT().WithdrawFromBalance(correctUserID, correctOrderId, correctSum, gomock.Any()).Return(gophermarterrors.MakeErrNotEnoughPoints())
					return store
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", makeRequestBody()).WithContext(context.WithValue(context.Background(), UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusPaymentRequired,
		},
		{
			name: "order not found",
			fields: fields{
				Logger: sugar,
				Storage: func() StorageInt {
					store := mock_handlers.NewMockStorageInt(controller)
					store.EXPECT().WithdrawFromBalance(correctUserID, correctOrderId, correctSum, gomock.Any()).Return(gophermarterrors.MakeErrOrderNotFound())
					return store
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", makeRequestBody()).WithContext(context.WithValue(context.Background(), UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Logger:  *tt.fields.Logger,
				Storage: tt.fields.Storage,
			}
			h.WithdrawHandler(tt.args.w, tt.args.r)
			assert.Equal(t, tt.statusWant, tt.args.w.Code, "different status expected")
		})
	}
}
