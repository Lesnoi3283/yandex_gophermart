package main

import (
	"context"
	"go.uber.org/zap"
	"log"
	"net/http"
	"sync"
	"time"
	"yandex_gophermart/config"
	"yandex_gophermart/internal/app/accrualdaemon"
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
	sugar.Infof("db started")

	//shutdown server if db won`t response
	mainCtx, cancelMainCtx := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(ctx context.Context, cancelMainCtxFunc context.CancelFunc, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				dbErr := pg.Ping()
				if dbErr != nil {
					sugar.Errorf("DB ping error (shutting down a server...), err: %v", dbErr.Error())
					cancelMainCtxFunc()
					return
				} else {
					time.Sleep(time.Second * 3)
				}
			}
		}
	}(mainCtx, cancelMainCtx, &wg)

	//start an accrual daemon
	wg.Add(1)
	go accrualdaemon.AccrualCheckDaemon(mainCtx, sugar, pg, cfg.AccrualSystemAddress, &wg)
	sugar.Infof("starting an accrual daemon")

	//router set and server start
	router := handlers.NewRouter(*sugar, pg, cfg.AccrualSystemAddress)
	sugar.Infof("starting server")
	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		sugar.Fatalf("Cant run a server, err: %v", server.ListenAndServe().Error())
	}(&wg)

	//shutdown server when context is cancelled
	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		<-ctx.Done() //program will wait here
		errSh := server.Shutdown(context.Background())
		if errSh != nil {
			sugar.Fatalf("Tryed to shutdown server carefully, but got en error. Shutting down with Fatalf(). Err: %v", errSh.Error())
		}
	}(mainCtx, &wg)

	wg.Wait()
}
