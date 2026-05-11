package whisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Client вызывает Python-сервис Whisper (POST /transcribe).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

type transcribeResponse struct {
	Text    string `json:"text"`
	Success bool   `json:"success"`
}

// Transcribe отправляет аудио в сервис и возвращает распознанный текст.
func (c *Client) Transcribe(ctx context.Context, fileName, mimeType string, audio []byte) (string, error) {
	if c == nil || c.baseURL == "" {
		return "", fmt.Errorf("whisper client is not configured")
	}
	if len(audio) == 0 {
		return "", fmt.Errorf("empty audio payload")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	url := c.baseURL + "/transcribe"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = mimeType

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("whisper HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed transcribeResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("whisper JSON: %w", err)
	}
	text := strings.TrimSpace(parsed.Text)
	if text == "" {
		return "", fmt.Errorf("whisper returned empty text")
	}
	return text, nil
}
