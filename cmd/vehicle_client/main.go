package main

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

func main() {
	// Настройка TLS для QUIC (самоподписанный сертификат для разработки)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	quicConfig := &quic.Config{
		EnableDatagrams: true,
	}

	// Устанавливаем QUIC-соединение с сервером
	conn, err := quic.DialAddr(context.Background(), "localhost:4242", tlsConfig, quicConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.CloseWithError(0, "Connection closed")

	// Открываем текстовый поток для передачи информации о транспортном средстве
	infoStream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer infoStream.Close()

	// Открываем видеопоток для передачи данных
	videoStream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer videoStream.Close()

	// Согласно протоколу, в первые четыре байта записываем ID транспортного средства, далее пароль
	data := []byte{0x00, 0x00, 0x00, 0x02}    // ID = 2
	data = append(data, []byte("example")...) // Пароль: example
	log.Printf("Отправка информации о транспортном средстве: %v", data)
	err = conn.SendDatagram(data)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Информация о транспортном средстве отправлена.")

	wg := &sync.WaitGroup{}
	errChan := make(chan error)
	wg.Add(2)
	go sendVideoStream(wg, videoStream, errChan)
	go sendInfoStream(wg, infoStream, errChan)

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

func sendVideoStream(wg *sync.WaitGroup, stream quic.Stream, errChan chan error) {
	defer wg.Done()
	// Используем FFmpeg для сжатия и отправки видеопотока через стандартный вывод
	cmd := exec.Command("ffmpeg",
		"-re",                     // Реальное время
		"-i", "assets/output.mp4", // Исходное видео
		"-c:v", "libx264", // Кодек H.264
		"-preset", "ultrafast", // Быстрое кодирование
		"-tune", "zerolatency", // Низкая задержка
		"-f", "h264", // Сырой вывод
		"pipe:1",
	)

	// Связываем stdout FFmpeg с потоком QUIC
	cmd.Stdout = stream
	// cmd.Stderr = stream

	fmt.Println("Отправка видеопотока на сервер через QUIC...")
	err := cmd.Run()
	if err != nil {
		errChan <- err
		return
	}
	fmt.Println("Отправка видеопотока остановлена")
}

func sendInfoStream(wg *sync.WaitGroup, infoStream quic.Stream, errChan chan error) {
	defer wg.Done()

	// открываем CSV-файл с данными о транспортном средстве
	file, err := os.Open("assets/driving_log.csv")
	if err != nil {
		errChan <- err
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	_, err = reader.Read() // Пропускаем заголовок
	if err != nil {
		errChan <- err
		return
	}

	// Создаём таймер для отправки данных с частотой 25 раз в секунду
	ticker := time.NewTicker(time.Second / 25)
	defer ticker.Stop()

	fmt.Println("Отправка текстового потока на сервер через QUIC...")

	for {
		select {
		case <-ticker.C:
			// Читаем следующую строчку из CSV-файла
			record, err := reader.Read()
			if err == io.EOF {
				fmt.Println("Отправка текстового потока остановлена")
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			// Конвертируем в JSON
			data := map[string]string{
				"steering": record[0],
				"throttle": record[1],
				"brake":    record[2],
				"speed":    record[3],
			}
			jsonData, err := json.Marshal(data)
			if err != nil {
				errChan <- err
				return
			}

			// Отправляем JSON на сервер
			_, err = infoStream.Write(jsonData)
			if err != nil {
				errChan <- err
				return
			}
		}
	}
}
