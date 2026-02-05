package service

import (
	"os"
)

type TranscriptionService struct {
	api string
}

func NewTranscriptionService(apiUrl string) *TranscriptionService {
	return &TranscriptionService{
		api: apiUrl,
	}
}

func (s *TranscriptionService) Send(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return "", nil
}
