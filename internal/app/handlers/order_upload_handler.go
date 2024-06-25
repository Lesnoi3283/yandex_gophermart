package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	gophermart_errors "yandex_gophermart/pkg/errors"
)

func (h *Handler) OrderUploadHandler(w http.ResponseWriter, r *http.Request) {

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Errorf("error while reading body: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//todo: check with Luna`s alg
	orderID := string(bodyBytes)
	orderIDInt, err := strconv.Atoi(orderID)
	if err != nil {
		h.Logger.Errorf("cant convert orderID into int: %v", err.Error())
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	userID := r.Context().Value("userID")
	if userID == nil {
		h.Logger.Infof("user id wasn`t found in ctx")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userIDInt, ok := userID.(int)
	if !ok {
		h.Logger.Infof("userID is not an int")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.Storage.SaveNewOrder(userIDInt, orderIDInt, r.Context())
	if errors.Is(err, gophermart_errors.MakeErrThisOrderWasUploadedByDifferentUser()) {
		h.Logger.Infof("this order was already uploaded by different user")
		w.WriteHeader(http.StatusConflict)
		return
	} else if errors.Is(err, gophermart_errors.MakeErrUserHasAlreadyUploadedThisOrder()) {
		h.Logger.Infof("user has alrdeady uploaded this order")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	return
}
