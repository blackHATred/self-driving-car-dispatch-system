package usecase

import (
	"self-driving-car-dispatch-system/internal/entity"
)

type AdminUsecase interface {
	GetDispatcher(secret string, id int) (*entity.GetDispatcherResponse, error)
	AddDispatcher(secret string, dispatcher *entity.AddDispatcherRequest) (*entity.Dispatcher, error)
	EditDispatcher(secret string, dispatcher *entity.EditDispatcherRequest) error
	DeleteDispatcher(secret string, id int) error

	GetVehicle(secret string, id int) (*entity.GetVehicleResponse, error)
	AddVehicle(secret string, dispatcher *entity.AddVehicleRequest) (*entity.Vehicle, error)
	DeleteVehicle(secret string, id int) error
}
