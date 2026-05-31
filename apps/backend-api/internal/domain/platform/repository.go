package platform

import (
	"context"
	"time"
)

type AppSettingUpsertParams struct {
	Key         string
	Value       string
	Description string
	IsSecret    bool
	UpdatedAt   time.Time
}

type StoredFileCreateParams struct {
	ID         string
	Namespace  string
	StorageKey string
	FileName   string
	MimeType   string
	SizeBytes  int64
	SHA256Hex  string
	EntityType string
	EntityID   string
	CreatedAt  time.Time
}

type TranscriptionCreateParams struct {
	ID               string
	Source           string
	SourceDocumentID *string
	JobID            *string
	Transcript       string
	Language         string
	WhisperModel     string
	DurationMS       *int
	CreatedAt        time.Time
}

type JobAttemptCreateParams struct {
	ID        string
	JobID     string
	AttemptNo int
	Status    string
	StartedAt time.Time
}

type JobAttemptFinishParams struct {
	Status       string
	ErrorMessage string
	FinishedAt   time.Time
}

type Repository interface {
	Ping(ctx context.Context) error

	UpsertAppSetting(ctx context.Context, params AppSettingUpsertParams) error
	ListPublicAppSettings(ctx context.Context) ([]AppSetting, error)

	CreateStoredFile(ctx context.Context, params StoredFileCreateParams) error
	CreateTranscription(ctx context.Context, params TranscriptionCreateParams) error

	NextJobAttemptNo(ctx context.Context, jobID string) (int, error)
	CreateJobProcessingAttempt(ctx context.Context, params JobAttemptCreateParams) error
	FinishJobProcessingAttempt(ctx context.Context, attemptID string, params JobAttemptFinishParams) error

	RecoverStuckJobs(ctx context.Context, stuckAfter time.Duration) (int, error)
}
