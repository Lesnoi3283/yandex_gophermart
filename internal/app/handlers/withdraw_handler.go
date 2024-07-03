package handlers

import "net/http"

func (h *Handler) WithdrawHandler(w http.ResponseWriter, r *http.Request) {

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

}
