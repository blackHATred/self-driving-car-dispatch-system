package http3

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/sirupsen/logrus"
	"self-driving-car-dispatch-system/internal/usecase"
	"sync"
	"time"
)

type DispatcherDelivery struct {
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

func NewDispatcherDelivery(
	broadcastUsecase usecase.BroadcastUsecase,
	logger *logrus.Logger,
	tlsConfig *tls.Config,
	quicConfig *quic.Config,
) *DispatcherDelivery {
	ctx, cancel := context.WithCancel(context.Background())
	delivery := DispatcherDelivery{
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

func (v *DispatcherDelivery) Start(addr string) error {
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

func (v *DispatcherDelivery) Stop(timeout time.Duration) {
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
func (v *DispatcherDelivery) handleConnection(conn quic.Connection) {
	v.logger.Infoln("Новое соединение от диспетчера:", conn.RemoteAddr())
	defer v.logger.Infoln("Соединение с диспетчером закрыто:", conn.RemoteAddr())
	v.wg.Add(1)
	defer v.wg.Done()

	// Открываем поток для отправки информации о транспортном средстве диспетчеру
	infoStream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		v.logger.Errorf("Ошибка при открытии потока с диспетчером %s: %s\n", conn.RemoteAddr(), err)
		conn.CloseWithError(0, "Connection error")
		return
	}
	defer infoStream.Close()
	v.logger.Infof("Открыт infoStream с диспетчером %s\n", conn.RemoteAddr())

	// Открываем поток для отправки видеотрансляции диспетчеру
	videoStream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		v.logger.Errorf("Ошибка при открытии видеопотока с диспетчером %s: %s\n", conn.RemoteAddr(), err)
		conn.CloseWithError(0, "Connection error")
		return
	}
	defer videoStream.Close()
	v.logger.Infof("Открыт videoStream с диспетчером %s\n", conn.RemoteAddr())

	data, err := conn.ReceiveDatagram(v.ctx)
	if err != nil {
		v.logger.Errorf("Ошибка при получении данных от диспетчера %s: %s\n", conn.RemoteAddr(), err)
		conn.CloseWithError(0, "Connection error")
		return
	}
	if len(data) < 8 {
		v.logger.Errorf("Некорректные данные от диспетчера %s\n", conn.RemoteAddr())
	}

	// в первых четырех байтах содержится ID ТС, с которого хотим получать данные
	vehicleID := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	// в последующих четырех байтах содержится ID диспетчера, который хочет получать данные
	dispatcherID := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
	// последующие до n - ключ доступа в UTF-8
	secret := string(data[8:])
	v.logger.Infof("Получены данные о диспетчере %d %s\n", dispatcherID, secret)

	infoChan := make(chan []byte, 100)  // буферизированный канал для передачи информации о ТС
	videoChan := make(chan []byte, 100) // буферизированный канал для передачи видеопотока
	errChan := make(chan error)         // канал для передачи ошибок
	v.logger.Infof("Отправка информации о ТС %d диспетчеру %d\n", vehicleID, dispatcherID)
	go v.sendStream(infoStream, infoChan)
	go v.sendStream(videoStream, videoChan)
	go v.broadcastUsecase.GetInfoStream(vehicleID, dispatcherID, secret, infoChan, errChan)
	go v.broadcastUsecase.GetVideoStream(vehicleID, dispatcherID, secret, videoChan, errChan)
	for {
		select {
		case err := <-errChan:
			if err != nil {
				v.logger.Errorf("Ошибка при трансляции данных диспетчеру %d: %s\n", vehicleID, err)
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

func (v *DispatcherDelivery) sendStream(quicStream quic.Stream, stream chan []byte) {
	for {
		select {
		case data := <-stream:
			n, err := quicStream.Write(data)
			if err != nil {
				v.logger.Errorf("Ошибка при отправке данных в поток: %s\n", err)
				return
			}
			v.logger.Debugf("Отправлено %d байт в %d\n", n, quicStream.StreamID())
		case <-v.ctx.Done():
			return
		}
	}
}
