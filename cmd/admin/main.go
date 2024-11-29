package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	"self-driving-car-dispatch-system/config"
	"self-driving-car-dispatch-system/internal/delivery/http1"
	"self-driving-car-dispatch-system/internal/repo/redis"
	"self-driving-car-dispatch-system/internal/usecase/service"
	redisClient "self-driving-car-dispatch-system/pkg/redis"
	"syscall"
	"time"
)

// По умолчанию все логи будут писаться в stdout
var log = logrus.New()

// Структура конфигурации сервера
var cfg config.AdminConfig

func main() {
	/*
		Конфигурация сервера для администратора
	*/
	// Название файла конфигурации сервера (расширение значения не имеет - viper работает с разными форматами)
	viper.SetConfigName("admin")
	// Добавляем директории, в которых будем искать файл конфигурации по приоритету:
	viper.AddConfigPath("./config")         // Папка с конфигурацией
	viper.AddConfigPath(".")                // Корень проекта
	viper.AddConfigPath("./config/example") // Если не нашли актуальную конфигурацию, то читаем пример конфигурации
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Ошибка чтения файла конфигурации: %s", err)
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Ошибка чтения файла конфигурации: %s", err)
	}
	cfg.SecretKey = os.Getenv("SECRET_KEY")
	/*
		Подключение к redis
	*/
	rdsClient, err := redisClient.GetRedisClient(cfg.DatabaseUrl, cfg.DatabaseNumber)
	if err != nil {
		log.Fatalf("Ошибка подключения к redis: %s", err)
	}
	/*
		Инициализация репозиториев, сервисов и обработчиков
	*/
	dispatcherRepo := redis.NewDispatcherRepo(rdsClient)
	vehicleRepo := redis.NewVehicleRepo(rdsClient)
	adminUsecase := service.NewAdminService(vehicleRepo, dispatcherRepo, cfg.SecretKey)
	adminDelivery := http1.NewAdminDelivery(log, adminUsecase)
	/*
		Запуск сервера
	*/
	// Создаём новый сервер
	server := gin.New()
	adminRouter := server.Group("/admin")
	adminDelivery.Configure(adminRouter)
	log.Infof("Запуск сервера по адресу %s...", cfg.Addr)
	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: server.Handler(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Сервер прекратил работу по причине: %s", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	// kill (без параметров) по умолчанию отправит syscall.SIGTERM
	// kill -2 отправит syscall.SIGINT
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Infof("Завершаем работу сервера...")
	// Даём серверу 5 секунд на завершение работы
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Сервер прекратил работу по причине: %s", err)
	}
	select {
	case <-ctx.Done():
		log.Infoln("Превышено время ожидания завершения работы сервера, принудительное завершение...")
	}
	log.Infoln("Сервер остановил свою работу")
}
