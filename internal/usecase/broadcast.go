package usecase

type BroadcastUsecase interface {
	// GetVideoStream передает видеопоток с камеры ТС в поток передачи диспетчеру
	GetVideoStream(vehicleID, dispatcherID int, dispatcherPassword string, stream chan []byte, errChan chan error)
	// GetInfoStream передает информацию из потока ТС в поток передачи диспетчеру
	GetInfoStream(vehicleID, dispatcherID int, dispatcherPassword string, stream chan []byte, errChan chan error)
	// SendVideoStream отправляет видеопоток с камеры ТС в канал
	SendVideoStream(vehicleID int, vehiclePassword string, stream chan []byte, errChan chan error)
	// SendInfoStream отправляет информационный поток с ТС в канал
	SendInfoStream(vehicleID int, vehiclePassword string, stream chan []byte, errChan chan error)
}
