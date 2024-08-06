package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"yandex_gophermart/pkg/entities"
	g_errors "yandex_gophermart/pkg/errors"
	"yandex_gophermart/pkg/security"
)

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	//getting user data
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Errorf("error while reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	uData := entities.UserData{}
	err = json.Unmarshal(bodyBytes, &uData)
	if err != nil {
		h.Logger.Errorf("unmarshal err: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//creating user
	passwordSalt, err := security.GenPasswordSalt()
	if err != nil {
		h.Logger.Errorf("cant generate a password salt: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	uID, err := h.Storage.SaveUser(uData.Login, security.HashPassword(uData.Password, passwordSalt), passwordSalt, r.Context())
	if errors.Is(err, g_errors.MakeErrUserAlreadyExists()) {
		h.Logger.Warnf("user create error: %v", err.Error())
		w.WriteHeader(http.StatusConflict)
		_, err := w.Write([]byte("this user already exists"))
		if err != nil {
			h.Logger.Errorf("response write err: %v", err.Error())
			return
		}
		return
	} else if err != nil {
		h.Logger.Errorf("cant save user in db: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//creating and setting jwt token
	jwtString, err := h.JWTH.BuildNewJWTString(uID)
	if err != nil {
		h.Logger.Errorf("jwt err: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  security.JWTCookieName,
		Value: jwtString,
	})

	//return
	w.WriteHeader(http.StatusOK)
}
