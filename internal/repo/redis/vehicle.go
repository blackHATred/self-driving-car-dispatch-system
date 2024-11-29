package redis

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"self-driving-car-dispatch-system/internal/entity"
	"self-driving-car-dispatch-system/internal/repo"
	"time"
)

type VehicleRepo struct {
	redisClient *redis.Client
}

func NewVehicleRepo(client *redis.Client) repo.VehicleRepo {
	return &VehicleRepo{
		redisClient: client,
	}
}

// setVehicle используется как вспомогательный метод для сохранения ТС в redis.
// Так как redis не поддерживает операции вроде create / update, то публичные функции AddVehicle и EditVehicle
// будут использовать этот метод для сохранения или обновления ТС
func (d VehicleRepo) setVehicle(ctx context.Context, client redis.Cmdable, vehicle *entity.Vehicle) error {
	// сериализуем объект диспетчера в байты
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(*vehicle)
	if err != nil {
		return err
	}

	// сохраняем сериализованный объект в Redis
	return client.Set(ctx, fmt.Sprintf("vehicle:%d", vehicle.ID), buffer.Bytes(), 0).Err()
}

func (d VehicleRepo) isVehicleExists(ctx context.Context, client redis.Cmdable, id int) (bool, error) {
	n, err := client.Exists(ctx, fmt.Sprintf("vehicle:%d", id)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (d VehicleRepo) GetVehicle(id int) (*entity.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// получаем сериализованный объект из Redis
	data, err := d.redisClient.Get(ctx, fmt.Sprintf("vehicle:%d", id)).Bytes()
	switch {
	case errors.Is(err, redis.Nil):
		return nil, repo.ErrVehicleNotFound
	case err != nil:
		return nil, errors.Join(repo.ErrInternal, err)
	}

	// десериализуем объект
	var vehicle entity.Vehicle
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err = decoder.Decode(&vehicle)
	if err != nil {
		return nil, err
	}
	return &vehicle, nil
}

func (d VehicleRepo) AddVehicle(vehicle *entity.Vehicle) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := d.redisClient.Watch(ctx, func(tx *redis.Tx) error {
		// получаем следующий ID для ТС
		id, err := d.redisClient.Incr(ctx, "vehicle:id").Result()
		if err != nil {
			return err
		}
		vehicle.ID = int(id)

		// проверяем, существует ли уже диспетчер на случай непредвиденной ошибки,
		// потому что redis не поддерживает уникальные ключи
		exists, err := d.isVehicleExists(ctx, tx, vehicle.ID)
		if err != nil {
			return err
		}
		if exists {
			return repo.ErrVehicleAlreadyExists
		}

		// начинаем транзакцию
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			// используем вспомогательную функцию для сохранения диспетчера
			return d.setVehicle(ctx, pipe, vehicle)
		})
		return err
	}, fmt.Sprintf("vehicle:%d", vehicle.ID))

	if err != nil {
		return errors.Join(repo.ErrInternal, err)
	}
	return nil
}

func (d VehicleRepo) DeleteVehicle(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := d.redisClient.Del(ctx, fmt.Sprintf("vehicle:%d", id)).Result()
	if err != nil {
		return errors.Join(repo.ErrInternal, err)
	}
	return nil
}
