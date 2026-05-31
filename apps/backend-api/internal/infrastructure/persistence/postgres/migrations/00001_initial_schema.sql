-- +goose Up
CREATE TABLE IF NOT EXISTS document_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS document_jobs (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES document_templates(id),
    template_name TEXT NOT NULL,
    source_name TEXT NOT NULL,
    requested_by TEXT NOT NULL,
    payload TEXT NOT NULL,
    delivery_channel TEXT NOT NULL,
    delivery_address TEXT NOT NULL,
    dispatch_status TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    result_document_id TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_document_jobs_status_created_at
    ON document_jobs (status, created_at);

CREATE TABLE IF NOT EXISTS generated_documents (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES document_jobs(id),
    template_id TEXT NOT NULL REFERENCES document_templates(id),
    template_name TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_generated_documents_job_id
    ON generated_documents (job_id);

CREATE TABLE IF NOT EXISTS source_documents (
    id TEXT PRIMARY KEY,
    job_id TEXT REFERENCES document_jobs(id),
    template_id TEXT REFERENCES document_templates(id),
    kind TEXT NOT NULL,
    origin TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_source_documents_job_id
    ON source_documents (job_id);

CREATE TABLE IF NOT EXISTS processing_events (
    id TEXT PRIMARY KEY,
    job_id TEXT REFERENCES document_jobs(id),
    level TEXT NOT NULL,
    event_type TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_processing_events_created_at
    ON processing_events (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_processing_events_job_created_at
    ON processing_events (job_id, created_at DESC);

CREATE TABLE IF NOT EXISTS task_commands (
    id TEXT PRIMARY KEY,
    job_id TEXT REFERENCES document_jobs(id),
    source_document_id TEXT REFERENCES source_documents(id),
    target_system TEXT NOT NULL,
    command_text TEXT NOT NULL,
    status TEXT NOT NULL,
    integration_mode TEXT NOT NULL,
    external_reference TEXT NOT NULL DEFAULT '',
    result_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_commands_job_id
    ON task_commands (job_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS task_commands;
DROP TABLE IF EXISTS processing_events;
DROP TABLE IF EXISTS source_documents;
DROP TABLE IF EXISTS generated_documents;
DROP TABLE IF EXISTS document_jobs;
DROP TABLE IF EXISTS document_templates;
