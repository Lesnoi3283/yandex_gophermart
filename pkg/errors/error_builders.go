package gophermart_errors

import "errors"

// Database errors

var errUserAlreadyExists error = errors.New("this user already exists")

func MakeErrUserAlreadyExists() error {
	return errUserAlreadyExists
}

var errUserNotFound error = errors.New("user wasn`t found")

func MakeErrUserNotFound() error {
	return errUserNotFound
}

//Security errors

var errJWTTokenIsNotValid = errors.New("jwt token is not valid")

func MakeJWTTokenIsNotValid() error {
	return errJWTTokenIsNotValid
}
