package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
	"self-driving-car-dispatch-system/internal/entity"
	"self-driving-car-dispatch-system/internal/repo"
	"self-driving-car-dispatch-system/internal/usecase"
	"self-driving-car-dispatch-system/pkg/password"
	"sync"
)

// MaxJsonSize ограничивает размер JSON в приложении в 8 КБ
const MaxJsonSize = 1 << 13

type BroadcastService struct {
	vehicleRepo    repo.VehicleRepo
	dispatcherRepo repo.DispatcherRepo
	videoStreams   sync.Map
	infoStreams    sync.Map
	pool           sync.Pool
}

func NewBroadcastService(vehicleRepo repo.VehicleRepo, dispatcherRepo repo.DispatcherRepo) usecase.BroadcastUsecase {
	service := &BroadcastService{
		vehicleRepo:    vehicleRepo,
		dispatcherRepo: dispatcherRepo,
		videoStreams:   sync.Map{},
		infoStreams:    sync.Map{},
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096)
			},
		},
	}
	return service
}

func (b *BroadcastService) GetVideoStream(vehicleID, dispatcherID int, dispatcherPassword string, stream chan []byte, errChan chan error) {
	dispatcher, err := b.dispatcherRepo.GetDispatcher(dispatcherID)
	switch {
	case errors.Is(err, repo.ErrDispatcherNotFound):
		errChan <- usecase.ErrDispatcherNotFound
		return
	case errors.Is(err, repo.ErrInternal):
		errChan <- usecase.ErrInternal
		return
	}
	if !password.CheckPassword(dispatcherPassword, dispatcher.PasswordHash) {
		errChan <- errors.Join(usecase.ErrAccessDenied, fmt.Errorf("неверный пароль"))
		return
	}
	// проверяем, что у диспетчера есть доступ ко всем ТС или к данному ТС
	if !(dispatcher.GrantsType == entity.AllGrants || (dispatcher.GrantsType == entity.ListGrants && slices.Contains(dispatcher.Grants, vehicleID))) {
		fmt.Println(dispatcher)
		errChan <- usecase.ErrAccessDenied
		return
	}
	ch, ok := b.videoStreams.Load(vehicleID)
	if !ok {
		errChan <- usecase.ErrNotFound
		return
	}
	c := ch.(chan []byte)
	// передаём видеопоток ch в поток stream и если канал ch закрывается, то выходим
	for d := range c {
		stream <- d
	}
}

func (b *BroadcastService) GetInfoStream(vehicleID, dispatcherID int, dispatcherPassword string, stream chan []byte, errChan chan error) {
	dispatcher, err := b.dispatcherRepo.GetDispatcher(dispatcherID)
	switch {
	case errors.Is(err, repo.ErrDispatcherNotFound):
		errChan <- usecase.ErrDispatcherNotFound
		return
	case errors.Is(err, repo.ErrInternal):
		errChan <- usecase.ErrInternal
		return
	}
	if !password.CheckPassword(dispatcherPassword, dispatcher.PasswordHash) {
		errChan <- errors.Join(usecase.ErrAccessDenied, fmt.Errorf("неверный пароль"))
		return
	}
	// проверяем, что у диспетчера есть доступ ко всем ТС или к данному ТС
	if !(dispatcher.GrantsType == entity.AllGrants || (dispatcher.GrantsType == entity.ListGrants && slices.Contains(dispatcher.Grants, vehicleID))) {
		fmt.Println(dispatcher)
		errChan <- usecase.ErrAccessDenied
		return
	}
	ch, ok := b.infoStreams.Load(vehicleID)
	if !ok {
		errChan <- usecase.ErrNotFound
		return
	}
	c := ch.(chan []byte)
	for d := range c {
		stream <- d
	}
}

func (b *BroadcastService) SendVideoStream(vehicleID int, vehiclePassword string, stream chan []byte, errChan chan error) {
	vehicle, err := b.vehicleRepo.GetVehicle(vehicleID)
	switch {
	case errors.Is(err, repo.ErrVehicleNotFound):
		errChan <- usecase.ErrVehicleNotFound
		return
	case errors.Is(err, repo.ErrInternal):
		errChan <- usecase.ErrInternal
		return
	}
	if !password.CheckPassword(vehiclePassword, vehicle.PasswordHash) {
		errChan <- errors.Join(usecase.ErrBadRequest, fmt.Errorf("неверный пароль"))
		return
	}
	// если трансляция уже ведется, то удаляем её и начинаем новую
	// если трансляции нет, то создаем новую
	ch, loaded := b.videoStreams.LoadOrStore(vehicleID, make(chan []byte, 100))
	if loaded {
		ch = make(chan []byte, 100)
		b.videoStreams.Store(vehicleID, ch)
	}

	for d := range stream {
		select {
		case ch.(chan []byte) <- d:
		default:
			<-ch.(chan []byte)
			ch.(chan []byte) <- d
		}
	}
	close(ch.(chan []byte))
	b.videoStreams.Delete(vehicleID)
}

func (b *BroadcastService) SendInfoStream(vehicleID int, vehiclePassword string, stream chan []byte, errChan chan error) {
	vehicle, err := b.vehicleRepo.GetVehicle(vehicleID)
	switch {
	case errors.Is(err, repo.ErrVehicleNotFound):
		errChan <- usecase.ErrVehicleNotFound
		return
	case errors.Is(err, repo.ErrInternal):
		errChan <- usecase.ErrInternal
		return
	}
	if !password.CheckPassword(vehiclePassword, vehicle.PasswordHash) {
		errChan <- errors.Join(usecase.ErrBadRequest, fmt.Errorf("неверный пароль"))
		return
	}

	// если трансляция уже ведется, то удаляем её и начинаем новую
	// если трансляции нет, то создаем новую
	ch, loaded := b.infoStreams.LoadOrStore(vehicleID, make(chan []byte, 100))
	if loaded {
		ch = make(chan []byte, 100)
		b.infoStreams.Store(vehicleID, ch)
	}
	defer close(ch.(chan []byte))
	defer b.infoStreams.Delete(vehicleID)
	// Читаем информацию батчами, пока не сможем распарсить JSON. Полученный JSON отправляем в stream
	buffer := b.pool.Get().([]byte)[:0]
	defer b.pool.Put(buffer[:cap(buffer)])
	for d := range stream {
		buffer = append(buffer, d...)
		var jsonData map[string]interface{}
		if err = json.Unmarshal(buffer, &jsonData); err == nil {
			select {
			case ch.(chan []byte) <- buffer:
				// пакет успешно передан
			default:
				// канал заполнен, удаляем старые данные и записываем новые (закольцовываем буфер)
				<-ch.(chan []byte)
				ch.(chan []byte) <- buffer
			}
		}
		if len(buffer) > MaxJsonSize {
			errChan <- usecase.ErrBadRequest
			return
		}
		buffer = buffer[:0]
	}
}
