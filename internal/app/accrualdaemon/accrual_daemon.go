package accrualdaemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

const (
	dbWaitLong  = time.Second * 30
	dbWaitShort = time.Millisecond * 100
)

type UnfinishedOrdersStorageInt interface {
	UpdateOrder(orderData entities.OrderData, ctx context.Context) error
	GetUnfinishedOrdersList(ctx context.Context) ([]entities.OrderData, error)
	AddToBalance(userID int, amount float64, ctx context.Context) error
}

type respData struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

func AccrualCheckDaemon(ctx context.Context, logger *zap.SugaredLogger, storage UnfinishedOrdersStorageInt, accrualSystemAddress string, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Infof("Accrual daemon started")

	orders := make([]entities.OrderData, 0)
	i := 0

	//should be 0 at start (default value)
	var waitBeforeNewDBRequest time.Duration

	for {
		//get new unfinished orders
		if i >= len(orders) {
			//to not spam our db with a lot of requests
			time.Sleep(waitBeforeNewDBRequest)

			var err error
			orders, err = storage.GetUnfinishedOrdersList(ctx)
			if err != nil {
				logger.Errorf("cant get unfinished orders from db, err: %v", err.Error())
				return
			}
			i = 0

			if len(orders) > 0 {
				waitBeforeNewDBRequest = dbWaitShort
			} else {
				waitBeforeNewDBRequest = dbWaitLong
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
			//process order
			if len(orders) > 0 {
				data, err := askAccrual(accrualSystemAddress, orders[i], logger)
				if errors.Is(err, gophermart_errors.MakeErrNeedToResendRequestAccrual()) {
					//DONT INCREASE AN "i" HERE!
					continue
				} else if errors.Is(err, gophermart_errors.MakeErrNoContentAccrual()) {
					i++
					continue
				} else if errors.Is(err, gophermart_errors.MakeErrInternalServerErrorAccrual()) {
					i++
					continue
				} else {
					//update an order in db
					order := orders[i]
					order.Status = data.Status
					order.Accrual = data.Accrual
					err = storage.UpdateOrder(order, ctx)
					if err != nil {
						logger.Errorf(" err: %v", err.Error())
						i++
						continue
					}

					//increase users`s balance
					err = storage.AddToBalance(order.UserID, order.Accrual, ctx)
					if err != nil {
						logger.Errorf("error while increasing users balance in a storage: %v", err.Error())
						i++
						continue
					}
					i++
				}

			}

		}
	}
}

func askAccrual(accrualSystemAddress string, smg entities.OrderData, logger *zap.SugaredLogger) (respData, error) {
	targetURL := accrualSystemAddress + "/api/orders/" + smg.Number
	resp, err := http.Get(targetURL)
	if err != nil {
		logger.Error("cant send a request, err : %v", err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		{
			//read response
			bodyBytes, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				return respData{}, fmt.Errorf("cant read a responce body, err: %w", err)
			}
			//parse response
			data := respData{}
			err = json.Unmarshal(bodyBytes, &data)
			if err != nil {
				return respData{}, fmt.Errorf("cant unmurshal a responce body: %w", err)
			}
			return data, nil
		}
	case http.StatusTooManyRequests:
		{
			//daemon will sleep here
			retryAfter := resp.Header.Get("Retry-After")
			seconds, err := strconv.Atoi(retryAfter)
			if err != nil {
				date, err := time.Parse(time.RFC1123, retryAfter)
				if err != nil {
					logger.Errorf("TEST G error while parsing Retry-After: %v", err.Error())
				}
				time.Sleep(time.Until(date))
			} else if retryAfter == "" {
				time.Sleep(time.Second * 3)
			} else {
				time.Sleep(time.Duration(seconds) * time.Second)
			}

			return respData{}, gophermart_errors.MakeErrNeedToResendRequestAccrual()
		}
	case http.StatusNoContent:
		{
			return respData{}, gophermart_errors.MakeErrNoContentAccrual()
		}
	case http.StatusInternalServerError:
		{
			return respData{}, gophermart_errors.MakeErrInternalServerErrorAccrual()
		}
	default:
		return respData{}, fmt.Errorf("unprdefictable responce status code %v", resp.StatusCode)
	}
}
