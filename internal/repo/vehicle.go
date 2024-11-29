package repo

import (
	"self-driving-car-dispatch-system/internal/entity"
)

type VehicleRepo interface {
	GetVehicle(id int) (*entity.Vehicle, error)
	AddVehicle(vehicle *entity.Vehicle) error
	DeleteVehicle(id int) error
}
