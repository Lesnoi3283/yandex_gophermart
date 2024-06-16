package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	g_errors "yandex_gophermart/pkg/errors"
)

type userRegData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) registerUser(w http.ResponseWriter, r *http.Request) {
	//getting user data
	body := make([]byte, 0)
	r.Body.Read(body)
	r.Body.Close()
	uData := userRegData{}
	err := json.Unmarshal(body, &uData)
	if err != nil {
		h.Logger.Errorf("unmarshal err: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//creating user
	uID, err := h.Storage.saveUser(uData.Login, uData.Password, r.Context())
	if errors.Is(err, g_errors.MakeErrUserAlreadyExists()) {
		h.Logger.Infof("user already exists: %v", err.Error())
		w.WriteHeader(http.StatusConflict)
		_, err := w.Write([]byte("this user already exists"))
		if err != nil {
			h.Logger.Errorf("response write err: %v", err.Error())
			return
		}
		return
	}

	//creating and setting jwt token
	jwtString, err := h.JWTH.BuildNewJWTString(uID)
	if err != nil {
		h.Logger.Errorf("jwt err: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//todo: придумать где определить константу названия куки жвт токена
	http.SetCookie(w, &http.Cookie{
		Name:  "JWT_token",
		Value: jwtString,
	})

	//return
	w.WriteHeader(http.StatusOK)
	return
}
