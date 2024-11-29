package repo

import "errors"

var (
	ErrDispatcherNotFound      = errors.New("dispatcher not found")
	ErrDispatcherAlreadyExists = errors.New("dispatcher already exists")
	ErrInternal                = errors.New("db error")
	ErrVehicleNotFound         = errors.New("vehicle not found")
	ErrVehicleAlreadyExists    = errors.New("vehicle already exists")
)
