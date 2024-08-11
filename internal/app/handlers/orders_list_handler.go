package handlers

import (
	"encoding/json"
	"net/http"
	"yandex_gophermart/internal/app/middlewares"
)

func (h *Handler) OrdersListHandler(w http.ResponseWriter, r *http.Request) {
	h.Logger.Debug("OrderListHandler recieved a request")

	w.Header().Set("Content-Type", "application/json")
	//get userID
	userID := r.Context().Value(middlewares.UserIDContextKey)
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

	//getting orders from db
	orders, err := h.Storage.GetOrdersList(userIDInt, r.Context())
	if err != nil {
		h.Logger.Errorf("error while getting orders list from db: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.Logger.Debugf("orders amout from db: %d", len(orders))

	//return
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
	} else {
		jsonToRet, err := json.Marshal(orders)
		if err != nil {
			h.Logger.Errorf("error while marshalling orders list: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonToRet)
	}
}
