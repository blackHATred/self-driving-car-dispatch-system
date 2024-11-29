package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"log"
	"os/exec"
	"sync"
)

func main() {
	// Подключаемся к серверу
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Отключить проверку сертификатов
	}

	quicConfig := &quic.Config{
		EnableDatagrams: true,
	}

	conn, err := quic.DialAddr(context.Background(), "localhost:4243", tlsConfig, quicConfig)
	if err != nil {
		log.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.CloseWithError(0, "Connection closed")

	// Согласно протоколу, в первые четыре байта записываем ID транспортного средства, далее ID диспетчера, затем пароль
	data := []byte{0x00, 0x00, 0x00, 0x02}                 // ID = 2
	data = append(data, []byte{0x00, 0x00, 0x00, 0x02}...) // ID = 2
	data = append(data, []byte("example")...)              // Пароль: example
	log.Printf("Отправка информации о диспетчере: %v", data)
	err = conn.SendDatagram(data)
	if err != nil {
		log.Fatal(err)
	}

	infoStream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Fatalf("Не удалось открыть поток: %v", err)
	}
	defer infoStream.Close()
	log.Printf("Открыт infoStream с %s\n", conn.RemoteAddr())

	videoStream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Fatalf("Не удалось открыть видеопоток: %v", err)
	}
	defer videoStream.Close()
	log.Printf("Открыт videoStream с %s\n", conn.RemoteAddr())

	log.Printf("Информация о диспетчере отправлена.")

	wg := &sync.WaitGroup{}
	errChan := make(chan error)
	wg.Add(2)
	go getVideoStream(wg, videoStream, errChan)
	go getInfoStream(wg, infoStream, errChan)

	// Ожидаем ошибку
	go func() {
		for {
			select {
			case err := <-errChan:
				log.Fatal(err)
				return
			}
		}
	}()

	wg.Wait()
}

func getVideoStream(wg *sync.WaitGroup, videoStream quic.Stream, errChan chan error) {
	defer wg.Done()

	// Запускаем GStreamer
	cmd := exec.Command(
		"gst-launch-1.0",
		"-v", "fdsrc", // Получаем данные из pipe
		"!", "h264parse",
		"!", "avdec_h264",
		"!", "videoconvert",
		"!", "autovideosink",
	)

	// Подключаем stdout к потоку данных
	stdin, err := cmd.StdinPipe()
	if err != nil {
		errChan <- fmt.Errorf("ошибка создания stdin pipe для GStreamer: %w", err)
	}
	defer stdin.Close()

	// Запускаем GStreamer
	err = cmd.Start()
	if err != nil {
		errChan <- fmt.Errorf("ошибка запуска GStreamer: %w", err)
	}

	// Читаем данные из QUIC потока и отправляем в GStreamer
	buffer := make([]byte, 4096)
	for {
		n, err := videoStream.Read(buffer)
		if err != nil {
			errChan <- fmt.Errorf("ошибка чтения из видеопотока: %w", err)
			break
		}

		_, err = stdin.Write(buffer[:n])
		if err != nil {
			errChan <- fmt.Errorf("ошибка записи в GStreamer: %w", err)
			break
		}
	}

	// Ждем завершения GStreamer
	err = cmd.Wait()
	if err != nil {
		errChan <- fmt.Errorf("GStreamer завершился с ошибкой: %w", err)
	}
}

func getInfoStream(wg *sync.WaitGroup, infoStream quic.Stream, errChan chan error) {
	defer wg.Done()

	// Читаем информацию о транспортном средстве
	buffer := make([]byte, 4096)
	for {
		n, err := infoStream.Read(buffer)
		if err != nil {
			errChan <- fmt.Errorf("ошибка чтения из infoStream: %w", err)
			break
		}
		log.Printf("Получена информация о транспортном средстве: %v\n", string(buffer[:n]))
	}
}
