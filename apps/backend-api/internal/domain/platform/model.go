package platform

import "time"

type AppSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	IsSecret    bool      `json:"isSecret"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type StoredFile struct {
	ID         string    `json:"id"`
	Namespace  string    `json:"namespace"`
	StorageKey string    `json:"storageKey"`
	FileName   string    `json:"fileName"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int64     `json:"sizeBytes"`
	SHA256Hex  string    `json:"sha256Hex"`
	EntityType string    `json:"entityType"`
	EntityID   string    `json:"entityId"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Transcription struct {
	ID               string    `json:"id"`
	Source           string    `json:"source"`
	SourceDocumentID *string   `json:"sourceDocumentId"`
	JobID            *string   `json:"jobId"`
	Transcript       string    `json:"transcript"`
	Language         string    `json:"language"`
	WhisperModel     string    `json:"whisperModel"`
	DurationMS       *int      `json:"durationMs"`
	CreatedAt        time.Time `json:"createdAt"`
}

type JobProcessingAttempt struct {
	ID           string     `json:"id"`
	JobID        string     `json:"jobId"`
	AttemptNo    int        `json:"attemptNo"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"errorMessage"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
}

const (
	TranscriptionSourceMobileVoice       = "mobile_voice"
	TranscriptionSourceAdminVoice        = "admin_voice"
	TranscriptionSourceMobileBitrixIntent = "mobile_bitrix_intent"

	AttemptStatusStarted   = "started"
	AttemptStatusCompleted = "completed"
	AttemptStatusFailed    = "failed"

	EntityTypeTemplate          = "document_template"
	EntityTypeSourceDocument    = "source_document"
	EntityTypeGeneratedDocument = "generated_document"
)
