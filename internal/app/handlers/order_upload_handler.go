package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"yandex_gophermart/internal/app/middlewares"
	"yandex_gophermart/pkg/entities"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

func checkWithLuna(num string) (bool, error) {
	sum := 0
	double := false

	//count sum
	for i := len(num) - 1; i >= 0; i-- {
		cur, err := strconv.Atoi(num[i : i+1])
		if err != nil {
			return false, fmt.Errorf("cant count a luna`s sum : %w", err)
		}

		if double {
			cur = cur * 2
			if cur > 9 {
				cur = cur - 9
			}
		}

		sum += cur
		double = !double
	}

	//check
	return sum%10 == 0, nil
}

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
	
	//check with Luna`s alg
	orderNum := string(bodyBytes)
	ok, err := checkWithLuna(orderNum)
	if err != nil {
		h.Logger.Errorf("cant do a Luna`s check: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !ok {
		h.Logger.Debugf("order num `%s` is incorrect", orderNum)
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	//get userID
	userID := r.Context().Value(middlewares.UserIDContextKey)
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
		Number:     orderNum,
		Status:     entities.OrderStatusNew,
		Accrual:    0,
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

	//return
	w.WriteHeader(http.StatusAccepted)
}
