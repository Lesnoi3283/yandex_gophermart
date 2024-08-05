package handlers

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	//get user data
	userID := r.Context().Value(UserIDContextKey)
	if userID == nil {
		h.Logger.Debugf("no user ID was found in context")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userIDInt, ok := userID.(int)
	if !ok {
		h.Logger.Errorf("user ID is not an int")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//get withdrawals
	withdrawals, err := h.Storage.GetWithdrawals(userIDInt, r.Context())
	if err != nil {
		h.Logger.Errorf("cant get withdrawals from db, err: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		h.Logger.Debugf("no content")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	//marshall response data
	JSONData, err := json.Marshal(withdrawals)
	if err != nil {
		h.Logger.Errorf("cant marshal withdrawals, err: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//write response
	w.WriteHeader(http.StatusOK)
	w.Write(JSONData)
}
