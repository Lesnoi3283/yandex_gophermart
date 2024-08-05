package gophermarterrors

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

var errThisOrderWasUploadedByDifferentUser error = errors.New("this order was uploaded by different user")

func MakeErrThisOrderWasUploadedByDifferentUser() error {
	return errThisOrderWasUploadedByDifferentUser
}

var errUserHasAlreadyUploadedThisOrder error = errors.New("user has already uploaded this order")

func MakeErrUserHasAlreadyUploadedThisOrder() error {
	return errUserHasAlreadyUploadedThisOrder
}

var errOrderNotFound error = errors.New("order was not found")

func MakeErrOrderNotFound() error {
	return errOrderNotFound
}

//security errors

var errJWTTokenIsNotValid = errors.New("jwt token is not valid")

func MakeErrJWTTokenIsNotValid() error {
	return errJWTTokenIsNotValid
}

var errWrongLoginOrPassword error = errors.New("Incorrect login or password")

func MakeErrWrongLoginOrPassword() error {
	return errWrongLoginOrPassword
}

//business errors

var errNotEnoughPoints error = errors.New("not enough points")

func MakeErrNotEnoughPoints() error {
	return errNotEnoughPoints
}
