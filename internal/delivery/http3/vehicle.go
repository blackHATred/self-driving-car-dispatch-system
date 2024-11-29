package http3

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"self-driving-car-dispatch-system/internal/usecase"
	"sync"
	"time"
)

type VehicleDelivery struct {
	broadcastUsecase usecase.BroadcastUsecase
	logger           *logrus.Logger
	tlsConfig        *tls.Config
	quicConfig       *quic.Config

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	stop   struct {
		stop bool
		sync.RWMutex
	}
	bufferPool sync.Pool
}

func NewVehicleDelivery(
	broadcastUsecase usecase.BroadcastUsecase,
	logger *logrus.Logger,
	tlsConfig *tls.Config,
	quicConfig *quic.Config,
) *VehicleDelivery {
	ctx, cancel := context.WithCancel(context.Background())
	delivery := VehicleDelivery{
		broadcastUsecase: broadcastUsecase,
		logger:           logger,
		tlsConfig:        tlsConfig,
		quicConfig:       quicConfig,

		wg:     sync.WaitGroup{},
		ctx:    ctx,
		cancel: cancel,
		stop: struct {
			stop bool
			sync.RWMutex
		}{
			stop: false,
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				// Буфер для чтения данных из потока
				return make([]byte, 4096)
			},
		},
	}
	return &delivery
}

func (v *VehicleDelivery) Start(addr string) error {
	listener, err := quic.ListenAddr(addr, v.tlsConfig, v.quicConfig)
	if err != nil {
		return err
	}
	v.logger.Printf("QUIC сервер запущен на %s\n", addr)

	for !v.stop.stop {
		conn, err := listener.Accept(v.ctx)
		if err != nil {
			v.logger.Println("Ошибка при подключении: ", err)
			continue
		}
		go v.handleConnection(conn)
	}

	return nil
}

func (v *VehicleDelivery) Stop(timeout time.Duration) {
	v.stop.Lock()
	v.stop.stop = true
	v.stop.Unlock()
	go func() {
		time.Sleep(timeout)
		v.cancel()
	}()
	v.wg.Wait()
}

// HandleConnection обрабатывает входящие соединения для приёма с ТС
func (v *VehicleDelivery) handleConnection(conn quic.Connection) {
	v.logger.Infoln("Новое соединение от ТС:", conn.RemoteAddr())
	defer v.logger.Infoln("Соединение с ТС закрыто:", conn.RemoteAddr())
	v.wg.Add(1)
	defer v.wg.Done()

	// Открываем поток для получения информации о транспортном средстве
	infoStream, err := conn.AcceptStream(context.Background())
	if err != nil {
		v.logger.Errorf("Ошибка при открытии потока с ТС %s: %s\n", conn.RemoteAddr(), err)
		conn.CloseWithError(0, "Connection error")
		return
	}
	defer infoStream.Close()
	v.logger.Infof("Открыт infoStream с ТС %s\n", conn.RemoteAddr())

	// Открываем поток для получения видеотрансляции
	videoStream, err := conn.AcceptStream(context.Background())
	if err != nil {
		v.logger.Errorf("Ошибка при открытии видеопотока с ТС %s: %s\n", conn.RemoteAddr(), err)
		conn.CloseWithError(0, "Connection error")
		return
	}
	defer videoStream.Close()
	v.logger.Infof("Открыт videoStream с ТС %s\n", conn.RemoteAddr())

	data, err := conn.ReceiveDatagram(v.ctx)
	if err != nil {
		v.logger.Errorf("Ошибка при получении данных от ТС %s: %s\n", conn.RemoteAddr(), err)
		conn.CloseWithError(0, "Connection error")
		return
	}
	if len(data) < 4 {
		v.logger.Errorf("Неверный формат данных от ТС %s\n", conn.RemoteAddr())
		conn.CloseWithError(0, "Connection error")
	}

	// в первых четырех байтах содержится ID ТС, последующие до конца - ключ доступа в UTF-8
	vehicleID := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	secret := string(data[4:])
	v.logger.Infof("Получены данные о ТС %d\n", vehicleID)

	infoChan := make(chan []byte, 100)  // буферизированный канал для передачи информации о ТС
	videoChan := make(chan []byte, 100) // буферизированный канал для передачи видеопотока
	errChan := make(chan error)         // канал для передачи ошибок
	go v.getStream(infoStream, infoChan, errChan)
	go v.getStream(videoStream, videoChan, errChan)
	go v.broadcastUsecase.SendInfoStream(vehicleID, secret, infoChan, errChan)
	go v.broadcastUsecase.SendVideoStream(vehicleID, secret, videoChan, errChan)
	for {
		select {
		case err := <-errChan:
			if err != nil {
				v.logger.Errorf("Ошибка при трансляции данных от ТС %d: %s\n", vehicleID, err)
			}
			conn.CloseWithError(0, fmt.Sprintf("Connection error: %s", err))
			close(infoChan)
			close(videoChan)
			return
		case <-v.ctx.Done():
			conn.CloseWithError(0, "Connection closed")
			close(infoChan)
			close(videoChan)
			return
		}
	}
}

func (v *VehicleDelivery) getStream(quicStream quic.Stream, stream chan []byte, errChan chan error) {
	for {
		data := v.bufferPool.Get().([]byte)
		n, err := quicStream.Read(data)
		if err != nil {
			v.bufferPool.Put(data)
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				v.logger.Errorf("Ошибка при чтении данных с потока: %s\n", err)
				errChan <- err
			}
			return
		}
		if n != 0 {
			v.logger.Debugf("Получено %d байт от %d\n", n, quicStream.StreamID())
			stream <- data[:n]
		}
	}
}
