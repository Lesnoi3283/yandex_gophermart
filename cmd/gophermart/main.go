package main

import (
	"go.uber.org/zap"
	"log"
	"net/http"
	"yandex_gophermart/config"
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
		sugar.Infof("db started")
	} else {
		sugar.Fatalf("db ping error (afterstart check): %v", err.Error())
	}

	//router set and server start
	router := handlers.NewRouter(*sugar, pg, cfg.AccrualSystemAddress)
	sugar.Fatalf("failed to start a server:", http.ListenAndServe(cfg.RunAddress, router).Error())

}
