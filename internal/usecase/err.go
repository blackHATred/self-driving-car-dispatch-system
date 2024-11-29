package usecase

import "errors"

var (
	ErrAccessDenied            = errors.New("access denied")
	ErrDispatcherNotFound      = errors.New("dispatcher not found")
	ErrVehicleNotFound         = errors.New("vehicle not found")
	ErrDispatcherAlreadyExists = errors.New("dispatcher already exists")
	ErrVehicleAlreadyExists    = errors.New("vehicle already exists")
	ErrInternal                = errors.New("internal error")
	ErrBadRequest              = errors.New("bad request")
	ErrNotFound                = errors.New("not found")
)
