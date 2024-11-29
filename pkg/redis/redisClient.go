package redisClient

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

func GetRedisClient(addr string, db int) (*redis.Client, error) {
	// создаём клиент для подключения к redis
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})
	// задаём таймаут на подключение
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// отправляем ping для проверки соединения
	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}
	return client, nil
}
