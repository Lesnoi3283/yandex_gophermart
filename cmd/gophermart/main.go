package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"yandex_gophermart/config"
	"yandex_gophermart/internal/app/accrual_daemon"
	"yandex_gophermart/internal/app/handlers"
	"yandex_gophermart/pkg/databases"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

func main() {
	//conf
	cfg := config.Config{}
	cfg.Configure()

	//logger set
	zCfg := zap.NewProductionConfig()
	level, err := zap.ParseAtomicLevel(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Cant parse log level, err: %v", err)
	}
	zCfg.Level = level
	zCfg.DisableStacktrace = true
	logger, err := zCfg.Build()
	if err != nil {
		log.Fatalf("logger was not started, err: %v", err)
	}
	sugar := logger.Sugar()

	//db set
	pg, err := databases.NewPostgresql(cfg.DBConnStr)
	if err != nil {
		sugar.Fatalf("cant start database, err: %v", err.Error())
	}
	err = pg.SetTables()
	if err != nil {
		sugar.Fatalf("error while setting tables in database, err: %v", err.Error())
	}
	err = pg.Ping()
	if err != nil {
		sugar.Fatalf("db ping error (afterstart check): %v", err.Error())
	} else {
		sugar.Infof("db started")
	}

	//start a accrual daemon
	//go accrual_daemon.ProcessOrders(context.Background(), cfg.AccrualSystemAddress, pg, sugar)
	//sugar.Infof("starting an accrual daemon")

	go ProcessOrders(context.Background(), cfg.AccrualSystemAddress, pg, sugar)

	//router set and server start
	router := handlers.NewRouter(*sugar, pg, cfg.AccrualSystemAddress)
	sugar.Infof("starting server")
	sugar.Fatalf("failed to start a server:", http.ListenAndServe(cfg.RunAddress, router).Error())
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
			//retryAfter := resp.Header.Get("Retry-After")
			//seconds, err := strconv.Atoi(retryAfter)
			//if err != nil {
			//	date, err := time.Parse(time.RFC1123, retryAfter)
			//	if err != nil {
			//		return respData{}, fmt.Errorf("cant parse a retry-after header")
			//	}
			//	time.Sleep(time.Until(date))
			//} else if retryAfter == "" {
			//	time.Sleep(time.Second * 3)
			//} else {
			//	time.Sleep(time.Duration(seconds))
			//}
			return respData{}, gophermart_errors.MakeErrNeedToResendRequestAccrual()
		}

	default:
		return respData{}, fmt.Errorf("unprdefictable responce status code %v", resp.StatusCode)
	}

}

// ProcessOrders MUST be run as goroutine. It has an endless "for" loop. Use ctx to break it
func ProcessOrders(ctx context.Context, accrualSystemAddress string, storage accrual_daemon.UnfinishedOrdersStorageInt, logger *zap.SugaredLogger) {

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

			//i ll increase it just for test
			i++
			continue
		} else if err != nil {
			logger.Warnf("cant process an order with id `%v`, err: %v", orders[i].ID, err.Error())
			i++
			continue
		}

		//update order in a storage
		orderData := orders[i]
		orderData.Status = data.Status
		orderData.Accural = data.Accural
		err = storage.UpdateOrder(orderData, ctx)
		if err != nil {
			logger.Errorf("error while updating order data in a storage: %v", err.Error())
			i++
			continue
		}

		//increase users`s balance
		err = storage.AddToBalance(orderData.UserID, orderData.Accural, ctx)
		if err != nil {
			logger.Errorf("error while increasing users balance in a storage: %v", err.Error())
			i++
			continue
		}
		i++
	}
}

//
//func someTestGoroutine(ctx context.Context, logger *zap.SugaredLogger, storage accrual_daemon.UnfinishedOrdersStorageInt, accrualSystemAddress string) {
//	logger.Infof("TEST GOROUTINE STARTED")
//loop:
//	for {
//		select {
//		case <-ctx.Done():
//			break loop
//		default:
//			time.Sleep(time.Millisecond * 200)
//			smg, _ := storage.GetUnfinishedOrdersList(ctx)
//			logger.Infof("TEST GOROUTINE IS RUNNUNG, orders amount: %v", len(smg))
//
//			if len(smg) > 0 {
//				resp := someDifferentTestFunc(accrualSystemAddress, smg[0], logger)
//				switch resp.StatusCode {
//				case http.StatusOK:
//					{
//						bodyBytes, err := io.ReadAll(resp.Body)
//						defer resp.Body.Close()
//						if err != nil {
//							logger.Errorf("TEST G cant read a responce body: %v", err.Error())
//						}
//						data := respData{}
//						err = json.Unmarshal(bodyBytes, &data)
//						if err != nil {
//							logger.Errorf("TEST G cant unmurshal a responce body: %v", err.Error())
//						}
//						logger.Infof("TEST G resp data: %#v", data)
//
//						order := smg[0]
//						order.Status = data.Status
//						order.Accural = data.Accural
//						err = storage.UpdateOrder(order, ctx)
//						if err != nil {
//							logger.Errorf("TEST G err: %v", err.Error())
//						}
//						logger.Infof("TEST G Updated")
//					}
//				}
//			}
//
//		}
//	}
//}
//
//func someDifferentTestFunc(accrualSystemAddress string, smg entities.OrderData, logger *zap.SugaredLogger) *http.Response {
//	targetURL := accrualSystemAddress + "/api/orders/" + smg.Number
//	resp, err := http.Get(targetURL)
//	if err != nil {
//		logger.Error("TEST G err : %v", err.Error())
//	}
//	logger.Infof("TEST G resp: %#v", resp)
//	return resp
//}
