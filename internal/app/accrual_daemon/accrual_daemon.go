package accrual_daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"time"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

//
//const (
//	orderStatusRegistered = "REGISTERED"
//	orderStatusInvalid    = "INVALID"
//	orderStatusProcessing = "PROCESSING"
//	orderStatusProcessed  = "PROCESSED"
//)

type UnfinishedOrdersStorageInt interface {
	UpdateOrder(orderData entities.OrderData, ctx context.Context) error
	GetUnfinishedOrdersList(ctx context.Context) ([]entities.OrderData, error)
	AddToBalance(userID int, amount float64, ctx context.Context) error
}

type respData struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accural float64 `json:"accural"`
}

func askAccrualSystem(orderNum string, accrualSystemAddress string, logger *zap.SugaredLogger) (respData, error) {
	//make a request
	targetURL := accrualSystemAddress + "/api/orders/" + orderNum
	resp, err := http.Get(targetURL)
	if err != nil {
		return respData{}, fmt.Errorf("error while requesting an accrual system: %w", err)
	}

	//get data from a response
	switch resp.StatusCode {
	case http.StatusOK:
		{
			bodyBytes, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				logger.Errorf("cant read a responce body: %v", err.Error())
				return respData{}, fmt.Errorf("cant read an accrual system responce: %w", err)
			}
			data := respData{}
			err = json.Unmarshal(bodyBytes, &data)
			if err != nil {
				logger.Errorf("cant unmurshal a responce body: %v", err.Error())
				return respData{}, fmt.Errorf("cant unmurshal an accrual system responce: %w", err)
			}
			return data, nil
		}
	case http.StatusNoContent:
		{
			return respData{}, gophermart_errors.MakeErrNoContentAccrual()
		}
	case http.StatusInternalServerError:
		{
			return respData{}, gophermart_errors.MakeErrInternalServerErrorAccrual()
		}
	case http.StatusTooManyRequests:
		{
			retryAfter := resp.Header.Get("Retry-After")
			seconds, err := strconv.Atoi(retryAfter)
			if err != nil {
				date, err := time.Parse(time.RFC1123, retryAfter)
				if err != nil {
					return respData{}, fmt.Errorf("cant parse a retry-after header")
				}
				time.Sleep(time.Until(date))
			} else if retryAfter == "" {
				time.Sleep(time.Second * 3)
			} else {
				time.Sleep(time.Duration(seconds))
			}
			return respData{}, gophermart_errors.MakeErrNeedToResendRequestAccrual()
		}

	default:
		return respData{}, fmt.Errorf("unprdefictable responce status code %v", resp.StatusCode)
	}

}

// ProcessOrders MUST be run as goroutine. It has an endless "for" loop. Use ctx to break it
func ProcessOrders(ctx context.Context, accrualSystemAddress string, storage UnfinishedOrdersStorageInt, logger *zap.SugaredLogger) {

	orders := make([]entities.OrderData, 0)
	i := 0

loop:
	for {
		//check if goroutine has to die
		select {
		case <-ctx.Done():
			logger.Infof("Accrual daemon died because ctx is done")
			break loop
		default:
			//do nothing
		}

		//get new unfinished orders
		if i >= len(orders) {
			var err error
			orders, err = storage.GetUnfinishedOrdersList(ctx)
			if err != nil {
				logger.Errorf("cant get unfinished orders from db, err: %v", err.Error())
				return
			}
			i = 0
		}

		//ask accrual system
		data, err := askAccrualSystem(orders[i].Number, accrualSystemAddress, logger)
		if errors.Is(err, gophermart_errors.MakeErrNoContentAccrual()) {
			i++
			continue
		} else if errors.Is(err, gophermart_errors.MakeErrInternalServerErrorAccrual()) {
			i++
			continue
		} else if errors.Is(err, gophermart_errors.MakeErrNeedToResendRequestAccrual()) {
			//DONT INCREASE AN "i" HERE!
			continue
		} else if err != nil {
			logger.Warnf("cant process an order with id `%v`, err: %v", orders[i].ID, err.Error())
			i++
			continue
		}

		//update order in a storage
		orderData := orders[i]
		orderData.Status = data.Status
		orderData.Accrual = data.Accural
		err = storage.UpdateOrder(orderData, ctx)
		if err != nil {
			logger.Errorf("error while updating order data in a storage: %v", err.Error())
			i++
			continue
		}

		//increase users`s balance
		err = storage.AddToBalance(orderData.UserID, orderData.Accrual, ctx)
		if err != nil {
			logger.Errorf("error while increasing users balance in a storage: %v", err.Error())
			i++
			continue
		}
		i++
	}
}
