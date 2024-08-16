package main

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"time"
	"yandex_gophermart/config"
	"yandex_gophermart/internal/app/accrual_daemon"
	"yandex_gophermart/internal/app/handlers"
	"yandex_gophermart/pkg/databases"
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

	go someTestGoroutine(context.Background(), sugar, pg, cfg.AccrualSystemAddress)

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

func someTestGoroutine(ctx context.Context, logger *zap.SugaredLogger, storage accrual_daemon.UnfinishedOrdersStorageInt, accrualSystemAddress string) {
	logger.Infof("TEST GOROUTINE STARTED")
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			time.Sleep(time.Millisecond * 200)
			smg, _ := storage.GetUnfinishedOrdersList(ctx)
			logger.Infof("TEST GOROUTINE IS RUNNUNG, orders amount: %v", len(smg))

			if len(smg) > 0 {
				targetURL := accrualSystemAddress + "/api/orders/" + smg[0].Number
				resp, err := http.Get(targetURL)
				if err != nil {
					logger.Error("TEST G err : %v", err.Error())
				}
				logger.Infof("TEST G resp: %#v", resp)

				switch resp.StatusCode {
				case http.StatusOK:
					{
						bodyBytes, err := io.ReadAll(resp.Body)
						defer resp.Body.Close()
						if err != nil {
							logger.Errorf("TEST G cant read a responce body: %v", err.Error())
						}
						data := respData{}
						err = json.Unmarshal(bodyBytes, &data)
						if err != nil {
							logger.Errorf("TEST G cant unmurshal a responce body: %v", err.Error())
						}
						logger.Infof("TEST G resp data: %#v", data)

						order := smg[0]
						order.Status = data.Status
						order.Accural = data.Accural
						err = storage.UpdateOrder(order, ctx)
						if err != nil {
							logger.Errorf("TEST G err: %v", err.Error())
						}
						logger.Infof("TEST G Updated")
					}
				}
			}

		}
	}
}
