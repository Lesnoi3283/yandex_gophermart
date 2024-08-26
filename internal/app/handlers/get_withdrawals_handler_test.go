package handlers

import (
	"bytes"
	"context"
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
	"time"
	mock_handlers "yandex_gophermart/internal/app/handlers/mocks"
	"yandex_gophermart/internal/app/middlewares"
	"yandex_gophermart/pkg/entities"
)

func TestHandler_GetWithdrawals(t *testing.T) {
	//logger set
	logger := zaptest.NewLogger(t)
	sugared := logger.Sugar()

	//mocks set
	controller := gomock.NewController(t)

	//data set
	correctUserID := 2
	correctOrderID := "2377225624"
	correctSum := 500.0
	correctOrderTime, err := time.Parse(time.RFC3339, "2020-12-09T16:09:57+03:00")
	if err != nil {
		require.NoError(t, err, "error while preparing tests (while parsing time)")
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
		answerWant []byte
	}{
		{
			name: "normal",
			fields: fields{
				Logger: *sugared,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().GetWithdrawals(gomock.Any(), correctUserID).Return([]entities.WithdrawalData{
						{
							OrderNum:    correctOrderID,
							Sum:         correctSum,
							ProcessedAt: entities.TimeRFC3339{Time: correctOrderTime},
						},
					}, nil)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusOK,
			answerWant: func() []byte {
				correctResponceData := []struct {
					Order       string               `json:"order"`
					Sum         float64              `json:"sum"`
					ProcessedAt entities.TimeRFC3339 `json:"processed_at"`
				}{
					{
						Order:       correctOrderID,
						Sum:         correctSum,
						ProcessedAt: entities.TimeRFC3339{Time: correctOrderTime},
					},
				}
				JSONCorrectResponceData, err := json.Marshal(correctResponceData)
				if err != nil {
					require.NoError(t, err, "error while preparing tests (while marshalling responseData)")
				}
				return JSONCorrectResponceData
			}(),
		},
		{
			name: "no withdrawals",
			fields: fields{
				Logger: *sugared,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().GetWithdrawals(gomock.Any(), correctUserID).Return([]entities.WithdrawalData{}, nil)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusNoContent,
			answerWant: []byte(""),
		},
		{
			name: "not auth",
			fields: fields{
				Logger: *sugared,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil),
			},
			statusWant: http.StatusUnauthorized,
			answerWant: []byte(""),
		},
		{
			name: "db error",
			fields: fields{
				Logger: *sugared,
				Storage: func() StorageInt {
					storage := mock_handlers.NewMockStorageInt(controller)
					storage.EXPECT().GetWithdrawals(gomock.Any(), correctUserID).Return([]entities.WithdrawalData{}, errors.New("some test error"))
					return storage
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil).WithContext(context.WithValue(context.Background(), middlewares.UserIDContextKey, correctUserID)),
			},
			statusWant: http.StatusInternalServerError,
			answerWant: []byte(""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Logger:  tt.fields.Logger,
				Storage: tt.fields.Storage,
			}
			h.GetWithdrawals(tt.args.w, tt.args.r)
			if assert.Equal(t, tt.statusWant, tt.args.w.Code, "wrong status code") {
				if tt.args.w.Code == http.StatusOK {
					assert.Equal(t, bytes.NewBuffer(tt.answerWant), tt.args.w.Body, "wrong response")
				}
			}
		})
	}
}
