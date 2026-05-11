package documentjob

import "time"

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

func ValidStatuses() []Status {
	return []Status{
		StatusQueued,
		StatusRunning,
		StatusCompleted,
		StatusFailed,
		StatusCancelled,
	}
}

func (s Status) IsValid() bool {
	for _, candidate := range ValidStatuses() {
		if s == candidate {
			return true
		}
	}

	return false
}

type DeliveryChannel string

const (
	DeliveryChannelInternal DeliveryChannel = "internal"
	DeliveryChannelEmail    DeliveryChannel = "email"
	DeliveryChannelBitrix   DeliveryChannel = "bitrix"
)

func (c DeliveryChannel) IsValid() bool {
	switch c {
	case DeliveryChannelInternal, DeliveryChannelEmail, DeliveryChannelBitrix:
		return true
	default:
		return false
	}
}

type DispatchStatus string

const (
	DispatchStatusNotRequired DispatchStatus = "not_required"
	DispatchStatusPending     DispatchStatus = "pending"
	DispatchStatusSent        DispatchStatus = "sent"
	DispatchStatusFailed      DispatchStatus = "failed"
)

type SourceDocumentKind string

const (
	SourceDocumentKindVoiceRecording SourceDocumentKind = "voice_recording"
	SourceDocumentKindAttachment     SourceDocumentKind = "attachment"
)

type TaskTargetSystem string

const (
	TaskTargetBitrix24      TaskTargetSystem = "bitrix24"
	TaskTargetEmailApproval TaskTargetSystem = "email_approval"
)

type TaskCommandStatus string

const (
	TaskCommandStatusRecorded TaskCommandStatus = "recorded"
	TaskCommandStatusPending  TaskCommandStatus = "pending"
	TaskCommandStatusSent     TaskCommandStatus = "sent"
	TaskCommandStatusFailed   TaskCommandStatus = "failed"
)

type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	FileName    string    `json:"fileName"`
	MimeType    string    `json:"mimeType"`
	StorageKey  string    `json:"storageKey"`
	SizeBytes   int64     `json:"sizeBytes"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Job struct {
	ID               string          `json:"id"`
	TemplateID       string          `json:"templateId"`
	TemplateName     string          `json:"templateName"`
	SourceName       string          `json:"sourceName"`
	RequestedBy      string          `json:"requestedBy"`
	Payload          string          `json:"payload"`
	DeliveryChannel  DeliveryChannel `json:"deliveryChannel"`
	DeliveryAddress  string          `json:"deliveryAddress"`
	DispatchStatus   DispatchStatus  `json:"dispatchStatus"`
	Status           Status          `json:"status"`
	ErrorMessage     string          `json:"errorMessage"`
	ResultDocumentID *string         `json:"resultDocumentId"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	StartedAt        *time.Time      `json:"startedAt"`
	CompletedAt      *time.Time      `json:"completedAt"`
}

type GeneratedDocument struct {
	ID           string    `json:"id"`
	JobID        string    `json:"jobId"`
	TemplateID   string    `json:"templateId"`
	TemplateName string    `json:"templateName"`
	FileName     string    `json:"fileName"`
	MimeType     string    `json:"mimeType"`
	StorageKey   string    `json:"storageKey"`
	SizeBytes    int64     `json:"sizeBytes"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ProcessingEvent struct {
	ID        string    `json:"id"`
	JobID     *string   `json:"jobId"`
	Level     string    `json:"level"`
	EventType string    `json:"eventType"`
	Message   string    `json:"message"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"createdAt"`
}

type SourceDocument struct {
	ID         string             `json:"id"`
	JobID      *string            `json:"jobId"`
	TemplateID string             `json:"templateId"`
	Kind       SourceDocumentKind `json:"kind"`
	Origin     string             `json:"origin"`
	FileName   string             `json:"fileName"`
	MimeType   string             `json:"mimeType"`
	StorageKey string             `json:"storageKey"`
	SizeBytes  int64              `json:"sizeBytes"`
	CreatedAt  time.Time          `json:"createdAt"`
}

type TaskCommand struct {
	ID                string            `json:"id"`
	JobID             *string           `json:"jobId"`
	SourceDocumentID  *string           `json:"sourceDocumentId"`
	TargetSystem      TaskTargetSystem  `json:"targetSystem"`
	CommandText       string            `json:"commandText"`
	Status            TaskCommandStatus `json:"status"`
	IntegrationMode   string            `json:"integrationMode"`
	ExternalReference string            `json:"externalReference"`
	ResultMessage     string            `json:"resultMessage"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}
