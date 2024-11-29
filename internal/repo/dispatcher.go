package repo

import (
	"self-driving-car-dispatch-system/internal/entity"
)

type DispatcherRepo interface {
	GetDispatcher(id int) (*entity.Dispatcher, error)
	AddDispatcher(dispatcher *entity.Dispatcher) error
	EditDispatcher(dispatcher *entity.Dispatcher) error
	DeleteDispatcher(id int) error
}
