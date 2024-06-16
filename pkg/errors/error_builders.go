package gophermart_errors

//"g" means "gophermart"

import "errors"

var errUserAlreadyExists error = errors.New("this user already exists")

func MakeErrUserAlreadyExists() error {
	return errUserAlreadyExists
}
