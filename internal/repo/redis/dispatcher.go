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

type DispatcherRepo struct {
	redisClient *redis.Client
}

func NewDispatcherRepo(client *redis.Client) repo.DispatcherRepo {
	return &DispatcherRepo{
		redisClient: client,
	}
}

// setDispatcher используется как вспомогательный метод для сохранения диспетчера в redis.
// Так как redis не поддерживает операции вроде create / update, то публичные функции AddDispatcher и EditDispatcher
// будут использовать этот метод для сохранения или обновления диспетчера
func (d DispatcherRepo) setDispatcher(ctx context.Context, client redis.Cmdable, dispatcher *entity.Dispatcher) error {
	// сериализуем объект диспетчера в байты
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(*dispatcher)
	if err != nil {
		return err
	}

	// сохраняем сериализованный объект в Redis
	return client.Set(ctx, fmt.Sprintf("dispatcher:%d", dispatcher.ID), buffer.Bytes(), 0).Err()
}

func (d DispatcherRepo) isDispatcherExists(ctx context.Context, client redis.Cmdable, id int) (bool, error) {
	n, err := client.Exists(ctx, fmt.Sprintf("dispatcher:%d", id)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (d DispatcherRepo) GetDispatcher(id int) (*entity.Dispatcher, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// получаем сериализованный объект из Redis
	data, err := d.redisClient.Get(ctx, fmt.Sprintf("dispatcher:%d", id)).Bytes()
	switch {
	case errors.Is(err, redis.Nil):
		return nil, repo.ErrDispatcherNotFound
	case err != nil:
		return nil, errors.Join(repo.ErrInternal, err)
	}

	// десериализуем объект
	var dispatcher entity.Dispatcher
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err = decoder.Decode(&dispatcher)
	if err != nil {
		return nil, err
	}
	return &dispatcher, nil
}

func (d DispatcherRepo) AddDispatcher(dispatcher *entity.Dispatcher) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := d.redisClient.Watch(ctx, func(tx *redis.Tx) error {
		// получаем следующий ID для диспетчера
		id, err := d.redisClient.Incr(ctx, "dispatcher:id").Result()
		if err != nil {
			return err
		}
		dispatcher.ID = int(id)

		// проверяем, существует ли уже диспетчер на случай непредвиденной ошибки,
		// потому что redis не поддерживает уникальные ключи
		exists, err := d.isDispatcherExists(ctx, tx, dispatcher.ID)
		if err != nil {
			return err
		}
		if exists {
			return repo.ErrDispatcherAlreadyExists
		}

		// начинаем транзакцию
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			// используем вспомогательную функцию для сохранения диспетчера
			return d.setDispatcher(ctx, pipe, dispatcher)
		})
		return err
	}, fmt.Sprintf("dispatcher:%d", dispatcher.ID))

	if err != nil {
		return errors.Join(repo.ErrInternal, err)
	}
	return nil
}

func (d DispatcherRepo) EditDispatcher(dispatcher *entity.Dispatcher) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := d.redisClient.Watch(ctx, func(tx *redis.Tx) error {
		// проверяем, существует ли диспетчер
		exists, err := d.isDispatcherExists(ctx, tx, dispatcher.ID)
		if err != nil {
			return err
		}
		if !exists {
			return repo.ErrDispatcherNotFound
		}

		// начинаем транзакцию
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return d.setDispatcher(ctx, pipe, dispatcher)
		})
		return err
	}, fmt.Sprintf("dispatcher:%d", dispatcher.ID))

	if err != nil {
		return errors.Join(repo.ErrInternal, err)
	}
	return nil
}

func (d DispatcherRepo) DeleteDispatcher(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := d.redisClient.Del(ctx, fmt.Sprintf("dispatcher:%d", id)).Result()
	if err != nil {
		return errors.Join(repo.ErrInternal, err)
	}
	return nil
}
