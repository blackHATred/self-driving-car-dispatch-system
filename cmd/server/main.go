package main

import (
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"self-driving-car-dispatch-system/config"
	"self-driving-car-dispatch-system/internal/delivery/http3"
	"self-driving-car-dispatch-system/internal/repo/redis"
	"self-driving-car-dispatch-system/internal/usecase/service"
	redisClient "self-driving-car-dispatch-system/pkg/redis"
	"syscall"
	"time"
)

var logger = logrus.New()

// Структура конфигурации сервера
var cfg config.ServerConfig

func main() {
	/*
		Конфигурация сервера ретрансляции
	*/
	logger.SetLevel(logrus.DebugLevel)
	// Название файла конфигурации сервера (расширение значения не имеет - viper работает с разными форматами)
	viper.SetConfigName("server")
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
	vehicleRepo := redis.NewVehicleRepo(rdsClient)
	dispatcherRepo := redis.NewDispatcherRepo(rdsClient)
	broadcastUsecase := service.NewBroadcastService(vehicleRepo, dispatcherRepo)

	certFile := "config/localhost.pem"
	keyFile := "config/localhost-key.pem"
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("Ошибка загрузки сертификатов:", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}
	quicConfig := &quic.Config{
		EnableDatagrams: true,
	}

	vehicleDelivery := http3.NewVehicleDelivery(broadcastUsecase, logger, tlsConfig, quicConfig)
	dispatcherDelivery := http3.NewDispatcherDelivery(broadcastUsecase, logger, tlsConfig, quicConfig)

	/*
		Запуск сервера
	*/
	go func() {
		if err := vehicleDelivery.Start(fmt.Sprintf("%s:%d", cfg.VehicleHost, cfg.VehiclePort)); err != nil {
			log.Fatalf("Ошибка запуска сервера ретрансляции для ТС: %s", err)
		}
	}()
	go func() {
		if err := dispatcherDelivery.Start(fmt.Sprintf("%s:%d", cfg.DispatcherHost, cfg.DispatcherPort)); err != nil {
			log.Fatalf("Ошибка запуска сервера ретрансляции для диспетчера: %s", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	// kill (без параметров) по умолчанию отправит syscall.SIGTERM
	// kill -2 отправит syscall.SIGINT
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Infof("Завершаем работу сервера...")
	// Даём серверу 5 секунд на завершение работы
	vehicleDelivery.Stop(5 * time.Second)
	dispatcherDelivery.Stop(5 * time.Second)
	logger.Infoln("Сервер остановил свою работу")
}
