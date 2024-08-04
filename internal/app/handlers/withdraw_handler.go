package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	gophermarterrors "yandex_gophermart/pkg/errors"
)

type withdrawData struct {
	OrderID int     `json:"order"`
	Sum     float64 `json:"sum"`
}

func (h *Handler) WithdrawHandler(w http.ResponseWriter, r *http.Request) {

	//get userID
	userID := r.Context().Value(UserIDContextKey)
	if userID == nil {
		h.Logger.Debugf("user id wasn`t found in ctx")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userIDInt, ok := userID.(int)
	if !ok {
		h.Logger.Error("userID is not an int")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//get data
	data := withdrawData{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Errorf("cant read request body: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		h.Logger.Errorf("cant unmarshal request body: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//withdraw from db
	err = h.Storage.WithdrawFromBalance(userIDInt, data.OrderID, data.Sum, r.Context())
	if errors.Is(err, gophermarterrors.MakeErrNotEnoughPoints()) {
		h.Logger.Debugf("Not enough money, err: %v", err)
		w.WriteHeader(http.StatusPaymentRequired)
		return
	} else if errors.Is(err, gophermarterrors.MakeErrOrderNotFound()) {
		h.Logger.Debugf("Order not found, err: %v", err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	} else if err != nil {
		h.Logger.Errorf("cant withdraw points, err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}
