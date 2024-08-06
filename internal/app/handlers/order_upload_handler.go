package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"time"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

func (h *Handler) OrderUploadHandler(w http.ResponseWriter, r *http.Request) {

	//get request data
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Errorf("error while reading body: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(bodyBytes) == 0 {
		h.Logger.Debugf("empty request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//todo: check with Luna`s alg
	orderNum := string(bodyBytes)
	orderNumInt, err := strconv.Atoi(orderNum)
	if err != nil {
		h.Logger.Errorf("cant convert orderNum into int: %v", err.Error())
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	//get userID
	userID := r.Context().Value(UserIDContextKey)
	if userID == nil {
		h.Logger.Infof("user id wasn`t found in ctx")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userIDInt, ok := userID.(int)
	if !ok {
		h.Logger.Error("userID is not an int")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//save new order
	newOrder := entities.OrderData{
		UserID:     userIDInt,
		Number:     orderNumInt,
		Status:     entities.OrderStatusNew,
		Accural:    0,
		UploadedAt: entities.TimeRFC3339{Time: time.Now()},
	}

	err = h.Storage.SaveNewOrder(newOrder, r.Context())
	if errors.Is(err, gophermart_errors.MakeErrThisOrderWasUploadedByDifferentUser()) {
		h.Logger.Infof("this order was already uploaded by different user")
		w.WriteHeader(http.StatusConflict)
		return
	} else if errors.Is(err, gophermart_errors.MakeErrUserHasAlreadyUploadedThisOrder()) {
		h.Logger.Infof("user has alrdeady uploaded this order")
		w.WriteHeader(http.StatusOK)
		return
	}

	//process order
	go processOrder(newOrder, &h.Storage, h.Logger, 2)

	//return
	w.WriteHeader(http.StatusAccepted)
}

// todo: тесты на функцию
func processOrder(order entities.OrderData, storage *StorageInt, logger zap.SugaredLogger, maxTry int) {
	if maxTry == 0 {
		return
	}
	maxTry--

	//change order status
	order.Status = entities.OrderStatusProcessing
	ctx := context.Background()
	err := (*storage).UpdateOrder(order, ctx)
	if err != nil {
		logger.Errorf("error while updating order data in a storage: %v", err.Error())
		return
	}

	//ask different service
	targetURL := "/api/orders/" + strconv.Itoa(order.Number)
	resp, err := http.Get(targetURL)
	if err != nil {
		order.Status = entities.OrderStatusInvalid
		logger.Errorf("error while making a request: %v", err.Error())

		err = (*storage).UpdateOrder(order, ctx)
		if err != nil {
			logger.Errorf("error while updating order data in a storage: %v", err.Error())
			return
		}
		return
	}

	//get data from a response
	respData := struct {
		Order   int     `json:"order"`
		Status  string  `json:"status"`
		Accural float64 `json:"accural"`
	}{}

	if resp.StatusCode != http.StatusOK {
		order.Status = entities.OrderStatusInvalid
		err = (*storage).UpdateOrder(order, ctx)
		if err != nil {
			logger.Errorf("error while updating order data in a storage: %v", err.Error())
			return
		}
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logger.Errorf("cant read a responce body: %v", err.Error())
		return
	}

	err = json.Unmarshal(bodyBytes, &respData)
	if err != nil {
		logger.Errorf("cant unmurshal a responce body: %v", err.Error())
		return
	}

	//update order
	switch respData.Status {
	case "PROCESSED":
		order.Status = entities.OrderStatusProcessed
		order.Accural = respData.Accural
		err = (*storage).UpdateOrder(order, ctx)
		if err != nil {
			logger.Errorf("error while updating order data in a storage: %v", err.Error())
			return
		}
		err = (*storage).AddToBalance(order.UserID, order.Accural, ctx)
		if err != nil {
			logger.Errorf("cant increase user`s balance, err: %v", err.Error())
			return
		}
	case "PROCESSING":
		time.Sleep(3000 * time.Millisecond)
		processOrder(order, storage, logger, maxTry)
	case "INVALID":
		order.Status = entities.OrderStatusInvalid
		err = (*storage).UpdateOrder(order, ctx)
		if err != nil {
			logger.Errorf("error while updating order data in a storage: %v", err.Error())
			return
		}
	case "REGISTERED":
		//todo кейс registered
	default:
		logger.Errorf("unknown order status was received from outside service: `%v`", respData.Status)
		order.Status = entities.OrderStatusInvalid
		err = (*storage).UpdateOrder(order, ctx)
		if err != nil {
			logger.Errorf("error while updating order data in a storage: %v", err.Error())
			return
		}
	}

}
