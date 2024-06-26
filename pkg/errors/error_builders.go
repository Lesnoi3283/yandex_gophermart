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

//security errors

var errJWTTokenIsNotValid = errors.New("jwt token is not valid")

func MakeJWTTokenIsNotValid() error {
	return errJWTTokenIsNotValid
}
