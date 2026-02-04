package service

import (
	"os"
)

type TranscriptionService struct {
	api string
}

func NewTranscriptionService(apuUrl string) *TranscriptionService {
	return &TranscriptionService{
		api: apuUrl,
	}
}

func (s *TranscriptionService) Send(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// Нужен короче тут эндпоинт сервиса куда отправлять запрос пока заглушка ниже
	return "", nil
}

func (s *TranscriptionService) SendTest(filepath string) (string, error) {
	return "test tranc", nil
}
