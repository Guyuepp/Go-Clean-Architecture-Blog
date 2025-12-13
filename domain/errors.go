package domain

import "errors"

var (
	// ErrInternalServerError will throw if any the Internal Server Error happen
	ErrInternalServerError = errors.New("internal Server Error")
	// ErrNotFound will throw if the requested item is not exists
	ErrNotFound = errors.New("your requested Item is not found")
	// ErrConflict will throw if the current action already exists
	ErrConflict = errors.New("your Item already exist")
	// ErrBadParamInput will throw if the given request-body or params is not valid
	ErrBadParamInput = errors.New("given Param is not valid")
	// ErrUnauthorized will throw if the user is unauthorized to access the resource
	ErrUserAlreadyExists = errors.New("user with given username already exists")
	// ErrUnauthorized will throw if the user is unauthorized to access the resource
	ErrUnauthorized = errors.New("you are unauthorized to access this resource")
	// ErrUserNotFound will throw if the requested user is not exists
	ErrUserNotFound = errors.New("requested user is not found")
	// ErrBadParamInput will throw if the given request-body or params is not valid
	ErrInvalidCredentials = errors.New("invalid credentials")
)
