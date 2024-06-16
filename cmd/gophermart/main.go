package main

import (
	"go.uber.org/zap"
	"log"
	"net/http"
	"yandex_gophermart/internal/app/handlers"
)

func main() {
	//logger set
	zCfg := zap.NewProductionConfig()
	logger, err := zCfg.Build()
	if err != nil {
		log.Fatalf("logger was not started, err: %v", err)
	}
	sugar := logger.Sugar()

	//router set and server start
	router := handlers.NewRouter()
	sugar.Fatalf("failed to start a server:", http.ListenAndServe("127.0.0.1:8080", router).Error())

}
