package service

import (
	"errors"
	"fmt"
	"self-driving-car-dispatch-system/internal/entity"
	"self-driving-car-dispatch-system/internal/repo"
	"self-driving-car-dispatch-system/internal/usecase"
	"self-driving-car-dispatch-system/pkg/password"
)

type AdminService struct {
	vehicleRepo    repo.VehicleRepo
	dispatcherRepo repo.DispatcherRepo
	secretKey      string
}

func NewAdminService(vehicleRepo repo.VehicleRepo, dispatcherRepo repo.DispatcherRepo, secret string) usecase.AdminUsecase {
	return &AdminService{
		vehicleRepo:    vehicleRepo,
		dispatcherRepo: dispatcherRepo,
		secretKey:      secret,
	}
}

func (a AdminService) GetDispatcher(secret string, id int) (*entity.GetDispatcherResponse, error) {
	if secret != a.secretKey {
		return nil, usecase.ErrAccessDenied
	}
	dispatcher, err := a.dispatcherRepo.GetDispatcher(id)
	switch {
	case err == nil:
		return &entity.GetDispatcherResponse{
			ID:         dispatcher.ID,
			GrantsType: dispatcher.GrantsType,
			Grants:     dispatcher.Grants,
		}, nil
	case errors.Is(err, repo.ErrDispatcherNotFound):
		return nil, usecase.ErrDispatcherNotFound
	default:
		return nil, errors.Join(usecase.ErrInternal, err)
	}
}

func (a AdminService) AddDispatcher(secret string, dispatcherRequest *entity.AddDispatcherRequest) (*entity.Dispatcher, error) {
	if secret != a.secretKey {
		return nil, usecase.ErrAccessDenied
	}
	passwordHash, err := password.HashPassword(dispatcherRequest.Password)
	if err != nil {
		return nil, errors.Join(usecase.ErrBadRequest, fmt.Errorf("failed to hash password: %w", err))
	}
	if !entity.IsGrantsTypeValid(dispatcherRequest.GrantsType) {
		return nil, errors.Join(usecase.ErrBadRequest, fmt.Errorf("invalid dispatcher grants type"))
	}
	dispatcher := &entity.Dispatcher{
		PasswordHash: passwordHash,
	}
	switch dispatcherRequest.GrantsType {
	case entity.AllGrants:
		dispatcher.GrantsType = entity.AllGrants
		dispatcher.Grants = make([]int, 0)
	case entity.ListGrants:
		dispatcher.GrantsType = entity.ListGrants
		dispatcher.Grants = dispatcherRequest.Grants
	default:
		return nil, errors.Join(usecase.ErrBadRequest, fmt.Errorf("invalid dispatcher grants"))
	}
	err = a.dispatcherRepo.AddDispatcher(dispatcher)
	switch {
	case err == nil:
		return dispatcher, nil
	case errors.Is(err, repo.ErrDispatcherAlreadyExists):
		return nil, usecase.ErrDispatcherAlreadyExists
	default:
		return nil, errors.Join(usecase.ErrInternal, err)
	}
}

func (a AdminService) EditDispatcher(secret string, dispatcherRequest *entity.EditDispatcherRequest) error {
	if secret != a.secretKey {
		return usecase.ErrAccessDenied
	}
	if !entity.IsGrantsTypeValid(dispatcherRequest.GrantsType) {
		return errors.Join(usecase.ErrBadRequest, fmt.Errorf("invalid dispatcher grants type"))
	}
	// получаем текущий объект диспетчера
	dispatcher, err := a.dispatcherRepo.GetDispatcher(dispatcherRequest.ID)
	switch {
	case err == nil:
	case errors.Is(err, repo.ErrDispatcherNotFound):
		return usecase.ErrDispatcherNotFound
	default:
		return errors.Join(usecase.ErrInternal, err)
	}
	// обновляем объект диспетчера
	dispatcher.GrantsType = dispatcherRequest.GrantsType
	if dispatcherRequest.GrantsType == entity.AllGrants {
		// если тип разрешений - все, то список разрешений не нужен
		dispatcher.Grants = make([]int, 0)
	} else if dispatcherRequest.GrantsType == entity.ListGrants && dispatcherRequest.Grants != nil {
		dispatcher.Grants = dispatcherRequest.Grants
	} else {
		return errors.Join(usecase.ErrBadRequest, fmt.Errorf("invalid dispatcher grants"))
	}
	err = a.dispatcherRepo.EditDispatcher(dispatcher)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrDispatcherNotFound):
		return usecase.ErrDispatcherNotFound
	default:
		return errors.Join(usecase.ErrInternal, err)
	}
}

func (a AdminService) DeleteDispatcher(secret string, id int) error {
	if secret != a.secretKey {
		return usecase.ErrAccessDenied
	}
	err := a.dispatcherRepo.DeleteDispatcher(id)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrDispatcherNotFound):
		return usecase.ErrDispatcherNotFound
	default:
		return errors.Join(usecase.ErrInternal, err)
	}
}

func (a AdminService) GetVehicle(secret string, id int) (*entity.GetVehicleResponse, error) {
	if secret != a.secretKey {
		return nil, usecase.ErrAccessDenied
	}
	vehicle, err := a.vehicleRepo.GetVehicle(id)
	switch {
	case err == nil:
		return &entity.GetVehicleResponse{
			ID: vehicle.ID,
		}, nil
	case errors.Is(err, repo.ErrVehicleNotFound):
		return nil, usecase.ErrVehicleNotFound
	default:
		return nil, errors.Join(usecase.ErrInternal, err)
	}
}

func (a AdminService) AddVehicle(secret string, vehicleRequest *entity.AddVehicleRequest) (*entity.Vehicle, error) {
	if secret != a.secretKey {
		return nil, usecase.ErrAccessDenied
	}
	passwordHash, err := password.HashPassword(vehicleRequest.Password)
	if err != nil {
		return nil, errors.Join(usecase.ErrBadRequest, fmt.Errorf("failed to hash password: %w", err))
	}
	vehicle := &entity.Vehicle{
		PasswordHash: passwordHash,
	}
	err = a.vehicleRepo.AddVehicle(vehicle)
	switch {
	case err == nil:
		return vehicle, nil
	case errors.Is(err, repo.ErrVehicleAlreadyExists):
		return nil, usecase.ErrVehicleAlreadyExists
	default:
		return nil, errors.Join(usecase.ErrInternal, err)
	}
}

func (a AdminService) DeleteVehicle(secret string, id int) error {
	if secret != a.secretKey {
		return usecase.ErrAccessDenied
	}
	err := a.vehicleRepo.DeleteVehicle(id)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrVehicleNotFound):
		return usecase.ErrVehicleNotFound
	default:
		return errors.Join(usecase.ErrInternal, err)
	}
}
