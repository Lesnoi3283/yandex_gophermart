package handlers

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {

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

	//getting balance from db
	balance, err := h.Storage.GetBalance(userIDInt, r.Context())
	if err != nil {
		h.Logger.Errorf("error while getting balance from db: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//return
	jsonToRet, err := json.Marshal(balance)
	if err != nil {
		h.Logger.Errorf("error while marshalling balance: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonToRet)

}
