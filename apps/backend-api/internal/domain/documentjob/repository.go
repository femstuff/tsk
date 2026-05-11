package documentjob

import (
	"context"
	"errors"
	"time"
)

var ErrTemplateNotFound = errors.New("document template not found")
var ErrJobNotFound = errors.New("document job not found")
var ErrGeneratedDocumentNotFound = errors.New("generated document not found")
var ErrSourceDocumentNotFound = errors.New("source document not found")
var ErrTaskCommandNotFound = errors.New("task command not found")

type TemplateCreateParams struct {
	ID          string
	Name        string
	Category    string
	Version     string
	Description string
	FileName    string
	MimeType    string
	StorageKey  string
	SizeBytes   int64
	CreatedAt   time.Time
}

type JobCreateParams struct {
	ID              string
	TemplateID      string
	TemplateName    string
	SourceName      string
	RequestedBy     string
	Payload         string
	DeliveryChannel DeliveryChannel
	DeliveryAddress string
	DispatchStatus  DispatchStatus
	Status          Status
	CreatedAt       time.Time
}

type JobStatusUpdateParams struct {
	Status           Status
	DispatchStatus   DispatchStatus
	ErrorMessage     string
	ResultDocumentID *string
	StartedAt        *time.Time
	CompletedAt      *time.Time
	UpdatedAt        time.Time
}

type GeneratedDocumentCreateParams struct {
	ID           string
	JobID        string
	TemplateID   string
	TemplateName string
	FileName     string
	MimeType     string
	StorageKey   string
	SizeBytes    int64
	CreatedAt    time.Time
}

type ProcessingEventCreateParams struct {
	ID        string
	JobID     *string
	Level     string
	EventType string
	Message   string
	Details   string
	CreatedAt time.Time
}

type SourceDocumentCreateParams struct {
	ID         string
	JobID      *string
	TemplateID string
	Kind       SourceDocumentKind
	Origin     string
	FileName   string
	MimeType   string
	StorageKey string
	SizeBytes  int64
	CreatedAt  time.Time
}

type TaskCommandCreateParams struct {
	ID                string
	JobID             *string
	SourceDocumentID  *string
	TargetSystem      TaskTargetSystem
	CommandText       string
	Status            TaskCommandStatus
	IntegrationMode   string
	ExternalReference string
	ResultMessage     string
	CreatedAt         time.Time
}

type TaskCommandStatusUpdateParams struct {
	Status            TaskCommandStatus
	IntegrationMode   string
	ExternalReference string
	ResultMessage     string
	UpdatedAt         time.Time
}

type TemplateRepository interface {
	ListTemplates(context.Context) ([]Template, error)
	GetTemplateByID(context.Context, string) (Template, error)
	CreateTemplate(context.Context, TemplateCreateParams) (Template, error)
}

type JobRepository interface {
	ListJobs(context.Context) ([]Job, error)
	GetJobByID(context.Context, string) (Job, error)
	CreateJob(context.Context, JobCreateParams) (Job, error)
	ClaimNextQueuedJob(context.Context) (Job, error)
	UpdateJobStatus(context.Context, string, JobStatusUpdateParams) (Job, error)
	CountJobsByStatus(context.Context) (map[Status]int, error)
}

type GeneratedDocumentRepository interface {
	ListGeneratedDocuments(context.Context, string) ([]GeneratedDocument, error)
	GetGeneratedDocumentByID(context.Context, string) (GeneratedDocument, error)
	CreateGeneratedDocument(context.Context, GeneratedDocumentCreateParams) (GeneratedDocument, error)
}

type ProcessingEventRepository interface {
	ListProcessingEvents(context.Context, string, int) ([]ProcessingEvent, error)
	CreateProcessingEvent(context.Context, ProcessingEventCreateParams) (ProcessingEvent, error)
}

type SourceDocumentRepository interface {
	ListSourceDocuments(context.Context, string) ([]SourceDocument, error)
	GetSourceDocumentByID(context.Context, string) (SourceDocument, error)
	CreateSourceDocument(context.Context, SourceDocumentCreateParams) (SourceDocument, error)
}

type TaskCommandRepository interface {
	ListTaskCommands(context.Context, string) ([]TaskCommand, error)
	GetTaskCommandByID(context.Context, string) (TaskCommand, error)
	CreateTaskCommand(context.Context, TaskCommandCreateParams) (TaskCommand, error)
	UpdateTaskCommandStatus(context.Context, string, TaskCommandStatusUpdateParams) (TaskCommand, error)
}
